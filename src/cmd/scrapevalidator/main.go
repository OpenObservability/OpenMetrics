package main

import (
	"flag"
	"log"
	"time"

	"github.com/OpenObservability/OpenMetrics/src/cmd/scrapevalidator/scrape"
)

var (
	endpoint       = flag.String("endpoint", "", "prom endpoint to validate")
	scrapeTimeout  = flag.Duration("scrape-timeout", time.Second, "timeout for each scrape")
	scrapeInterval = flag.Duration("scrape-interval", 10*time.Second, "time between scrapes")
)

func main() {
	flag.Parse()
	s := scrape.NewScrapeLoop(*endpoint, *scrapeTimeout, *scrapeInterval)
	if err := s.Run(); err != nil {
		log.Fatalf("validation failed: %s", err)
	}
}
