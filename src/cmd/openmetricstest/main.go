package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type code int

const (
	testFileName = "test.json"

	exitOK code = iota
	exitUsage
	exitFailures
)

var (
	testDataDirArg       = flag.String("testdata-dir", "./tests/testdata", "testdata directory to use")
	cmdTestParserTextArg = flag.String("cmd-parser-text", "", "command to run to test parser in text mode")
)

func main() {
	flag.Parse()

	testDataDir := *testDataDirArg
	cmdTestParserText := *cmdTestParserTextArg

	if cmdTestParserText == "" {
		flag.Usage()
		os.Exit(int(exitUsage))
		return
	}

	r, err := runTests(testDataDir, runTestsOptions{
		cmdTestParserText: cmdTestParserText,
	})
	if err != nil {
		log.Fatalf("failed to run tests: %v", err)
	}

	var success, failed, failures int
	for _, elem := range r.tests {
		if len(elem.failures) == 0 {
			success++
		} else {
			failed++
		}
		failures += len(elem.failures)
	}

	result := fmt.Sprintf("passed=%d, failed=%d, total_failures=%d",
		success, failed, failures)
	if failures != 0 {
		log.Println("FAILED", result)
		os.Exit(1)
	}

	log.Println("OK", result)
}

type runTestsOptions struct {
	cmdTestParserText string
}

type runTestsResults struct {
	tests []runTestResult
}

type runTestResult struct {
	name     string
	failures []testFailure
}

type testFailure struct {
	name string
	err  error
}

func runTests(dir string, opts runTestsOptions) (runTestsResults, error) {
	var results runTestsResults
	err := filepath.Walk(dir, func(filePath string, file os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("cannot read path=%s: %v", filePath, err)
		}
		if file.Name() != testFileName {
			return nil
		}

		testDir, _ := path.Split(filePath)
		result, err := runTest(testDir, opts)
		results.tests = append(results.tests, result)
		return err
	})
	return results, err
}

type testDef struct {
	Comment     string `json:"comment"`
	Type        string `json:"type"`
	File        string `json:"file"`
	ShouldParse bool   `json:"shouldParse"`
}

func runTest(dir string, opts runTestsOptions) (runTestResult, error) {
	result := runTestResult{
		name: path.Base(dir),
	}
	testFile := path.Join(dir, testFileName)
	data, err := ioutil.ReadFile(testFile)
	if err != nil {
		return result, fmt.Errorf("cannot read test file: %v", err)
	}

	var test testDef
	if err := json.Unmarshal(data, &test); err != nil {
		return result, fmt.Errorf("cannot parse test file: %v", err)
	}

	var description string
	if test.Comment != "" {
		description = fmt.Sprintf("(%s)", test.Comment)
	}

	log.Println("RUN test:", result.name, description)

	inputFile := path.Join(dir, test.File)
	input, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return result, fmt.Errorf("cannot read test input: file=%s, err=%v", inputFile, err)
	}

	switch test.Type {
	case "text":
		var testArgs []string
		testCmd := opts.cmdTestParserText
		if strings.Contains(testCmd, " ") {
			// Extrapolate the rest of command as arguments.
			split := strings.Split(testCmd, " ")
			testCmd = split[0]
			testArgs = split[1:]
		}

		cmd := exec.Command(testCmd, testArgs...)

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return result, fmt.Errorf("cannot access stdin: %v", err)
		}

		if err := cmd.Start(); err != nil {
			return result, fmt.Errorf("cannot start cmd: %v", err)
		}

		if _, err := stdin.Write(input); err != nil {
			return result, fmt.Errorf("cannot write input: %v", err)
		}
		if err := stdin.Close(); err != nil {
			return result, fmt.Errorf("cannot close input: %v", err)
		}

		// Run validators on result and collect failures
		result.failures = validateResult(test, cmd)

		if len(result.failures) > 0 {
			log.Println("FAIL test:", result.name)
			return result, nil
		}

		log.Println("PASS test:", result.name)
	default:
		return result, fmt.Errorf("parse type unknown: %s", test.Type)
	}

	return result, nil
}

func validateResult(test testDef, cmd *exec.Cmd) []testFailure {
	result := testResult{
		cmd: cmd,
		err: cmd.Wait(),
	}

	var failures []testFailure
	validate := func(v testResultValidator) {
		if err := v.ValidateResult(result); err != nil {
			log.Println(v.Name(), "error:")
			log.Println(">", err)
			failures = append(failures, testFailure{
				name: v.Name(),
				err:  err,
			})
			return
		}
		log.Println(v.Name(), "ok")
	}

	// Run validators
	validate(newParseResultValidator(test))

	// Future validators come here
	return failures
}
