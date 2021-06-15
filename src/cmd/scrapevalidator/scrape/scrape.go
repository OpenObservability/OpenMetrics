package scrape

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"
	"unsafe"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/scrape"
)

type nowFn func() time.Time

type ScrapeLoop struct {
	validator      validator
	scraper        scraper
	scrapeTimeout  time.Duration
	scrapeInterval time.Duration

	nowFn nowFn
}

func NewScraperLoop(
	endpoint string,
	scrapeTimeout time.Duration,
	scrapeInterval time.Duration,
) *ScrapeLoop {
	return &ScrapeLoop{
		validator:      newValidator(),
		scraper:        newSimpleScraper(endpoint),
		scrapeTimeout:  scrapeTimeout,
		scrapeInterval: scrapeInterval,
		nowFn:          time.Now,
	}
}

func (s *ScrapeLoop) Run() error {
	ticker := time.NewTicker(s.scrapeInterval)
	defer ticker.Stop()

	if err := s.runOnce(); err != nil {
		return err
	}
	for range ticker.C {
		if err := s.runOnce(); err != nil {
			return err
		}
	}
	return nil
}

func (s *ScrapeLoop) runOnce() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.scrapeTimeout)
	defer cancel()

	b, err := s.scraper.Scrape(ctx)
	if err != nil {
		return err
	}
	log.Println("scraped successfully")
	num, err := s.parseAndValidate(b, s.nowFn())
	if err != nil {
		return err
	}
	log.Printf("parsed %d data points, validated successfully", num)
	return nil
}

// parseAndValidate parses the scraped bytes and validates the metrics against
// OpenMetrics spec between scrapes.
func (s *ScrapeLoop) parseAndValidate(b []byte, ts time.Time) (int, error) {
	var (
		p                  = textparse.NewOpenMetricsParser(b)
		defTime            = timestamp.FromTime(ts)
		m                  scrape.MetricMetadata
		dataPointFound     bool
		numDataPointsFound int
	)
	for {
		// TODO: Handle exemplar.
		et, err := p.Next()
		if err != nil {
			if err == io.EOF {
				// Validate at the end of a scrape.
				return numDataPointsFound, s.validator.Validate()
			}
			return 0, err
		}
		switch et {
		case textparse.EntryType:
			name, metricType := p.Type()
			if err := processMetadata(&dataPointFound, &m, yoloString(name)); err != nil {
				return 0, err
			}
			m.Type = metricType
			continue
		case textparse.EntryHelp:
			name, helpBytes := p.Help()
			if err := processMetadata(&dataPointFound, &m, yoloString(name)); err != nil {
				return 0, err
			}
			m.Help = string(helpBytes)
			continue
		case textparse.EntryUnit:
			name, unitBytes := p.Unit()
			if err := processMetadata(&dataPointFound, &m, yoloString(name)); err != nil {
				return 0, err
			}
			m.Unit = string(unitBytes)
			continue
		case textparse.EntryComment:
			continue
		default:
		}

		t := defTime
		_, tp, v := p.Series()
		if tp != nil {
			t = *tp
		}

		var lset labels.Labels
		_ = p.Metric(&lset)

		if !lset.Has(labels.MetricName) {
			return 0, errors.New("metric must contain a name")
		}
		if err := s.validator.Record(m, lset, t, v); err != nil {
			return 0, err
		}
		// Mark that a metric data point is found.
		dataPointFound = true
		numDataPointsFound++
	}
}

// processMetadata resets the metadata if the parser finds metadata
// for a new metric.
func processMetadata(dataPointFound *bool, m *scrape.MetricMetadata, name string) error {
	if *dataPointFound {
		*dataPointFound = false
		*m = scrape.MetricMetadata{}
		m.Metric = name
		return nil
	}
	if m.Metric != "" && m.Metric != name {
		return fmt.Errorf("metric name changed from %q to %q", m.Metric, name)
	}
	m.Metric = name
	return nil
}

func yoloString(b []byte) string {
	return *((*string)(unsafe.Pointer(&b)))
}
