package validator

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

func TestValidateShouldAndMust(t *testing.T) {
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
			name: "bad_must_not_metric_name_change",
			exports: []string{
				`# TYPE a counter
# HELP b help
a_total1 2
# EOF`,
			},
			expectedErr: errors.New(`metric name changed from "a" to "b"`),
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
			expectedErr: errMustQuantileBeBetweenZeroAndOne,
		},
		{
			name: "bad_must_summary_quantile_be_between_0_and_1",
			exports: []string{
				`# TYPE a summary
a{quantile="NaN"} 0
# EOF`,
			},
			expectedErr: errMustQuantileBeBetweenZeroAndOne,
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
			name: "bad_clashing_names",
			exports: []string{
				`# TYPE a counter
# TYPE a counter
# EOF
`,
			},
			expectedErr: errMustMetricNameBeUnique,
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
		{
			name: "bad_must_not_metric_name_change",
			exports: []string{
				`# TYPE a counter
# HELP b help
a_total1 2
# EOF`,
			},
			expectedErr: errors.New(`metric name changed from "a" to "b"`),
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
