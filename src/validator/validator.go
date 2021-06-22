package validator

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/scrape"
	"go.uber.org/multierr"
)

var (
	errMustNotCounterValueDecrease = errorWithLevel{
		err:   errors.New("counter total MUST be monotonically non-decreasing over time"),
		level: ErrorLevelMust,
	}

	errMustTimestampIncrease = errorWithLevel{
		err:   errors.New("MetricPoints MUST have monotonically increasing timestamps"),
		level: ErrorLevelMust,
	}

	errMustContainPositiveInfBucket = errorWithLevel{
		err:   errors.New("Histogram MetricPoints MUST have at least a bucket with an +Inf threshold"),
		level: ErrorLevelMust,
	}

	errMustQuantileBeBetweenZeroAndOne = errorWithLevel{
		err:   errors.New("Quantiles MUST be between 0 and 1 inclusive"),
		level: ErrorLevelMust,
	}

	errMustStateSetContainLabel = errorWithLevel{
		err:   errors.New("Each State's sample MUST have a label with the MetricFamily name as the label name and the State name as the label value"),
		level: ErrorLevelMust,
	}

	errMustMetricNameBeUnique = errorWithLevel{
		err:   errors.New("MetricFamily names are a string and MUST be unique within a MetricSet"),
		level: ErrorLevelMust,
	}

	errMustMetricFamilyWithMetadata = errorWithLevel{
		err:   errors.New("A MetricFamily MUST have a name, HELP, TYPE, and UNIT metadata"),
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

type metric struct {
	lset      labels.Labels
	timestamp int64
	value     float64
}

type metricFamily struct {
	metricType *textparse.MetricType
	metrics    map[string]metric
}

func newMetricFamily() *metricFamily {
	return &metricFamily{
		metrics: make(map[string]metric),
	}
}

type nowFn func() time.Time

// OpenMetricsValidator validates metrics against OpenMetrics spec.
type OpenMetricsValidator struct {
	level         ErrorLevel
	lastMetricSet map[string]*metricFamily
	curMetricSet  map[string]*metricFamily
	mErr          error

	nowFn nowFn
}

// NewValidator creates an OpenMetricsValidator.
func NewValidator(level ErrorLevel) *OpenMetricsValidator {
	return &OpenMetricsValidator{
		lastMetricSet: make(map[string]*metricFamily),
		curMetricSet:  make(map[string]*metricFamily),
		level:         level,
		nowFn:         time.Now,
	}
}

// Reset resets the validator.
func (v *OpenMetricsValidator) Reset() {
	v.lastMetricSet = make(map[string]*metricFamily)
	v.curMetricSet = make(map[string]*metricFamily)
	v.mErr = nil
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
		if err == io.EOF {
			// Validate at the end of a scrape.
			v.validateRecorded()
			return v.mErr
		}
		if err != nil {
			return multierr.Append(v.mErr, err)
		}
		switch et {
		case textparse.EntryType:
			name, metricType := p.Type()
			v.recordMetricType(string(name), metricType)
			v.tryResetMetadata(&dataPointFound, &m, string(name))
			m.Type = metricType
			continue
		case textparse.EntryHelp:
			name, helpBytes := p.Help()
			v.tryResetMetadata(&dataPointFound, &m, string(name))
			m.Help = string(helpBytes)
			continue
		case textparse.EntryUnit:
			name, unitBytes := p.Unit()
			v.tryResetMetadata(&dataPointFound, &m, string(name))
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
		v.recordMetric(m.Metric, lset, t, value)
		// Mark that a metric data point is found.
		dataPointFound = true
	}
}

// tryResetMetadata resets the metadata if the parser finds metadata
// for a new metric.
func (v *OpenMetricsValidator) tryResetMetadata(dataPointFound *bool, m *scrape.MetricMetadata, mn string) {
	// If new metadata is read after reading a data point, reset.
	if *dataPointFound {
		*dataPointFound = false
		*m = scrape.MetricMetadata{}
		m.Metric = mn
		return
	}
	if m.Metric != "" && m.Metric != mn {
		v.addError(mn, fmt.Errorf("metric name changed from %q to %q", m.Metric, mn))
		return
	}
	m.Metric = mn
}

func (v *OpenMetricsValidator) recordMetricType(
	mn string,
	mt textparse.MetricType,
) {
	mf, ok := v.curMetricSet[mn]
	if ok {
		if mf.metricType != nil {
			v.addError(mn, errMustMetricNameBeUnique)
			return
		}
		mf.metricType = &mt
		return
	}
	mf = newMetricFamily()
	mf.metricType = &mt
	v.curMetricSet[mn] = mf
	return
}

func (v *OpenMetricsValidator) recordMetric(
	mn string,
	lset labels.Labels,
	timestamp int64,
	value float64,
) {
	mf := v.curMetricSet[mn]
	cur := metric{
		lset:      lset,
		value:     value,
		timestamp: timestamp,
	}
	key := labelKey(lset)
	last, ok := mf.metrics[key]
	if !ok {
		mf.metrics[key] = cur
		return
	}
	v.compareMetric(mn, mf.metricType, last, cur)
}

func (v *OpenMetricsValidator) validateRecorded() {
	v.validateLabels()
	for mn, curMF := range v.curMetricSet {
		v.validateMetricFamily(mn, curMF)
	}
	for mn, lastMF := range v.lastMetricSet {
		curMF, ok := v.curMetricSet[mn]
		if ok {
			v.compareMetricFamilies(mn, lastMF, curMF)
			continue
		}
		v.addError(mn, errShouldNotMetricsDisappear.tryReport(v.level))
	}
	v.lastMetricSet = v.curMetricSet
	v.curMetricSet = make(map[string]*metricFamily, len(v.lastMetricSet))
}

// validateLabels makes sure that the same label name and value does not appear
// on every metric within a metric set.
func (v *OpenMetricsValidator) validateLabels() {
	if len(v.curMetricSet) <= 1 {
		// When there is only one metric, skip this check.
		return
	}
	var lset labels.Labels
	for _, mf := range v.curMetricSet {
		for _, metric := range mf.metrics {
			if len(lset) == 0 {
				lset = labels.New(metric.lset...)
				continue
			}
			lset = duplicatedLabels(lset, metric.lset)
		}
	}
	if len(lset) > 0 {
		v.mErr = multierr.Append(v.mErr, errShouldNotDuplicateLabel.tryReport(v.level))
	}
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

func (v *OpenMetricsValidator) compareMetricFamilies(mn string, last, cur *metricFamily) {
	for lset, lastMF := range last.metrics {
		curMF, ok := cur.metrics[lset]
		if ok {
			v.compareMetric(mn, cur.metricType, lastMF, curMF)
			continue
		}
		v.addError(mn, errShouldNotMetricsDisappear.tryReport(v.level))
	}
}

func (v *OpenMetricsValidator) validateMetricFamily(mn string, cur *metricFamily) {
	switch *cur.metricType {
	case textparse.MetricTypeHistogram, textparse.MetricTypeGaugeHistogram:
		v.validateMetricFamilyHistogram(mn, cur)
	case textparse.MetricTypeSummary:
		v.validateMetricFamilySummary(mn, cur)
	case textparse.MetricTypeStateset:
		v.validateMetricFamilyStateSet(mn, cur)
	default:
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyStateSet(mn string, cur *metricFamily) {
	for _, m := range cur.metrics {
		if len(m.lset) < 2 {
			v.addError(mn, errMustStateSetContainLabel)
		}
	}
}

func (v *OpenMetricsValidator) validateMetricFamilySummary(mn string, cur *metricFamily) {
	for _, m := range cur.metrics {
		mn := m.lset.Get(labels.MetricName)
		if strings.HasSuffix(mn, "_count") ||
			strings.HasSuffix(mn, "_sum") ||
			strings.HasSuffix(mn, "_created") {
			continue
		}
		// Metrics with empty suffix are expected to have quantile label.
		strVal := m.lset.Get("quantile")
		val, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			v.addError(mn, fmt.Errorf("invalid quantile value %q: %v", strVal, err))
			continue
		}
		if val < 0 || val > 1 || math.IsNaN(val) {
			v.addError(mn, errMustQuantileBeBetweenZeroAndOne)
		}
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyHistogram(mn string, cur *metricFamily) {
	var positiveInfBucketFound bool
	// Histogram MetricPoints MUST have at least a bucket with an +Inf threshold
	for _, m := range cur.metrics {
		val := m.lset.Get(labels.BucketLabel)
		if val == "+Inf" {
			positiveInfBucketFound = true
		}
	}
	if !positiveInfBucketFound {
		v.addError(mn, errMustContainPositiveInfBucket)
	}
}

// compareMetric validates the current record against last record for a metric.
// TODO: compareMetric more metric types.
func (v *OpenMetricsValidator) compareMetric(mn string, mt *textparse.MetricType, last, cur metric) {
	if mt == nil {
		v.addError(mn, errMustMetricFamilyWithMetadata)
	}
	if cur.timestamp <= last.timestamp {
		v.addError(mn, errMustTimestampIncrease)
	}
	switch *mt {
	case textparse.MetricTypeCounter:
		v.compareMetricCounter(mn, last, cur)
	}
}

func (v *OpenMetricsValidator) compareMetricCounter(mn string, last, cur metric) {
	if cur.value < last.value {
		v.addError(mn, errMustNotCounterValueDecrease)
	}
}

func (v *OpenMetricsValidator) addError(mn string, err error) {
	if err != nil {
		err = fmt.Errorf("error found on metric %q: %v", mn, err)
	}
	v.mErr = multierr.Append(v.mErr, err)
}

// labelKey generates a key for the labels.
func labelKey(lset labels.Labels) string {
	return lset.String()
}
