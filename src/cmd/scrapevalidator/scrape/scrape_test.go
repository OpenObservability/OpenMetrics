package scrape

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

func TestValidate(t *testing.T) {
	tcs := []testCase{
		{
			name: "bad_counter_decreasing",
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
			name: "bad_metric_disappearing",
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
			expectedErr: errMustNotSeriesDisappear,
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
			name: "bad_duplicate_labels",
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
			name: "bad_timestamp_decrease_in_metric_set",
			exports: []string{
				`# TYPE a counter
# HELP a help
a_total{a="1",foo="bar"} 1 2
a_total{a="1",foo="bar"} 2 1
# EOF`,
			},
			expectedErr: errMustNotTimestampDecrease,
		},
		{
			name: "bad_timestamp_decrease_between_metric_sets",
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
			expectedErr: errMustNotTimestampDecrease,
		},
		{
			name: "bad_metric_name_change",
			exports: []string{
				`# TYPE a counter
# HELP b help
a_total1 2
# EOF`,
			},
			expectedErr: errors.New(`metric name changed in metadata from "a" to "b"`),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

type testCase struct {
	name        string
	exports     []string
	expectedErr error
}

func run(t *testing.T, tc testCase) {
	s := testScraperLoop()
	var mErr error
	for _, export := range tc.exports {
		_, err := s.parseAndValidate([]byte(export), s.nowFn())
		mErr = multierr.Append(mErr, err)
	}
	if tc.expectedErr == nil {
		require.NoError(t, mErr)
		return
	}
	require.Equal(t, mErr.Error(), tc.expectedErr.Error())
}

func testNowFn() nowFn {
	var sec int64
	return func() time.Time {
		sec++
		return time.Unix(sec, 0)
	}
}

func testScraperLoop() *Loop {
	l := &Loop{
		validator: newValidator(),
		nowFn:     testNowFn(),
	}
	return l
}
