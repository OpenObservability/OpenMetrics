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
