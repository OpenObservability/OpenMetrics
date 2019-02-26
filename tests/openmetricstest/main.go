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
	"strings"
)

const (
	testFileName = "test.json"
)

var (
	testDataDirArg       = flag.String("testdata_dir", "testdata", "testdata directory to use")
	cmdTestParserTextArg = flag.String("cmd_test_parser_text", "", "command to run to test parser in text mode")
)

func main() {
	flag.Parse()

	testDataDir := strings.TrimSpace(*testDataDirArg)
	cmdTestParserText := strings.TrimSpace(*cmdTestParserTextArg)

	if cmdTestParserText == "" {
		flag.Usage()
		os.Exit(1)
		return
	}

	runTests(testDataDir, runTestsOptions{
		cmdTestParserText: cmdTestParserText,
	})
}

type runTestsOptions struct {
	cmdTestParserText string
}

func runTests(dir string, opts runTestsOptions) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("cannot read testdata dir: %v", err)
	}

	for _, file := range files {
		if file.Name() == testFileName {
			runTest(dir, opts)
			continue
		}
		if file.IsDir() {
			// Recurse
			runTests(path.Join(dir, file.Name()), opts)
		}
	}
}

type testDef struct {
	Parse   []parseDef   `json:"parse"`
	Outcome []outcomeDef `json:"outcome"`
}

type parseDef struct {
	Type string `json:"type"`
	File string `json:"file"`
}

type outcomeDef struct {
	CanParse *canParseDef `json:"canParse"`
}

type canParseDef struct {
	Result bool `json:"result"`
}

func runTest(dir string, opts runTestsOptions) {
	testFile := path.Join(dir, testFileName)
	data, err := ioutil.ReadFile(testFile)
	if err != nil {
		log.Fatalf("cannot read test file: %v", err)
	}

	var test testDef
	if err := json.Unmarshal(data, &test); err != nil {
		log.Fatalf("cannot parse test file: %v", err)
	}

	for _, step := range test.Parse {
		log.Println("running test: ", dir)

		inputFile := path.Join(dir, step.File)
		input, err := ioutil.ReadFile(inputFile)
		if err != nil {
			log.Fatalf("cannot read test input: file=%s, err=%v", inputFile, err)
		}

		switch step.Type {
		case "text":
			testCmd := opts.cmdTestParserText
			cmd := exec.Command(testCmd)

			stdin, err := cmd.StdinPipe()
			if err != nil {
				log.Fatalf("cannot access stdin: %v", err)
			}

			if err := cmd.Start(); err != nil {
				log.Fatalf("cannot start cmd: %v", err)
			}

			if _, err := stdin.Write(input); err != nil {
				log.Fatalf("cannot write input: %v", err)
			}

			if err := validateTestOutcome(test, cmd); err != nil {
				log.Fatalf("failed test: test=%s, err=%v", dir, err)
			}

			log.Println("passed test: ", dir)
		default:
			log.Fatalf("unknown type: %s", step.Type)
		}
	}
}

func validateTestOutcome(test testDef, cmd *exec.Cmd) error {
	cmdErr := cmd.Wait()
	for _, outcome := range test.Outcome {
		switch {
		case outcome.CanParse != nil:
			def := outcome.CanParse
			if def.Result && cmdErr != nil {
				return fmt.Errorf("expecting no parse error, found error: %v", cmdErr)
			} else if !def.Result && cmdErr == nil {
				return fmt.Errorf("expecting parse error, but none encountered")
			}
		}
	}

	return nil
}
