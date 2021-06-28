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
FAIL test: bad_timestamp_4
FAIL test: bad_timestamp_5
FAIL test: bad_timestamp_7
FAIL test: duplicate_timestamps
FAILED passed=201, failed=4, total_failures=4
```