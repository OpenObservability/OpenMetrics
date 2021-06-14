package scrape

import (
	"errors"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
)

var (
	errMustNotCounterValueDecrease = errors.New("counter total MUST be monotonically non-decreasing over time")

	errMustNotSeriesDisappear = errors.New("series MUST NOT disappear between scrapes")

	errShouldNotDuplicateLabel = errors.New("the same label name and value SHOULD NOT appear on every Metric within a MetricSet")
)

// validator records metrics in a scrape and validates them against previous
// scrapes.
type validator interface {
	// Record records a metric.
	Record(
		m metadata,
		lset labels.Labels,
		timestamp int64,
		value float64,
	) error

	// Validate validates the recorded metrics against previous scrapes.
	Validate() error
}

type metadata struct {
	metricType textparse.MetricType
	unit       string
	help       string
}

type metricPoint struct {
	m         metadata
	lset      labels.Labels
	timestamp int64
	value     float64
}

type openMetricsValidator struct {
	lastMetricSet map[string]metricPoint
	curMetricSet  map[string]metricPoint
}

func newValidator() *openMetricsValidator {
	return &openMetricsValidator{
		curMetricSet: make(map[string]metricPoint),
	}
}

func (v *openMetricsValidator) Record(
	m metadata,
	lset labels.Labels,
	timestamp int64,
	value float64,
) error {
	key, err := labelKey(lset)
	if err != nil {
		return err
	}
	cur := metricPoint{
		m:         m,
		lset:      lset,
		value:     value,
		timestamp: timestamp,
	}
	v.curMetricSet[key] = cur
	return nil
}

func (v *openMetricsValidator) Validate() error {
	// TODO: differentiate SHOULD NOT and MUST NOT errors.
	if err := v.validateLabels(); err != nil {
		return err
	}
	for lset, lastData := range v.lastMetricSet {
		curData, ok := v.curMetricSet[lset]
		if !ok {
			return errMustNotSeriesDisappear
		}
		return validate(lastData, curData)
	}
	v.lastMetricSet = v.curMetricSet
	v.curMetricSet = make(map[string]metricPoint, len(v.lastMetricSet))
	return nil
}

// validateLabels makes sure that the same label name and value does not appear
// on every metric within a metric set.
func (v *openMetricsValidator) validateLabels() error {
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
		return errShouldNotDuplicateLabel
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
func validate(last, cur metricPoint) error {
	switch last.m.metricType {
	case textparse.MetricTypeCounter:
		return validateCounter(last, cur)
	}
	return nil
}

func validateCounter(last, cur metricPoint) error {
	if cur.value < last.value {
		return errMustNotCounterValueDecrease
	}
	return nil
}

// labelKey generates a key for the labels.
func labelKey(lset labels.Labels) (string, error) {
	return lset.String(), nil
}
