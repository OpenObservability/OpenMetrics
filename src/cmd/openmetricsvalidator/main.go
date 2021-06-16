package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/OpenObservability/OpenMetrics/src/validator"
)

func main() {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("could not read stdin: %v", err)
	}
	v := validator.NewValidator(validator.ErrorLevelMust)
	if err := v.Validate(b); err != nil {
		log.Fatalf("failed to validate input: %v", err)
	}
	log.Println("successfully validated input")
}
