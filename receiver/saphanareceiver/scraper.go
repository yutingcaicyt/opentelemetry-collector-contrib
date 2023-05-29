// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package saphanareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/saphanareceiver"

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/saphanareceiver/internal/metadata"
)

// Runs intermittently, fetching info from SAP HANA, creating metrics/datapoints,
// and feeding them to a metricsConsumer.
type sapHanaScraper struct {
	settings receiver.CreateSettings
	cfg      *Config
	mbs      map[string]*metadata.MetricsBuilder
	factory  sapHanaConnectionFactory
}

func newSapHanaScraper(settings receiver.CreateSettings, cfg *Config, factory sapHanaConnectionFactory) (scraperhelper.Scraper, error) {
	rs := &sapHanaScraper{
		settings: settings,
		cfg:      cfg,
		mbs:      make(map[string]*metadata.MetricsBuilder),
		factory:  factory,
	}
	return scraperhelper.NewScraper(typeStr, rs.scrape)
}

func (s *sapHanaScraper) getMetricsBuilder(resourceAttributes map[string]string) (*metadata.MetricsBuilder, error) {
	bytes, err := json.Marshal(resourceAttributes)
	if err != nil {
		return nil, fmt.Errorf("Error accessing MetricsBuilder for sap hana collection: %w", err)
	}

	key := string(bytes)
	mb, ok := s.mbs[key]
	if !ok {
		mb = metadata.NewMetricsBuilder(s.cfg.MetricsBuilderConfig, s.settings)
		s.mbs[key] = mb
	}

	return mb, nil
}

// Scrape is called periodically, querying SAP HANA and building Metrics to send to
// the next consumer.
func (s *sapHanaScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	client := newSapHanaClient(s.cfg, s.factory)
	if err := client.Connect(ctx); err != nil {
		return pmetric.NewMetrics(), err
	}

	defer client.Close()

	errs := &scrapererror.ScrapeErrors{}
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, query := range queries {
		if query.Enabled == nil || query.Enabled(s.cfg) {
			query.CollectMetrics(ctx, s, client, now, errs)
		}
	}

	metrics := pmetric.NewMetrics()
	for k, mb := range s.mbs {
		var resourceAttributes map[string]string
		err := json.Unmarshal([]byte(k), &resourceAttributes)
		if err != nil {
			errs.Add(fmt.Errorf("Error unmarshaling resource attributes for saphana scraper: %w", err))
			continue
		}
		resourceOptions := []metadata.ResourceMetricsOption{metadata.WithDbSystem("saphana")}
		for attribute, value := range resourceAttributes {
			if attribute == "host" {
				resourceOptions = append(resourceOptions, metadata.WithSaphanaHost(value))
			} else {
				errs.Add(fmt.Errorf("Unsupported resource attribute: %s", attribute))
			}
		}
		resourceMetrics := mb.Emit(resourceOptions...)
		resourceMetrics.ResourceMetrics().At(0).MoveTo(metrics.ResourceMetrics().AppendEmpty())
	}

	s.mbs = make(map[string]*metadata.MetricsBuilder)
	return metrics, errs.Combine()
}
