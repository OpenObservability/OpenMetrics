package scrape

import (
	"context"
	"io/ioutil"
	"net/http"
)

type scraper interface {
	Scrape(ctx context.Context) ([]byte, error)
}

type simpleScraper struct {
	addr string
}

func newSimpleScraper(addr string) *simpleScraper {
	return &simpleScraper{addr: addr}
}

func (s simpleScraper) Scrape(ctx context.Context) ([]byte, error) {
	resp, err := http.Get(s.addr)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}
