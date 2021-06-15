package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/OpenObservability/OpenMetrics/src/cmd/scrapevalidator/scrape"
)

var (
	endpoint       = flag.String("endpoint", "", "prom endpoint to validate")
	scrapeTimeout  = flag.Duration("scrape-timeout", 8*time.Second, "timeout for each scrape")
	scrapeInterval = flag.Duration("scrape-interval", 10*time.Second, "time between scrapes")
)

func main() {
	flag.Parse()

	if *endpoint == "" {
		flag.Usage()
		os.Exit(2)
	}
	opts := []scrape.Option{
		scrape.WithScrapeInterval(*scrapeInterval),
		scrape.WithScrapeTimeout(*scrapeTimeout),
	}
	s := scrape.NewScrapeLoop(*endpoint, opts...)
	if err := s.Run(); err != nil {
		log.Fatalf("validation failed: %s", err)
	}
}
