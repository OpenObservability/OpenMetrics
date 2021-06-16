# openmetricsvalidator

A command tool to wrap the golang [OpenMetrics](https://github.com/prometheus/prometheus/blob/39d79c3cfb86c47d6bc06a9e9317af582f1833bb/pkg/textparse/openmetricsparse.go#L102) parser.

## Compile

From the /src directory:

```
make openmetricsvalidator
```

## Example

Here are some examples of running the golang parser from the root directory.

```
cat ./tests/testdata/parsers/bad_help_0/metrics | ./bin/openmetricsvalidator
2021/06/08 20:04:20 failed to validate input: "INVALID" "\n" is not a valid start token

cat ./tests/testdata/parsers/simple_counter/metrics | ./bin/openmetricsvalidator
2021/06/08 20:04:30 successfully validated input
```

## Known issues

Like the open metrics golang parser, this validator is also failing some test cases that are passing in the python parser.

We are working on updating the tool to bridge the gap between the python client and this validator.
Here is a list of the failing tests:

```
FAIL test: bad_clashing_names_0
FAIL test: bad_clashing_names_1
FAIL test: bad_counter_values_0
FAIL test: bad_counter_values_1
FAIL test: bad_counter_values_10
FAIL test: bad_counter_values_11
FAIL test: bad_counter_values_12
FAIL test: bad_counter_values_13
FAIL test: bad_counter_values_14
FAIL test: bad_counter_values_15
FAIL test: bad_counter_values_16
FAIL test: bad_counter_values_17
FAIL test: bad_counter_values_18
FAIL test: bad_counter_values_19
FAIL test: bad_counter_values_2
FAIL test: bad_counter_values_3
FAIL test: bad_counter_values_4
FAIL test: bad_counter_values_5
FAIL test: bad_counter_values_6
FAIL test: bad_counter_values_7
FAIL test: bad_counter_values_8
FAIL test: bad_counter_values_9
FAIL test: bad_exemplars_6
FAIL test: bad_exemplars_on_unallowed_samples_2
FAIL test: bad_grouping_or_ordering_0
FAIL test: bad_grouping_or_ordering_10
FAIL test: bad_grouping_or_ordering_2
FAIL test: bad_grouping_or_ordering_3
FAIL test: bad_grouping_or_ordering_4
FAIL test: bad_grouping_or_ordering_5
FAIL test: bad_grouping_or_ordering_6
FAIL test: bad_grouping_or_ordering_7
FAIL test: bad_grouping_or_ordering_8
FAIL test: bad_grouping_or_ordering_9
FAIL test: bad_histograms_0
FAIL test: bad_histograms_1
FAIL test: bad_histograms_6
FAIL test: bad_histograms_10
FAIL test: bad_histograms_11
FAIL test: bad_histograms_12
FAIL test: bad_histograms_2
FAIL test: bad_histograms_3
FAIL test: bad_histograms_6
FAIL test: bad_histograms_7
FAIL test: bad_histograms_8
FAIL test: bad_histograms_5
FAIL test: bad_histograms_9
FAIL test: bad_info_and_stateset_values_0
FAIL test: bad_info_and_stateset_values_1
FAIL test: bad_invalid_labels_2
FAIL test: bad_invalid_labels_5
FAIL test: bad_metadata_in_wrong_place_0
FAIL test: bad_metadata_in_wrong_place_1
FAIL test: bad_metadata_in_wrong_place_2
FAIL test: bad_missing_or_invalid_labels_for_a_type_0
FAIL test: bad_missing_or_invalid_labels_for_a_type_1
FAIL test: bad_missing_or_invalid_labels_for_a_type_2
FAIL test: bad_missing_or_invalid_labels_for_a_type_3
FAIL test: bad_missing_or_invalid_labels_for_a_type_4
FAIL test: bad_missing_or_invalid_labels_for_a_type_5
FAIL test: bad_missing_or_invalid_labels_for_a_type_6
FAIL test: bad_missing_or_invalid_labels_for_a_type_7
FAIL test: bad_repeated_metadata_0
FAIL test: bad_repeated_metadata_1
FAIL test: bad_repeated_metadata_3
FAIL test: bad_stateset_info_values_0
FAIL test: bad_stateset_info_values_1
FAIL test: bad_stateset_info_values_2
FAIL test: bad_stateset_info_values_3
FAIL test: bad_timestamp_4
FAIL test: bad_timestamp_5
FAIL test: bad_timestamp_7
FAIL test: bad_unit_6
FAIL test: bad_unit_7
FAILED passed=131, failed=74, total_failures=74
```
