package validator

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

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

	errMustHistogramHaveSumAndCount = errorWithLevel{
		err:   errors.New("If and only if a Sum Value is present in a MetricPoint, then the MetricPoint's +Inf Bucket value MUST also appear in a Sample with a MetricName with the suffix \"_count\""),
		level: ErrorLevelMust,
	}

	errMustHistogramNotHaveSumAndNegative = errorWithLevel{
		err:   errors.New("Cannot have _sum with negative buckets"),
		level: ErrorLevelMust,
	}

	errMustGaugeHistogramNotHaveGSumAndNegative = errorWithLevel{
		err:   errors.New("Cannot have negative _gsum with non-negative buckets"),
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

var (
	_reservedSuffixes = map[textparse.MetricType]validSuffixes{
		textparse.MetricTypeCounter:        {suffixes: []string{"_total", "_created"}},
		textparse.MetricTypeSummary:        {suffixes: []string{"_count", "_sum", "_created"}, allowEmpty: true},
		textparse.MetricTypeHistogram:      {suffixes: []string{"_count", "_sum", "_bucket", "_created"}},
		textparse.MetricTypeGaugeHistogram: {suffixes: []string{"_gcount", "_gsum", "_bucket"}},
		textparse.MetricTypeInfo:           {suffixes: []string{"_info"}},
		textparse.MetricTypeGauge:          {allowEmpty: true},
		textparse.MetricTypeStateset:       {allowEmpty: true},
		textparse.MetricTypeUnknown:        {allowEmpty: true},
	}
)

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
	lset         labels.Labels
	timestamp    int64
	value        float64
	withExemplar bool
}

type metricFamily struct {
	metricType *textparse.MetricType
	help       *string
	unit       *string
	metrics    map[string]metric

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
	mErr                 error

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

		v.recordMetric(mn, lset, t, value, withExemplar, withTimestamp)
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
		v.addError(mfn, fmt.Errorf("metric name changed from %q to %q", m.Metric, mfn))
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
		return mf
	}
	if v.lastMetricFamilyName != "" && mfn != v.lastMetricFamilyName {
		// When the metric family name differs from the last seen metric family name
		// and the metric family is already created, it means the metric families
		// are interleaved.
		v.addError(mfn, errMustNotMetricFamiliesInterleave)
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
		v.addError(mfn, errMetricTypeAlreadySet)
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
		v.addError(mfn, errHelpAlreadySet)
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
		v.addError(mfn, errUnitAlreadySet)
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
	withExemplar bool,
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
		lset:         lset,
		value:        value,
		timestamp:    timestamp,
		withExemplar: withExemplar,
	}
	v.validateMetric(mn, mf.MetricType(), cur)
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
		v.addError(mfn, errShouldNotMetricsDisappear.tryReport(v.level))
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
	for mfn, mf := range v.curMetricSet {
		for _, metric := range mf.metrics {
			if _, ok := metric.lset.HasDuplicateLabelNames(); ok {
				v.addError(mfn, errMustLabelNamesBeUnique)
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
		v.addError(mfn, errShouldNotMetricsDisappear.tryReport(v.level))
	}
}

func (v *OpenMetricsValidator) validateMetricFamily(mfn string, cur *metricFamily) {
	cur.trySetDefaultMetadata()
	if cur.metricWithTimestampRecorded && cur.metricWithoutTimestampRecorded {
		v.addError(mfn, errMustNotMixTimestampPresense)
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
				v.addError(mn, errMustCounterValueBeValid)
			}
		}
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyInfo(mfn string, cur *metricFamily) {
	if cur.unit != nil && *cur.unit != "" {
		v.addError(mfn, errMustNoUnitForInfo)
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyStateSet(mfn string, cur *metricFamily) {
	if cur.unit != nil && *cur.unit != "" {
		v.addError(mfn, errMustNoUnitForStateSet)
	}
	for _, m := range cur.metrics {
		if !m.lset.Has(mfn) {
			v.addError(mfn, errMustStateSetContainLabel)
		}
	}
}

func (v *OpenMetricsValidator) validateMetricFamilySummary(mfn string, cur *metricFamily) {
	for _, m := range cur.metrics {
		mn := m.lset.Get(labels.MetricName)
		if strings.HasSuffix(mn, "_count") ||
			strings.HasSuffix(mn, "_sum") {
			if m.value < 0 || math.IsNaN(m.value) {
				v.addError(mn, errInvalidSummaryCountAndSum)
			}
			continue
		}
		if strings.HasSuffix(mn, "_created") {
			continue
		}
		// Metrics with empty suffix are expected be quantiles.
		if m.value < 0 {
			v.addError(mn, errMustNotSummaryQuantileValueBeNegative)
		}
		strVal := m.lset.Get("quantile")
		val, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			v.addError(mn, fmt.Errorf("invalid quantile value %q: %v", strVal, err))
			continue
		}
		if val < 0 || val > 1 || math.IsNaN(val) {
			v.addError(mn, errMustSummaryQuantileBeBetweenZeroAndOne)
		}
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyHistogram(mfn string, cur *metricFamily) {
	var (
		positiveInfBucketFound bool
		negativeBucketFound    bool
		sumFound               bool
		countFound             bool
	)
	// Histogram MetricPoints MUST have at least a bucket with an +Inf threshold
	for _, m := range cur.metrics {
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
			positiveInfBucketFound = true
			continue
		}
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			v.addError(mn, err)
			continue
		}
		if floatVal < 0 {
			negativeBucketFound = true
		}
	}
	if !positiveInfBucketFound {
		v.addError(mfn, errMustContainPositiveInfBucket)
	}
	if sumFound != countFound {
		v.addError(mfn, errMustHistogramHaveSumAndCount)
	}
	if sumFound && negativeBucketFound {
		v.addError(mfn, errMustHistogramNotHaveSumAndNegative)
	}
}

func (v *OpenMetricsValidator) validateMetricFamilyGaugeHistogram(mfn string, cur *metricFamily) {
	var (
		positiveInfBucketFound bool
		negativeBucketFound    bool
		negativeGSumFound      bool
	)
	// Histogram MetricPoints MUST have at least a bucket with an +Inf threshold
	for _, m := range cur.metrics {
		mn := m.lset.Get(labels.MetricName)
		if strings.HasSuffix(mn, "_gsum") {
			if m.value < 0 {
				negativeGSumFound = true
			}
			continue
		}
		if strings.HasSuffix(mn, "_gcount") {
			continue
		}
		val := m.lset.Get(labels.BucketLabel)
		if val == "+Inf" {
			positiveInfBucketFound = true
			continue
		}
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			v.addError(mn, err)
			continue
		}
		if floatVal < 0 {
			negativeBucketFound = true
		}
	}
	if !positiveInfBucketFound {
		v.addError(mfn, errMustContainPositiveInfBucket)
	}
	if negativeGSumFound && !negativeBucketFound {
		v.addError(mfn, errMustGaugeHistogramNotHaveGSumAndNegative)
	}
}

func (v *OpenMetricsValidator) validateMetric(mn string, mt textparse.MetricType, cur metric) {
	if cur.withExemplar {
		if mt != textparse.MetricTypeGaugeHistogram && mt != textparse.MetricTypeHistogram && mt != textparse.MetricTypeCounter {
			v.addError(mn, errExemplar)
		}
	}
	switch mt {
	case textparse.MetricTypeInfo:
		v.validateMetricInfo(mn, cur)
	case textparse.MetricTypeStateset:
		v.validateMetricStateSet(mn, cur)
	}
}

func (v *OpenMetricsValidator) validateMetricInfo(mn string, cur metric) {
	if cur.value != 1 {
		v.addError(mn, errInvalidInfoValue)
	}
}

func (v *OpenMetricsValidator) validateMetricStateSet(mn string, cur metric) {
	if cur.value != 1 && cur.value != 0 {
		v.addError(mn, errInvalidStateSetValue)
	}
}

// compareMetric compares the current record against last record for a metric.
// TODO: compare more metric types.
func (v *OpenMetricsValidator) compareMetric(mn string, mt textparse.MetricType, last, cur metric) {
	if cur.timestamp < last.timestamp {
		v.addError(mn, errMustTimestampIncrease)
	}
	switch mt {
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
	if err == nil {
		return
	}
	v.mErr = multierr.Append(v.mErr, fmt.Errorf("error found on metric %q: %v", mn, err))
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
