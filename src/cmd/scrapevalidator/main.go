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
	errorLevelArg     = flag.String("error-level", "should", `OpenMetrics defines rules in different categories like "SHOULD" and "MUST", by default this parameter is set to "should" so that it validates the rules in both the "MUST" and "SHOULD" categories, the alternative value is "must" which validates only the rules in the "MUST" category.`)
	killAfter         = flag.Duration("kill-after", 5*time.Minute, "kill the tool after")
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
	s.Run(*killAfter)
}
