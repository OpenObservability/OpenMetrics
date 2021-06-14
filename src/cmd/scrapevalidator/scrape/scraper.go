package scrape

import "context"

type scraper interface {
	Scrape(ctx context.Context) ([]byte, error)
}
