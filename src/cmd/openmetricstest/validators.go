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

type resultParseValidator struct {
	def resultParseDef
}

func (v resultParseValidator) Name() string {
	return "result-parse-validator"
}

func (v resultParseValidator) ValidateResult(r testResult) error {
	if v.def.Valid && r.err != nil {
		return fmt.Errorf("parse error: %v", r.err)
	} else if !v.def.Valid && r.err == nil {
		return fmt.Errorf("no parse error")
	}
	return nil
}
