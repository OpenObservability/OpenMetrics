package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/OpenObservability/OpenMetrics/src/cmd/scrapevalidator/scrape"
	"github.com/OpenObservability/OpenMetrics/src/validator"
)

var (
	endpointArg       = flag.String("endpoint", "", "prom endpoint to validate, this is required")
	scrapeTimeoutArg  = flag.Duration("scrape-timeout", 8*time.Second, "timeout for each scrape")
	scrapeIntervalArg = flag.Duration("scrape-interval", 10*time.Second, "time between scrapes")
	errorLevelArg     = flag.String("error-level", "must", "OpenMetrics defines rules in different categories like \"SHOULD\" and \"MUST\", by default this parameter is set to \"must\" so that only the rules in the \"MUST\" category is checked, the alternative value is \"should\" which validates the rules in both categories.")
)

func main() {
	flag.Parse()

	if *endpointArg == "" {
		flag.Usage()
		os.Exit(2)
	}

	opts := []scrape.Option{
		scrape.WithScrapeInterval(*scrapeIntervalArg),
		scrape.WithScrapeTimeout(*scrapeTimeoutArg),
	}
	if *errorLevelArg != "" {
		el, err := validator.NewErrorLevel(*errorLevelArg)
		if err != nil {
			log.Fatalf("invalid error level: %v", err)
		}
		opts = append(opts, scrape.WithErrorLevel(el))
	}

	s := scrape.NewLoop(*endpointArg, opts...)
	if err := s.Run(); err != nil {
		log.Fatalf("validation failed: %s", err)
	}
}
