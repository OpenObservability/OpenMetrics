package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/prometheus/prometheus/pkg/textparse"
)

func main() {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("could not read stdin: %v", err)
	}
	if err := parse(b); err != nil {
		log.Fatalf("failed to parse input: %v", err)
	}
	log.Println("successfully parsed input")
}

func parse(b []byte) error {
	p := textparse.NewOpenMetricsParser(b)
	for {
		_, err := p.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
