package main

import (
	"fmt"
	"os/exec"
)

type testResult struct {
	cmd *exec.Cmd
	err error
}

type testResultValidator interface {
	Name() string
	ValidateResult(r testResult) error
}

type parseResultValidator struct {
	shouldParse bool
}

func newParseResultValidator(def testDef) testResultValidator {
	return parseResultValidator{shouldParse: def.ShouldParse}
}

func (v parseResultValidator) Name() string {
	return "parse-result-validator"
}

func (v parseResultValidator) ValidateResult(r testResult) error {
	if v.shouldParse && r.err != nil {
		return fmt.Errorf("parse error: %v", r.err)
	} else if !v.shouldParse && r.err == nil {
		return fmt.Errorf("expected a parse error, none found")
	}
	return nil
}
