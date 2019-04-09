# openmetricstest

A command line tool to drive testing an Open Metrics parser and exposition implementation.

## Compile

From the repository root:

```
make openmetricstest
```

## Usage

```
Usage of ./openmetricstest:
  -cmd-parser-text string
        command to run to test parser in text mode
  -testdata-dir string
        testdata directory to use (default "./tests/testdata")
```

## Testing an implementation

The tool will walk the `testdata-dir` directory and execute all tests it finds (any directory with `test.json` is a test).

To run the tests against an implementation you must specify a process to test a feature (such as parsing).

When testing a library you will need to have a process that can house the library to test a feature.

### Test types

> `cmd-parser-text`

This command specifies the process to run to test parsing OpenMetrics text exposition test cases.  It should take the OpenMetrics text exposition as stdin.

## Examples

### Testing text parsing with "echo"

This is an example of tersting that "echo" can parse a simple counter text exposition:

```
# Run and since echo will always accept any stdin observe success
./openmetricstest -testdata-dir ../../tests/testdata/parsers/simple_counter -cmd-parser-text echo
2019/04/09 21:25:56 RUN test: simple_counter
2019/04/09 21:25:56 parse-result-validator ok
2019/04/09 21:25:56 PASS test: simple_counter
2019/04/09 21:25:56 OK passed=1, failed=0, total_failures=0
```

### Testing text parsing with a script that always fails

```
# Create a shell script that will always fail on any stdin
echo '#!/bin/bash' > /tmp/fail && echo 'exit 1' >> /tmp/fail && chmod +x /tmp/fail

# Run and since the script will always fail regardless of stdin observe failure
./openmetricstest -testdata-dir ../../tests/testdata/parsers/simple_counter -cmd-parser-text /tmp/fail
2019/04/09 21:40:30 RUN test: simple_counter
2019/04/09 21:40:30 parse-result-validator error:
2019/04/09 21:40:30 > parse error: exit status 1
2019/04/09 21:40:30 FAIL test: simple_counter
2019/04/09 21:40:30 FAILED passed=0, failed=1, total_failures=1
```
