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

type record struct {
	m         metadata
	lset      labels.Labels
	timestamp int64
	value     float64
}

type openMetricsValidator struct {
	lastScrape map[string]record
	curScrape  map[string]record
}

func newValidator() *openMetricsValidator {
	return &openMetricsValidator{
		curScrape: make(map[string]record),
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
	cur := record{
		m:         m,
		lset:      lset,
		value:     value,
		timestamp: timestamp,
	}
	v.curScrape[key] = cur
	return nil
}

func (v *openMetricsValidator) Validate() error {
	// TODO: differentiate SHOULD NOT and MUST NOT errors.
	if err := v.validateLabels(); err != nil {
		return err
	}
	for lset, lastData := range v.lastScrape {
		curData, ok := v.curScrape[lset]
		if !ok {
			return errMustNotSeriesDisappear
		}
		return validate(lastData, curData)
	}
	v.lastScrape = v.curScrape
	v.curScrape = make(map[string]record, len(v.lastScrape))
	return nil
}

// validateLabels makes sure that the same label name and value does not appear
// on every metric within a metric set.
func (v *openMetricsValidator) validateLabels() error {
	if len(v.curScrape) <= 1 {
		// When there is only one metric, skip this check.
		return nil
	}
	var lset labels.Labels
	for _, data := range v.curScrape {
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
func validate(last, cur record) error {
	switch last.m.metricType {
	case textparse.MetricTypeCounter:
		return validateCounter(last, cur)
	}
	return nil
}

func validateCounter(last, cur record) error {
	if cur.value < last.value {
		return errMustNotCounterValueDecrease
	}
	return nil
}

// labelKey generates a key for the labels.
func labelKey(lset labels.Labels) (string, error) {
	b, err := lset.MarshalJSON()
	return string(b), err
}
