# openmetricstest

A command line tool to drive testing an Open Metrics parser and exposition implementation.

## Compile

From the /src directory:

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

### Testing parsing

The parse tests require the path to a binary to be supplied, the tests then pass input to that binary through `stdin` and the binary is expected to return a success exit code of 0 if it was able to parse the input, or any other exit code if not able to process the input.

### How it works

The tool will walk the `testdata-dir` directory and execute all tests it finds (any directory with `test.json` is a test).

To run the tests against an implementation you must specify a process to test a feature (such as parsing).

When testing a library you will need to have a process that can house the library to test a feature.

### Test types

> `cmd-parser-text`

This command specifies the process to run to test parsing OpenMetrics text exposition test cases.  It should take the OpenMetrics text exposition as stdin.

## Examples

These examples are from running the tool in the repository root after building it.

### Testing text parsing with "true"

This is an example of testing that "true" can parse a simple counter text exposition:

```
# Run and since echo will always accept any stdin observe success in the root directory.
./bin/openmetricstest -testdata-dir ./tests/testdata/parsers/simple_counter -cmd-parser-text true
2019/07/02 09:15:07 RUN test: simple_counter
2019/07/02 09:15:07 parse-result-validator ok
2019/07/02 09:15:07 PASS test: simple_counter
2019/07/02 09:15:07 OK passed=1, failed=0, total_failures=0
```

### Testing text parsing with a script that always fails

```
# Run and since the script will always fail regardless of stdin observe failure in the root directory.
./bin/openmetricstest -testdata-dir ./tests/testdata/parsers/simple_counter -cmd-parser-text false
2019/04/09 21:40:30 RUN test: simple_counter
2019/04/09 21:40:30 parse-result-validator error:
2019/04/09 21:40:30 > parse error: exit status 1
2019/04/09 21:40:30 FAIL test: simple_counter
2019/04/09 21:40:30 FAILED passed=0, failed=1, total_failures=1
```
