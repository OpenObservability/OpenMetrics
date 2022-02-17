package validator

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/prometheus/prometheus/pkg/exemplar"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/scrape"
	"go.uber.org/multierr"
)

var (
	errExemplar = errorWithLevel{
		err:   errors.New("only histogram/gaugehistogram buckets and counters can have exemplars"),
		level: ErrorLevelMust,
	}

	errMustNotMixTimestampPresense = errorWithLevel{
		err:   errors.New("Mix of timestamp presence within a group"),
		level: ErrorLevelMust,
	}

	errMustLabelNamesBeUnique = errorWithLevel{
		err:   errors.New("Label names MUST be unique within a LabelSet"),
		level: ErrorLevelMust,
	}

	errMetricTypeAlreadySet = errorWithLevel{
		err:   errors.New("metric type already set"),
		level: ErrorLevelMust,
	}

	errUnitAlreadySet = errorWithLevel{
		err:   errors.New("unit already set"),
		level: ErrorLevelMust,
	}

	errHelpAlreadySet = errorWithLevel{
		err:   errors.New("help already set"),
		level: ErrorLevelMust,
	}

	errMustTimestampIncrease = errorWithLevel{
		err:   errors.New("MetricPoints MUST have monotonically increasing timestamps"),
		level: ErrorLevelMust,
	}

	errMustNotMetricFamiliesInterleave = errorWithLevel{
		err:   errors.New("MetricFamilies MUST NOT be interleaved"),
		level: ErrorLevelMust,
	}

	errMustNotCounterValueDecrease = errorWithLevel{
		err:   errors.New("counter total MUST be monotonically non-decreasing over time"),
		level: ErrorLevelMust,
	}

	errMustCounterValueBeValid = errorWithLevel{
		err:   errors.New("A Total is a non-NaN and MUST be monotonically non-decreasing over time, starting from 0"),
		level: ErrorLevelMust,
	}

	errCounterValueNaN = errorWithLevel{
		err:   errors.New("counter like value must not be NaN"),
		level: ErrorLevelMust,
	}

	errCounterValueNegative = errorWithLevel{
		err:   errors.New("counter like value must not be negative"),
		level: ErrorLevelMust,
	}

	errMustContainPositiveInfBucket = errorWithLevel{
		err:   errors.New("Histogram MetricPoints MUST have at least a bucket with an +Inf threshold"),
		level: ErrorLevelMust,
	}

	errMustSummaryQuantileBeBetweenZeroAndOne = errorWithLevel{
		err:   errors.New("Quantiles MUST be between 0 and 1 inclusive"),
		level: ErrorLevelMust,
	}

	errMustNotSummaryQuantileValueBeNegative = errorWithLevel{
		err:   errors.New("Quantile values MUST NOT be negative"),
		level: ErrorLevelMust,
	}

	errInvalidSummaryCountAndSum = errorWithLevel{
		err:   errors.New("Count and Sum values are counters so MUST NOT be NaN or negative"),
		level: ErrorLevelMust,
	}

	errMustStateSetContainLabel = errorWithLevel{
		err:   errors.New("Each State's sample MUST have a label with the MetricFamily name as the label name and the State name as the label value"),
		level: ErrorLevelMust,
	}

	errMustNoUnitForStateSet = errorWithLevel{
		err:   errors.New("MetricFamilies of type StateSets MUST have an empty Unit string"),
		level: ErrorLevelMust,
	}

	errMustNoUnitForInfo = errorWithLevel{
		err:   errors.New("MetricFamilies of type Info MUST have an empty Unit string"),
		level: ErrorLevelMust,
	}

	errInvalidInfoValue = errorWithLevel{
		err:   errors.New("The Sample value MUST always be 1"),
		level: ErrorLevelMust,
	}

	errInvalidStateSetValue = errorWithLevel{
		err:   errors.New("The State sample's value MUST be 1 if the State is true and MUST be 0 if the State is false"),
		level: ErrorLevelMust,
	}

	errMustHistogramBucketsInOrder = errorWithLevel{
		err:   errors.New("histogram must have buckets in order"),
		level: ErrorLevelMust,
	}

	errMustHistogramHaveSumAndCount = errorWithLevel{
		err:   errors.New("If and only if a Sum Value is present in a MetricPoint, then the MetricPoint's +Inf Bucket value MUST also appear in a Sample with a MetricName with the suffix \"_count\""),
		level: ErrorLevelMust,
	}

	errMustHistogramNotHaveSumAndNegative = errorWithLevel{
		err:   errors.New("Cannot have _sum with negative buckets"),
		level: ErrorLevelMust,
	}

	errGaugeHistogramBucketValueNaN = errorWithLevel{
		err:   errors.New("gauge histogram bucket value must not be NaN"),
		level: ErrorLevelMust,
	}

	errGaugeHistogramBucketValueNegative = errorWithLevel{
		err:   errors.New("gauge histogram bucket value must not be negative"),
		level: ErrorLevelMust,
	}

	errGaugeHistogramGSumValueNaN = errorWithLevel{
		err:   errors.New("gauge histogram _gsum value must not be negative"),
		level: ErrorLevelMust,
	}

	errMustGaugeHistogramBucketsInOrder = errorWithLevel{
		err:   errors.New("gauge histogram must have buckets in order"),
		level: ErrorLevelMust,
	}

	errMustGaugeHistogramNotHaveGSumAndNegative = errorWithLevel{
		err:   errors.New("Cannot have negative _gsum with non-negative buckets"),
		level: ErrorLevelMust,
	}

	errMustGaugeHistogramHaveGSumAndGCountOrNeither = errorWithLevel{
		err:   errors.New("must have both _gsum and _gcount or neither"),
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

var _reservedSuffixes = map[textparse.MetricType]validSuffixes{
	textparse.MetricTypeCounter:        {suffixes: []string{"_total", "_created"}},
	textparse.MetricTypeSummary:        {suffixes: []string{"_count", "_sum", "_created"}, allowEmpty: true},
	textparse.MetricTypeHistogram:      {suffixes: []string{"_count", "_sum", "_bucket", "_created"}},
	textparse.MetricTypeGaugeHistogram: {suffixes: []string{"_gcount", "_gsum", "_bucket"}},
	textparse.MetricTypeInfo:           {suffixes: []string{"_info"}},
	textparse.MetricTypeGauge:          {allowEmpty: true},
	textparse.MetricTypeStateset:       {allowEmpty: true},
	textparse.MetricTypeUnknown:        {allowEmpty: true},
}

type validSuffixes struct {
	suffixes   []string
	allowEmpty bool
}

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
	exemplar  *exemplar.Exemplar
}

type histogramMetric struct {
	le     float64
	metric metric
}

func (m metric) String() string {
	name := m.lset.Get(labels.MetricName)
	labelsWithoutName := m.lset.Copy().WithoutLabels(labels.MetricName)
	exemplarDesc := ""
	if m.exemplar != nil {
		exemplarDesc = fmt.Sprintf(" # %s %v %v",
			m.exemplar.Labels.String(), m.exemplar.Value, m.exemplar.Ts)
	}
	return fmt.Sprintf("%s%s %v %v%s", name, labelsWithoutName, m.value,
		m.timestamp, exemplarDesc)
}

type metricFamily struct {
	metricType          *textparse.MetricType
	help                *string
	unit                *string
	metrics             map[string]metric
	orderedByAppearance []metric

	metricWithoutTimestampRecorded bool
	metricWithTimestampRecorded    bool
}

func newMetricFamily() *metricFamily {
	return &metricFamily{
		metrics: make(map[string]metric),
	}
}

// trySetDefaultMetadata sets the metadata to default values if not present.
func (mf *metricFamily) trySetDefaultMetadata() {
	if mf.metricType == nil {
		mt := textparse.MetricTypeUnknown
		mf.metricType = &mt
	}
	if mf.help == nil {
		help := ""
		mf.help = &help
	}
	if mf.unit == nil {
		unit := ""
		mf.unit = &unit
	}
}

func (mf *metricFamily) resetAfterValidate() {
	for i := range mf.orderedByAppearance {
		mf.orderedByAppearance[i] = metric{}
	}
	mf.orderedByAppearance = mf.orderedByAppearance[:0]
}

func (mf *metricFamily) MetricType() textparse.MetricType {
	if mf.metricType == nil {
		return textparse.MetricTypeUnknown
	}
	return *mf.metricType
}

type nowFn func() time.Time

// OpenMetricsValidator validates metrics against OpenMetrics spec.
type OpenMetricsValidator struct {
	level                ErrorLevel
	lastMetricSet        map[string]*metricFamily
	curMetricSet         map[string]*metricFamily
	lastMetricFamilyName string
	seenLabelSets        map[uint64]labels.Labels
	lastLabelSet         labels.Labels
	mErr                 error

	nowFn nowFn
}

// NewValidator creates an OpenMetricsValidator.
func NewValidator(level ErrorLevel) *OpenMetricsValidator {
	return &OpenMetricsValidator{
		lastMetricSet: make(map[string]*metricFamily),
		curMetricSet:  make(map[string]*metricFamily),
		seenLabelSets: make(map[uint64]labels.Labels),
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
			mfn, metricType := p.Type()
			v.recordMetricType(string(mfn), metricType, &dataPointFound, &m)
			continue
		case textparse.EntryHelp:
			mfn, helpBytes := p.Help()
			v.recordHelp(string(mfn), string(helpBytes), &dataPointFound, &m)
			continue
		case textparse.EntryUnit:
			mfn, unitBytes := p.Unit()
			v.recordUnit(string(mfn), string(unitBytes), &dataPointFound, &m)
			continue
		case textparse.EntryComment:
			continue
		default:
		}

		var (
			t             = defTime
			withTimestamp bool
		)
		_, tp, value := p.Series()
		if tp != nil {
			withTimestamp = true
			t = *tp
		}

		var lset labels.Labels
		_ = p.Metric(&lset)

		var e exemplar.Exemplar
		withExemplar := p.Exemplar(&e)

		mn := lset.Get(labels.MetricName)
		if mn == "" {
			v.mErr = multierr.Append(v.mErr, fmt.Errorf("labels must contain metric name %q", lset.String()))
			continue
		}

		var maybeExemplar *exemplar.Exemplar
		if withExemplar {
			maybeExemplar = &e
		}

		v.recordMetric(mn, lset, t, value, maybeExemplar, withTimestamp)

		// Mark that a metric data point is found.
		dataPointFound = true
	}
}

// tryResetMetadata resets the metadata if the parser finds metadata
// for a new metric.
func (v *OpenMetricsValidator) tryResetMetadata(dataPointFound *bool, m *scrape.MetricMetadata, mfn string) {
	// If new metadata is read after reading a data point, reset.
	if *dataPointFound {
		*dataPointFound = false
		*m = scrape.MetricMetadata{}
		m.Metric = mfn
		return
	}
	if m.Metric != "" && m.Metric != mfn {
		v.addMetricFamilyError(m.Metric,
			fmt.Errorf("metric name changed from %q to %q", m.Metric, mfn))
		return
	}
	m.Metric = mfn
}

func (v *OpenMetricsValidator) addOrGetMetricFamily(mfn string) *metricFamily {
	mf, ok := v.curMetricSet[mfn]
	if !ok {
		mf = newMetricFamily()
		v.curMetricSet[mfn] = mf
		v.lastMetricFamilyName = mfn
		v.lastLabelSet = nil
		v.seenLabelSets = make(map[uint64]labels.Labels)

		return mf
	}
	if v.lastMetricFamilyName != "" && mfn != v.lastMetricFamilyName {
		// When the metric family name differs from the last seen metric family name
		// and the metric family is already created, it means the metric families
		// are interleaved.
		v.addMetricFamilyError(mfn, errMustNotMetricFamiliesInterleave)
	}
	v.lastMetricFamilyName = mfn
	return mf
}

func (v *OpenMetricsValidator) recordMetricType(
	mfn string,
	mt textparse.MetricType,
	dataPointFound *bool,
	m *scrape.MetricMetadata,
) {
	mf := v.addOrGetMetricFamily(mfn)
	if mf.metricType != nil {
		v.addMetricFamilyError(mfn, errMetricTypeAlreadySet)
		return
	}
	mf.metricType = &mt
	m.Type = mt
	v.tryResetMetadata(dataPointFound, m, mfn)
}

func (v *OpenMetricsValidator) recordHelp(
	mfn string,
	help string,
	dataPointFound *bool,
	m *scrape.MetricMetadata,
) {
	mf := v.addOrGetMetricFamily(mfn)
	if mf.help != nil {
		v.addMetricFamilyError(mfn, errHelpAlreadySet)
		return
	}
	mf.help = &help
	m.Help = help
	v.tryResetMetadata(dataPointFound, m, mfn)
}

func (v *OpenMetricsValidator) recordUnit(
	mfn string,
	unit string,
	dataPointFound *bool,
	m *scrape.MetricMetadata,
) {
	mf := v.addOrGetMetricFamily(mfn)
	if mf.unit != nil {
		v.addMetricFamilyError(mfn, errUnitAlreadySet)
		return
	}
	mf.unit = &unit
	m.Unit = unit
	v.tryResetMetadata(dataPointFound, m, mfn)
}

func (v *OpenMetricsValidator) recordMetric(
	mn string,
	lset labels.Labels,
	timestamp int64,
	value float64,
	e *exemplar.Exemplar,
	withTimestamp bool,
) {
	mfn := v.sanitizedMetricName(mn)
	mf := v.addOrGetMetricFamily(mfn)
	mf.trySetDefaultMetadata()
	if withTimestamp {
		mf.metricWithTimestampRecorded = true
	} else {
		mf.metricWithoutTimestampRecorded = true
	}
	cur := metric{
		lset:      lset,
		value:     value,
		timestamp: timestamp,
		exemplar:  e,
	}
	mf.orderedByAppearance = append(mf.orderedByAppearance, cur)
	v.validateMetric(mn, mf.MetricType(), cur)

	ignoredLabels := getIgnoredLabels(mn, mfn, mf)
	hash, _ := lset.HashWithoutLabels([]byte{}, ignoredLabels...)
	_, seen := v.seenLabelSets[hash]
	if v.lastLabelSet != nil && !labels.Equal(v.lastLabelSet, lset.WithoutLabels(ignoredLabels...)) && seen {
		v.addMetricError(cur, errMustNotMetricFamiliesInterleave)
	}

	v.lastLabelSet = lset.WithoutLabels(ignoredLabels...)
	v.seenLabelSets[hash] = lset

	key := labelKey(lset)
	last, ok := mf.metrics[key]
	if !ok {
		mf.metrics[key] = cur
		return
	}
	v.compareMetric(mn, mf.MetricType(), last, cur)
}

func (v *OpenMetricsValidator) validateRecorded() {
	v.validateLabels()
	for mfn, curMF := range v.curMetricSet {
		v.validateMetricFamily(mfn, curMF)
	}
	for mfn, lastMF := range v.lastMetricSet {
		curMF, ok := v.curMetricSet[mfn]
		if ok {
			v.compareMetricFamilies(mfn, lastMF, curMF)
			continue
		}
		v.addMetricFamilyError(mfn, errShouldNotMetricsDisappear.tryReport(v.level))
	}
	for _, mf := range v.curMetricSet {
		mf.resetAfterValidate()
	}
	v.lastMetricSet = v.curMetricSet
	v.curMetricSet = make(map[string]*metricFamily, len(v.lastMetricSet))
}

// validateLabels makes sure that the same label name and value does not appear
// on every metric within a metric set.
func (v *OpenMetricsValidator) validateLabels() {
	var (
		numMetrics  int
		lset        labels.Labels
		initialized bool
	)
	for _, mf := range v.curMetricSet {
		for _, metric := range mf.metrics {
			if _, ok := metric.lset.HasDuplicateLabelNames(); ok {
				v.addMetricError(metric, errMustLabelNamesBeUnique)
			}
			numMetrics++
			if !initialized {
				lset = labels.New(metric.lset...)
				initialized = true
				continue
			}
			lset = duplicatedLabels(lset, metric.lset)
		}
	}
	if numMetrics <= 1 {
		// When there is only one metric, skip this check.
		return
	}
	if len(lset) > 0 {
		v.mErr = multierr.Append(v.mErr, errShouldNotDuplicateLabel.tryReport(v.level))
	}
}

func duplicatedLabels(this, other labels.Labels) labels.Labels {
	res := labels.New()
	for _, l := range this {
		otherValue := other.Get(l.Name)
		if l.Value == otherValue {
			res = append(res, l)
		}
	}
	return res
}

func (v *OpenMetricsValidator) compareMetricFamilies(mfn string, last, cur *metricFamily) {
	for lset, lastMF := range last.metrics {
		curMF, ok := cur.metrics[lset]
		if ok {
			v.compareMetric(mfn, cur.MetricType(), lastMF, curMF)
			continue
		}
		v.addMetricFamilyError(mfn, errShouldNotMetricsDisappear.tryReport(v.level))
	}
}

func getIgnoredLabels(name string, mfn string, cur *metricFamily) []string {
	ignored := []string{}
	ignored = append(ignored, "__name__")
	switch cur.MetricType() {
	case textparse.MetricTypeHistogram:
		fallthrough
	case textparse.MetricTypeGaugeHistogram:
		if strings.HasSuffix(name, "_bucket") {
			ignored = append(ignored, labels.BucketLabel)
		}
	case textparse.MetricTypeSummary:
		if name == mfn {
			ignored = append(ignored, "quantile")
		}
	}

	return ignored
}

func (v *OpenMetricsValidator) validateMetricFamily(mfn string, cur *metricFamily) {
	cur.trySetDefaultMetadata()
	if cur.metricWithTimestampRecorded && cur.metricWithoutTimestampRecorded {
		v.addMetricFamilyError(mfn, errMustNotMixTimestampPresense)
	}
	switch cur.MetricType() {
	case textparse.MetricTypeCounter:
		v.validateMetricFamilyCounter(cur)
	case textparse.MetricTypeGaugeHistogram:
		v.validateMetricFamilyGaugeHistogram(mfn, cur)
	case textparse.MetricTypeHistogram:
		v.validateMetricFamilyHistogram(mfn, cur)
	case textparse.MetricTypeInfo:
		v.validateMetricFamilyInfo(mfn, cur)
	case textparse.MetricTypeStateset:
		v.validateMetricFamilyStateSet(mfn, cur)
	case textparse.MetricTypeSummary:
		v.validateMetricFamilySummary(mfn, cur)
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyCounter(cur *metricFamily) {
	for _, m := range cur.metrics {
		mn := m.lset.Get(labels.MetricName)
		if strings.HasSuffix(mn, "_total") {
			if m.value < 0 || math.IsNaN(m.value) {
				v.addMetricError(m, errMustCounterValueBeValid)
			}
		}
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyInfo(mfn string, cur *metricFamily) {
	if cur.unit != nil && *cur.unit != "" {
		v.addMetricFamilyError(mfn, errMustNoUnitForInfo)
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyStateSet(mfn string, cur *metricFamily) {
	if cur.unit != nil && *cur.unit != "" {
		v.addMetricFamilyError(mfn, errMustNoUnitForStateSet)
	}
	for _, m := range cur.metrics {
		if !m.lset.Has(mfn) {
			v.addMetricError(m, errMustStateSetContainLabel)
		}
	}
}

func (v *OpenMetricsValidator) validateMetricFamilySummary(mfn string, cur *metricFamily) {
	for _, m := range cur.metrics {
		mn := m.lset.Get(labels.MetricName)
		if strings.HasSuffix(mn, "_count") ||
			strings.HasSuffix(mn, "_sum") {
			if m.value < 0 || math.IsNaN(m.value) {
				v.addMetricError(m, errInvalidSummaryCountAndSum)
			}
			continue
		}
		if strings.HasSuffix(mn, "_created") {
			continue
		}
		// Metrics with empty suffix are expected be quantiles.
		if m.value < 0 {
			v.addMetricError(m, errMustNotSummaryQuantileValueBeNegative)
		}
		strVal := m.lset.Get("quantile")
		val, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			v.addMetricError(m, fmt.Errorf("invalid quantile value %q: %v", strVal, err))
			continue
		}
		if val < 0 || val > 1 || math.IsNaN(val) {
			v.addMetricError(m, errMustSummaryQuantileBeBetweenZeroAndOne)
		}
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyHistogram(mfn string, cur *metricFamily) {
	var (
		positiveInfBucketFound bool
		negativeBucketFound    bool
		sumFound               bool
		countFound             bool
		byBucket               = make([]histogramMetric, 0, len(cur.metrics))
	)
	// Histogram MetricPoints MUST have at least a bucket with an +Inf threshold.
	for _, m := range cur.orderedByAppearance {
		mn := m.lset.Get(labels.MetricName)
		if strings.HasSuffix(mn, "_sum") {
			sumFound = true
			continue
		}
		if strings.HasSuffix(mn, "_count") {
			countFound = true
			continue
		}
		if strings.HasSuffix(mn, "_created") {
			continue
		}
		val := m.lset.Get(labels.BucketLabel)
		if val == "+Inf" {
			byBucket = append(byBucket, histogramMetric{
				le:     math.Inf(1),
				metric: m,
			})
			positiveInfBucketFound = true
			continue
		}
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			v.addMetricError(m, err)
			continue
		}
		byBucket = append(byBucket, histogramMetric{
			le:     floatVal,
			metric: m,
		})
		if floatVal < 0 {
			negativeBucketFound = true
		}
	}

	// Histogram must have increasing bucket counts since they are all counting
	// less than or equal to with the bucket.
	sorted := sort.SliceIsSorted(byBucket, func(i, j int) bool {
		return byBucket[i].le < byBucket[j].le
	})
	if !sorted {
		v.addMetricFamilyError(mfn, errMustHistogramBucketsInOrder)
	} else {
		for i := 1; i < len(byBucket); i++ {
			last, cur := byBucket[i-1], byBucket[i]
			if last.metric.value > cur.metric.value {
				v.addMetricError(cur.metric, errorWithLevel{
					err: fmt.Errorf("bucket value %v is out of order: last=%v, cur=%v",
						cur.le, last.metric.value, cur.metric.value),
					level: ErrorLevelMust,
				})
				break
			}
		}
	}

	if !positiveInfBucketFound {
		v.addMetricFamilyError(mfn, errMustContainPositiveInfBucket)
	}
	if sumFound != countFound {
		v.addMetricFamilyError(mfn, errMustHistogramHaveSumAndCount)
	}
	if sumFound && negativeBucketFound {
		v.addMetricFamilyError(mfn, errMustHistogramNotHaveSumAndNegative)
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyGaugeHistogram(mfn string, cur *metricFamily) {
	var (
		positiveInfBucketFound bool
		gsumFound              bool
		gcountFound            bool
		negativeBucketFound    bool
		negativeGSumFound      bool
		byBucket               = make([]histogramMetric, 0, len(cur.metrics))
	)
	// Histogram MetricPoints MUST have at least a bucket with an +Inf threshold
	for _, m := range cur.orderedByAppearance {
		mn := m.lset.Get(labels.MetricName)
		if strings.HasSuffix(mn, "_gsum") {
			gsumFound = true
			if m.value < 0 {
				negativeGSumFound = true
			}
			continue
		}
		if strings.HasSuffix(mn, "_gcount") {
			gcountFound = true
			continue
		}
		val := m.lset.Get(labels.BucketLabel)
		if val == "+Inf" {
			byBucket = append(byBucket, histogramMetric{
				le:     math.Inf(1),
				metric: m,
			})
			positiveInfBucketFound = true
			continue
		}
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			v.addMetricError(m, err)
			continue
		}
		byBucket = append(byBucket, histogramMetric{
			le:     floatVal,
			metric: m,
		})
		if floatVal < 0 {
			negativeBucketFound = true
		}
	}

	// Histogram must have increasing bucket counts since they are all counting
	// less than or equal to with the bucket.
	sorted := sort.SliceIsSorted(byBucket, func(i, j int) bool {
		return byBucket[i].le < byBucket[j].le
	})
	if !sorted {
		v.addMetricFamilyError(mfn, errMustGaugeHistogramBucketsInOrder)
	}

	if !positiveInfBucketFound {
		v.addMetricFamilyError(mfn, errMustContainPositiveInfBucket)
	}
	if negativeGSumFound && !negativeBucketFound {
		v.addMetricFamilyError(mfn, errMustGaugeHistogramNotHaveGSumAndNegative)
	}
	if (gsumFound && !gcountFound) || (!gsumFound && gcountFound) {
		v.addMetricFamilyError(mfn, errMustGaugeHistogramHaveGSumAndGCountOrNeither)
	}
}

func (v *OpenMetricsValidator) validateMetric(mn string, mt textparse.MetricType, cur metric) {
	switch mt {
	case textparse.MetricTypeCounter:
		if !strings.HasSuffix(mn, "_created") {
			v.validateMetricCounterValue(mn, cur)
		}
	case textparse.MetricTypeHistogram:
		if strings.HasSuffix(mn, "_count") || strings.HasSuffix(mn, "_sum") || strings.HasSuffix(mn, "_bucket") {
			v.validateMetricCounterValue(mn, cur)
		}
	case textparse.MetricTypeGaugeHistogram:
		switch {
		case strings.HasSuffix(mn, "_bucket"):
			if math.IsNaN(cur.value) {
				v.addMetricError(cur, errGaugeHistogramBucketValueNaN)
			}
			if cur.value < 0 {
				v.addMetricError(cur, errGaugeHistogramBucketValueNegative)
			}
		case strings.HasSuffix(mn, "_gsum"):
			if math.IsNaN(cur.value) {
				v.addMetricError(cur, errGaugeHistogramGSumValueNaN)
			}
		}
	case textparse.MetricTypeSummary:
		if strings.HasSuffix(mn, "_count") || strings.HasSuffix(mn, "_sum") {
			v.validateMetricCounterValue(mn, cur)
		}
	case textparse.MetricTypeInfo:
		v.validateMetricInfo(mn, cur)
	case textparse.MetricTypeStateset:
		v.validateMetricStateSet(mn, cur)
	}

	v.validateExemplar(mt, cur)
}

func (v *OpenMetricsValidator) validateExemplar(mt textparse.MetricType, cur metric) {
	if cur.exemplar == nil {
		return
	}

	// Check valid for this type at all.
	if mt != textparse.MetricTypeGaugeHistogram && mt != textparse.MetricTypeHistogram && mt != textparse.MetricTypeCounter {
		v.addMetricError(cur, errExemplar)
		return
	}

	// Check the exemplar length of the labels is valid.
	total := 0
	for _, l := range cur.exemplar.Labels {
		total += utf8.RuneCountInString(l.Name)
		total += utf8.RuneCountInString(l.Value)
	}
	if total > exemplar.ExemplarMaxLabelSetLength {
		v.addMetricError(cur, fmt.Errorf("exemplar label contents of %d exceeds maximum of %d UTF-8 characters",
			total, exemplar.ExemplarMaxLabelSetLength))
	}
}

func (v *OpenMetricsValidator) validateMetricCounterValue(mn string, cur metric) {
	if math.IsNaN(cur.value) {
		v.addMetricError(cur, errCounterValueNaN)
		return
	}

	if cur.value < 0 {
		v.addMetricError(cur, errCounterValueNegative)
	}
}

func (v *OpenMetricsValidator) validateMetricInfo(mn string, cur metric) {
	if cur.value != 1 {
		v.addMetricError(cur, errInvalidInfoValue)
	}
}

func (v *OpenMetricsValidator) validateMetricStateSet(mn string, cur metric) {
	if cur.value != 1 && cur.value != 0 {
		v.addMetricError(cur, errInvalidStateSetValue)
	}
}

// compareMetric compares the current record against last record for a metric.
// TODO: compare more metric types.
func (v *OpenMetricsValidator) compareMetric(mn string, mt textparse.MetricType, last, cur metric) {
	if cur.timestamp < last.timestamp {
		v.addMetricError(cur, errMustTimestampIncrease)
	}
	switch mt {
	case textparse.MetricTypeCounter:
		v.compareMetricCounter(mn, last, cur)
	}
}

func (v *OpenMetricsValidator) compareMetricCounter(mn string, last, cur metric) {
	if cur.value < last.value {
		v.addMetricError(cur, errMustNotCounterValueDecrease)
	}
}

func (v *OpenMetricsValidator) addMetricError(m metric, err error) {
	if err == nil {
		return
	}
	v.mErr = multierr.Append(v.mErr, fmt.Errorf("error for metric %s: %v", m.String(), err))
}

func (v *OpenMetricsValidator) addMetricFamilyError(name string, err error) {
	if err == nil {
		return
	}
	v.mErr = multierr.Append(v.mErr, fmt.Errorf("error for metric family %s: %v", name, err))
}

// labelKey generates a key for the labels.
func labelKey(lset labels.Labels) string {
	return lset.String()
}

func (v *OpenMetricsValidator) sanitizedMetricName(mn string) string {
	for _, vs := range _reservedSuffixes {
		for _, suffix := range vs.suffixes {
			if strings.HasSuffix(mn, suffix) {
				return strings.TrimSuffix(mn, suffix)
			}
		}
	}
	return mn
}

func (v *OpenMetricsValidator) isValidMetricName(mn, mfn string, mt *textparse.MetricType) bool {
	vs := _reservedSuffixes[*mt]
	for _, suffix := range vs.suffixes {
		if !strings.HasSuffix(mn, suffix) {
			continue
		}
		sanitizedMetricName := strings.TrimSuffix(mn, suffix)
		return sanitizedMetricName == mfn
	}
	if vs.allowEmpty {
		return mn == mfn
	}
	// The metric name does not match any valid suffix for the metric type.
	return false
}
