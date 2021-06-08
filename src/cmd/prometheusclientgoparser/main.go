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
		log.Println("could not read stdin: ", err)
	}
	if err := parse(b); err != nil {
		os.Exit(1)
	}
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
