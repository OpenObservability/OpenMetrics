package validator

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/scrape"
)

var (
	errMustNotCounterValueDecrease = errorWithLevel{
		err:   errors.New("counter total MUST be monotonically non-decreasing over time"),
		level: ErrorLevelMust,
	}

	errMustNotTimestampDecrease = errorWithLevel{
		err:   errors.New("MetricPoints MUST have monotonically increasing timestamps"),
		level: ErrorLevelMust,
	}

	errShouldNotMetricsDisappear = errorWithLevel{
		err:   errors.New("metrics and samples SHOULD NOT appear and disappear from exposition to exposition"),
		level: ErrorLevelShould,
	}
	errShouldNotDuplicateLabel = errorWithLevel{
		err:   errors.New("the same label name and value SHOULD NOT appear on every Metric within a MetricSet"),
		level: ErrorLevelShould,
	}
)

// ErrorLevel is the level of the validation error.
// The OpenMetrics spec defines rules in different categories like "SHOULD"
// and "MUST", the value of ErrorLevel identifies which category is the error
// falling into.
type ErrorLevel int

// A list of supported error levels, ordered by severity.
const (
	ErrorLevelShould ErrorLevel = iota
	ErrorLevelMust
)

var validErrorLevels = []ErrorLevel{ErrorLevelShould, ErrorLevelMust}

// String returns a readable value for the error level.
// Use custom string value here because the standard string `ErrorLevelMust` is not ergnonomic in tooling.
func (el ErrorLevel) String() string {
	if el == ErrorLevelShould {
		return "should"
	}
	if el == ErrorLevelMust {
		return "must"
	}
	return ""
}

// NewErrorLevel creates an ErrorLevel.
func NewErrorLevel(str string) (ErrorLevel, error) {
	for _, el := range validErrorLevels {
		if el.String() == str {
			return el, nil
		}
	}
	return 0, fmt.Errorf("unknown error level %q", str)
}

type errorWithLevel struct {
	err   error
	level ErrorLevel
}

func (e errorWithLevel) Error() string {
	return e.err.Error()
}

// tryReport reports the error if the level is equal or above the target level, otherwise the error is omitted.
func (e errorWithLevel) tryReport(level ErrorLevel) error {
	if e.level >= level {
		return e
	}
	return nil
}

type metricDataPoint struct {
	m         scrape.MetricMetadata
	lset      labels.Labels
	timestamp int64
	value     float64
}

type nowFn func() time.Time

// OpenMetricsValidator validates metrics against OpenMetrics spec.
type OpenMetricsValidator struct {
	lastMetricSet map[string]metricDataPoint
	curMetricSet  map[string]metricDataPoint
	level         ErrorLevel

	nowFn nowFn
}

// NewValidator creates an OpenMetricsValidator.
func NewValidator(level ErrorLevel) *OpenMetricsValidator {
	return &OpenMetricsValidator{
		curMetricSet: make(map[string]metricDataPoint),
		level:        level,
		nowFn:        time.Now,
	}
}

// Validate parses the bytes and validates the metrics against OpenMetrics spec.
func (v *OpenMetricsValidator) Validate(b []byte) error {
	var (
		p              = textparse.NewOpenMetricsParser(b)
		defTime        = timestamp.FromTime(v.nowFn())
		m              scrape.MetricMetadata
		dataPointFound bool
	)
	for {
		// TODO: Handle exemplar.
		et, err := p.Next()
		if err != nil {
			if err == io.EOF {
				// Validate at the end of a scrape.
				return v.validateRecorded()
			}
			return err
		}
		switch et {
		case textparse.EntryType:
			name, metricType := p.Type()
			if err := tryResetMetadata(&dataPointFound, &m, string(name)); err != nil {
				return err
			}
			m.Type = metricType
			continue
		case textparse.EntryHelp:
			name, helpBytes := p.Help()
			if err := tryResetMetadata(&dataPointFound, &m, string(name)); err != nil {
				return err
			}
			m.Help = string(helpBytes)
			continue
		case textparse.EntryUnit:
			name, unitBytes := p.Unit()
			if err := tryResetMetadata(&dataPointFound, &m, string(name)); err != nil {
				return err
			}
			m.Unit = string(unitBytes)
			continue
		case textparse.EntryComment:
			continue
		default:
		}

		t := defTime
		_, tp, value := p.Series()
		if tp != nil {
			t = *tp
		}

		var lset labels.Labels
		_ = p.Metric(&lset)

		if !lset.Has(labels.MetricName) {
			return errors.New("metric must contain a name")
		}
		if err := v.record(m, lset, t, value); err != nil {
			return err
		}
		// Mark that a metric data point is found.
		dataPointFound = true
	}
}

// tryResetMetadata resets the metadata if the parser finds metadata
// for a new metric.
func tryResetMetadata(dataPointFound *bool, m *scrape.MetricMetadata, name string) error {
	// If new metadata is read after reading a data point, reset.
	if *dataPointFound {
		*dataPointFound = false
		*m = scrape.MetricMetadata{}
		m.Metric = name
		return nil
	}
	if m.Metric != "" && m.Metric != name {
		return fmt.Errorf("metric name changed from %q to %q", m.Metric, name)
	}
	m.Metric = name
	return nil
}

func (v *OpenMetricsValidator) record(
	m scrape.MetricMetadata,
	lset labels.Labels,
	timestamp int64,
	value float64,
) error {
	key := labelKey(lset)
	cur := metricDataPoint{
		m:         m,
		lset:      lset,
		value:     value,
		timestamp: timestamp,
	}
	last, ok := v.curMetricSet[key]
	if !ok {
		v.curMetricSet[key] = cur
		return nil
	}
	return validate(last, cur)
}

func (v *OpenMetricsValidator) validateRecorded() error {
	if err := v.validateLabels(); err != nil {
		return err
	}
	for lset, lastData := range v.lastMetricSet {
		curData, ok := v.curMetricSet[lset]
		if ok {
			return validate(lastData, curData)
		}
		if err := errShouldNotMetricsDisappear.tryReport(v.level); err != nil {
			return err
		}
	}
	v.lastMetricSet = v.curMetricSet
	v.curMetricSet = make(map[string]metricDataPoint, len(v.lastMetricSet))
	return nil
}

// validateLabels makes sure that the same label name and value does not appear
// on every metric within a metric set.
func (v *OpenMetricsValidator) validateLabels() error {
	if len(v.curMetricSet) <= 1 {
		// When there is only one metric, skip this check.
		return nil
	}
	var lset labels.Labels
	for _, data := range v.curMetricSet {
		if len(lset) == 0 {
			lset = labels.New(data.lset...)
			continue
		}
		lset = duplicatedLabels(lset, data.lset)
	}
	if len(lset) > 0 {
		return errShouldNotDuplicateLabel.tryReport(v.level)
	}
	return nil
}

func duplicatedLabels(this, other labels.Labels) labels.Labels {
	res := labels.New(this...)
	for _, l := range other {
		v := res.Get(l.Name)
		if v != l.Value {
			res = res.WithoutLabels(l.Name)
		}
	}
	return res
}

// validate validates the current record against last record for a metric.
// TODO: validate more metric types.
func validate(last, cur metricDataPoint) error {
	switch last.m.Type {
	case textparse.MetricTypeCounter:
		return validateCounter(last, cur)
	}
	return nil
}

func validateCounter(last, cur metricDataPoint) error {
	if cur.value < last.value {
		return errMustNotCounterValueDecrease
	}
	if cur.timestamp <= last.timestamp {
		return errMustNotTimestampDecrease
	}
	return nil
}

// labelKey generates a key for the labels.
func labelKey(lset labels.Labels) string {
	return lset.String()
}
