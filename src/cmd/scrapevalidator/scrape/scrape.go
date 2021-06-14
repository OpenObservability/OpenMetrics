package scrape

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/pkg/timestamp"
)

type nowFn func() time.Time

type scrapeLoop struct {
	validator validator
	scraper   scraper
	nowFn     nowFn
}

func newScraperLoop() *scrapeLoop {
	return &scrapeLoop{
		validator: newValidator(),
		nowFn:     time.Now,
	}
}

func (s *scrapeLoop) run() error {
	for {
		b, err := s.scraper.Scrape(context.TODO())
		if err != nil {
			return err
		}
		if err := s.parseAndValidate(b, s.nowFn()); err != nil {
			return err
		}
	}
}

// parseAndValidate parses the scraped bytes and validates the metrics against
// OpenMetrics spec between scrapes.
func (s *scrapeLoop) parseAndValidate(b []byte, ts time.Time) error {
	var (
		p          = textparse.NewOpenMetricsParser(b)
		defTime    = timestamp.FromTime(ts)
		m          metadata
		foundValue bool
	)
	for {
		// TODO: Handle exemplar.
		et, err := p.Next()
		if err != nil {
			if err == io.EOF {
				// Validate at the end of a scrape.
				return s.validator.Validate()
			}
			return err
		}
		switch et {
		case textparse.EntryType:
			processMetadata(&foundValue, &m)
			_, metricType := p.Type()
			m.metricType = metricType
			continue
		case textparse.EntryHelp:
			processMetadata(&foundValue, &m)
			_, helpBytes := p.Help()
			m.help = string(helpBytes)
			continue
		case textparse.EntryUnit:
			processMetadata(&foundValue, &m)
			_, unitBytes := p.Unit()
			m.unit = string(unitBytes)
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
			return errors.New("metric must contain a name")
		}
		if err := s.validator.Record(m, lset, t, v); err != nil {
			return err
		}
		// Mark that a metric value is found.
		foundValue = true
	}
}

// processMetadata resets the metadata if the parser finds metadata
// for a new metric.
func processMetadata(foundValue *bool, m *metadata) {
	if *foundValue {
		*foundValue = false
		*m = metadata{}
	}
}
