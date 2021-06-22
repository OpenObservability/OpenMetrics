package scrape

import (
	"context"
	"log"
	"time"

	"github.com/OpenObservability/OpenMetrics/src/validator"
)

// Option sets options in Loop.
type Option func(*Loop)

// WithScrapeTimeout sets the scrape timeout.
func WithScrapeTimeout(timeout time.Duration) Option {
	return func(l *Loop) {
		l.scrapeTimeout = timeout
	}
}

// WithScrapeInterval sets the scrape interval.
func WithScrapeInterval(interval time.Duration) Option {
	return func(l *Loop) {
		l.scrapeInterval = interval
	}
}

// WithErrorLevel sets the error level.
func WithErrorLevel(el validator.ErrorLevel) Option {
	return func(l *Loop) {
		l.validator = validator.NewValidator(el)
	}
}

// Loop and perform scrape and validate in a loop.
type Loop struct {
	validator      *validator.OpenMetricsValidator
	scraper        scraper
	scrapeTimeout  time.Duration
	scrapeInterval time.Duration
}

// NewLoop creates a new scrape and validate loop.
func NewLoop(
	endpoint string,
	opts ...Option,
) *Loop {
	l := &Loop{
		validator: validator.NewValidator(validator.ErrorLevelMust),
		scraper:   newSimpleScraper(endpoint),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Run runs the loop.
func (l *Loop) Run(killAfter time.Duration) {
	l.runOnce()

	ticker := time.NewTicker(l.scrapeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.runOnce()
		case <-time.After(killAfter):
			return
		}
	}
}

func (l *Loop) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), l.scrapeTimeout)
	defer cancel()

	b, err := l.scraper.Scrape(ctx)
	if err != nil {
		log.Printf("scrape failed: %v\n", err)
		return
	}
	log.Println("scraped successfully")

	if err := l.validator.Validate(b); err != nil {
		l.validator.Reset()
		log.Printf("validation failed: %v\n", err)
		return
	}
	log.Println("validated successfully")
}
