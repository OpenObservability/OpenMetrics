package validator

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

func TestValidateCaptureMultipleError(t *testing.T) {
	str := `# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 3 2
a_total{a="1",foo="bar"} 2 1
# EOF`
	v := NewValidator(ErrorLevelMust)
	err := v.Validate([]byte(str))
	require.Contains(t, err.Error(), errMustTimestampIncrease.Error())
	require.Contains(t, err.Error(), errMustNotCounterValueDecrease.Error())
}

func TestValidateShouldAndMust(t *testing.T) {
	tcs := []testCase{
		{
			name: "good_exemplar_in_counter",
			exports: []string{
				`# TYPE a counter
a_total 1 # {a="b"} 0.5
# EOF`,
			},
		},
		{
			name: "bad_exemplar_timestamp",
			exports: []string{
				`# TYPE a counter
a_total 1 # {a="b"} 0.5 NaN
# EOF`,
			},
			expectedErr: errors.New("invalid exemplar timestamp"),
		},
		{
			name: "bad_exemplar_in_gauge",
			exports: []string{
				`# TYPE a_bucket gauge
a_bucket 1 # {a="b"} 0.5
# EOF`,
			},
			expectedErr: errExemplar,
		},
		{
			name: "bad_mix_timestamp_presence",
			exports: []string{
				`# TYPE a gauge
a 0 0
a 0
# EOF`,
			},
			expectedErr: errMustNotMixTimestampPresense,
		},
		{
			name: "bad_mix_timestamp_presence",
			exports: []string{
				`# TYPE a gauge
a 0
a 0 0
# EOF`,
			},
			expectedErr: errMustNotMixTimestampPresense,
		},
		{
			name: "bad_invalid_gauge_histogram_buckets_missing_count",
			exports: []string{
				`# TYPE a gaugehistogram
a_bucket{le="+Inf"} 1
a_gsum -1
a_gcount 1
# EOF`,
			},
			expectedErr: errMustGaugeHistogramNotHaveGSumAndNegative,
		},
		{
			name: "bad_invalid_histogram_buckets_missing_count",
			exports: []string{
				`# TYPE a histogram
a_bucket{le="-1"} 0
a_bucket{le="+Inf"} 0
a_sum 0
a_count 0
# EOF`,
			},
			expectedErr: errMustHistogramNotHaveSumAndNegative,
		},
		{
			name: "bad_invalid_histogram_buckets_missing_count",
			exports: []string{
				`# TYPE a histogram
a_bucket{le="+Inf"} 0
a_sum 0
# EOF`,
			},
			expectedErr: errMustHistogramHaveSumAndCount,
		},
		{
			name: "bad_invalid_info_value",
			exports: []string{
				`# TYPE a info
a 2.0
# EOF`,
			},
			expectedErr: errInvalidInfoValue,
		},
		{
			name: "bad_invalid_stateset_value",
			exports: []string{
				`# TYPE a stateset
a{a="b"} 2.0
# EOF`,
			},
			expectedErr: errInvalidStateSetValue,
		},
		{
			name: "bad_must_label_names_be_unique",
			exports: []string{
				`a{a="1",a="1"} 1
# EOF`,
			},
			expectedErr: errMustLabelNamesBeUnique,
		},
		{
			name: "bad_must_not_counter_decreasing",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total 2
# EOF`,
				`# TYPE a counter
# HELP a help
a_total 1
# EOF`,
			},
			expectedErr: errMustNotCounterValueDecrease,
		},
		{
			name: "good_counter_increasing",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total 1
# EOF`,
				`# TYPE a counter
# HELP a help
a_total 2
# EOF`,
			},
		},
		{
			name: "bad_should_not_metric_disappearing",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total 1
# EOF`,
				`# TYPE b counter
# HELP b help
b_total 2
# EOF`,
			},
			expectedErr: errShouldNotMetricsDisappear,
		},
		{
			name: "good_not_duplicate_labels",
			exports: []string{
				`# TYPE a1 counter
# HELP a1 help
a1_total{bar="baz1"} 1
# TYPE a2 counter
# HELP a2 help
a2_total{bar="baz2"} 1
# EOF`,
			},
		},
		{
			name: "bad_should_not_duplicate_labels",
			exports: []string{
				`# TYPE a1 counter
# HELP a1 help
a1_total{bar="baz"} 1
# TYPE a2 counter
# HELP a2 help
a2_total{bar="baz"} 1
# EOF`,
			},
			expectedErr: errShouldNotDuplicateLabel,
		},
		{
			name: "good_timestamp_monotonically_increasing_1",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 1 1
a_total{a="1",foo="bar"} 2 2
# EOF`,
			},
		},
		{
			name: "good_timestamp_monotonically_increasing_2",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 1 1
a_total{a="1",foo="bar"} 2 1
# EOF`,
			},
		},
		{
			name: "bad_must_not_timestamp_decrease_in_metric_set",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 1 2
a_total{a="1",foo="bar"} 2 1
# EOF`,
			},
			expectedErr: errMustTimestampIncrease,
		},
		{
			name: "bad_must_not_timestamp_decrease_between_metric_sets",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 1 2
# EOF`,
				`# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 2 1
# EOF`,
			},
			expectedErr: errMustTimestampIncrease,
		},
		{
			name: "bad_must_histogram_have_+Inf_bucket",
			exports: []string{
				`# TYPE a gaugehistogram
a_bucket{le="10"} NaN
# EOF`,
			},
			expectedErr: errMustContainPositiveInfBucket,
		},
		{
			name: "bad_must_summary_metric_with_empty_suffix_have_quantile_label",
			exports: []string{
				`# TYPE a summary
a 0
# EOF`,
			},
			expectedErr: errors.New(`invalid quantile value ""`),
		},
		{
			name: "bad_must_summary_quantile_be_between_0_and_1",
			exports: []string{
				`# TYPE a summary
a{quantile="2"} 0
# EOF`,
			},
			expectedErr: errMustSummaryQuantileBeBetweenZeroAndOne,
		},
		{
			name: "bad_must_summary_quantile_be_between_0_and_1",
			exports: []string{
				`# TYPE a summary
a{quantile="NaN"} 0
# EOF`,
			},
			expectedErr: errMustSummaryQuantileBeBetweenZeroAndOne,
		},
		{
			name: "good_must_stateset_contain_label",
			exports: []string{
				`# TYPE a stateset
# HELP a help
a{a="b"} 0
# EOF`,
			},
		},
		{
			name: "bad_must_stateset_contain_label",
			exports: []string{
				`# TYPE a stateset
# HELP a help
a 0
# EOF`,
			},
			expectedErr: errMustStateSetContainLabel,
		},
		{
			name: "good_must_stateset_contain_label",
			exports: []string{
				`# TYPE a stateset
# HELP a help
a{a="bar"} 0
# EOF`,
			},
		},
		{
			name: "bad_duplicated_metric_type",
			exports: []string{
				`# TYPE a counter
# TYPE a counter
# EOF`,
			},
			expectedErr: errMetricTypeAlreadySet,
		},
		{
			name: "bad_duplicated_help",
			exports: []string{
				`# HELP a help
# HELP a help
# EOF`,
			},
			expectedErr: errHelpAlreadySet,
		},
		{
			name: "bad_duplicated_unit",
			exports: []string{
				`# UNIT cc_seconds seconds
# UNIT cc_seconds seconds
# EOF`,
			},
			expectedErr: errUnitAlreadySet,
		},
		{
			name: "bad_must_not_counter_total_be_nan",
			exports: []string{
				`# TYPE a counter
a_total NaN
# EOF`,
			},
			expectedErr: errMustCounterValueBeValid,
		},
		{
			name: "bad_must_not_counter_total_be_negative",
			exports: []string{
				`# TYPE a counter
a_total -1
# EOF`,
			},
			expectedErr: errMustCounterValueBeValid,
		},
		{
			name: "bad_must_not_summary_sum_be_nan",
			exports: []string{
				`# TYPE a summary
a_sum NaN
# EOF`,
			},
			expectedErr: errInvalidSummaryCountAndSum,
		},
		{
			name: "bad_must_not_summary_count_be_negative",
			exports: []string{
				`# TYPE a summary
a_count -1
# EOF`,
			},
			expectedErr: errInvalidSummaryCountAndSum,
		},
		{
			name: "bad_must_not_summary_quantile_value_be_negative",
			exports: []string{
				`# TYPE a summary
a{quantile="0.5"} -1
# EOF`,
			},
			expectedErr: errMustNotSummaryQuantileValueBeNegative,
		},
		{
			name: "good_metric_name_without_metadata",
			exports: []string{
				`a 0
b 0
# EOF`,
			},
		},
		{
			name: "bad_metric_families_interleaved",
			exports: []string{
				`# TYPE a summary
quantile{quantile="0"} 0
a_sum{a="1"} 0
quantile{quantile="1"} 0
# EOF`,
			},
			expectedErr: errMustNotMetricFamiliesInterleave,
		},
		{
			name: "bad_unit_for_info",
			exports: []string{
				`# TYPE x_u info
# UNIT x_u u
# EOF`,
			},
			expectedErr: errMustNoUnitForInfo,
		},
		{
			name: "bad_unit_for_stateset",
			exports: []string{
				`# TYPE x_u stateset
# UNIT x_u u
# EOF`,
			},
			expectedErr: errMustNoUnitForStateSet,
		},
		{
			name: "bad_metadata_in_wrong_place",
			exports: []string{
				`# TYPE a_s gauge
a_s 1
# UNIT a_s s
# EOF`,
			},
			expectedErr: errUnitAlreadySet,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Validates against both SHOULD rules and MUST rules.
			v := testValidator(ErrorLevelShould)
			run(t, v, tc)
		})
	}
}

func TestValidateMustOnly(t *testing.T) {
	tcs := []testCase{
		{
			name: "bad_must_not_counter_decreasing",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total 2
# EOF`,
				`# TYPE a counter
# HELP a help
a_total 1
# EOF`,
			},
			expectedErr: errMustNotCounterValueDecrease,
		},
		{
			name: "good_counter_increasing",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total 1
# EOF`,
				`# TYPE a counter
# HELP a help
a_total 2
# EOF`,
			},
		},
		{
			name: "bad_should_not_metric_disappearing",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total 1
# EOF`,
				`# TYPE b counter
# HELP b help
b_total 2
# EOF`,
			},
		},
		{
			name: "good_not_duplicate_labels",
			exports: []string{
				`# TYPE a1 counter
# HELP a1 help
a1_total{bar="baz1"} 1
# TYPE a2 counter
# HELP a2 help
a2_total{bar="baz2"} 1
# EOF`,
			},
		},
		{
			name: "bad_should_not_duplicate_labels",
			exports: []string{
				`# TYPE a1 counter
# HELP a1 help
a1_total{bar="baz"} 1
# TYPE a2 counter
# HELP a2 help
a2_total{bar="baz"} 1
# EOF`,
			},
		},
		{
			name: "bad_must_not_timestamp_decrease_in_metric_set",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 1 2
a_total{a="1",foo="bar"} 2 1
# EOF`,
			},
			expectedErr: errMustTimestampIncrease,
		},
		{
			name: "bad_must_not_timestamp_decrease_between_metric_sets",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 1 2
# EOF`,
				`# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 2 1
# EOF`,
			},
			expectedErr: errMustTimestampIncrease,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Validate against must rules only.
			v := testValidator(ErrorLevelMust)
			run(t, v, tc)
		})
	}
}

type testCase struct {
	name        string
	exports     []string
	expectedErr error
}

func run(t *testing.T, v *OpenMetricsValidator, tc testCase) {
	var mErr error
	for _, export := range tc.exports {
		err := v.Validate([]byte(export))
		mErr = multierr.Append(mErr, err)
	}
	if tc.expectedErr == nil {
		require.NoError(t, mErr)
		return
	}
	require.Error(t, mErr)
	require.Contains(t, mErr.Error(), tc.expectedErr.Error())
}

func testNowFn() nowFn {
	var sec int64
	return func() time.Time {
		sec++
		return time.Unix(sec, 0)
	}
}

func testValidator(el ErrorLevel) *OpenMetricsValidator {
	v := NewValidator(el)
	v.nowFn = testNowFn()
	return v
}
