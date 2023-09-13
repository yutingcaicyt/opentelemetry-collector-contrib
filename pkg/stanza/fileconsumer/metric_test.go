// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fileconsumer // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestFileLogReceiverMetrics(t *testing.T) {
	viewNames := []string{
		"file_read_delay",
	}
	views := metricViews()
	for i, viewName := range viewNames {
		assert.Equal(t, "receiver_filelog_"+viewName, views[i].Name)
	}
}

type testTelemetry struct {
	meter         view.Meter
	promHandler   http.Handler
	useOtel       bool
	meterProvider *sdkmetric.MeterProvider
}

type expectedMetricsValue struct {
	// receiver_filelog_file_read_delay
	fileReadDelaySum int64
	count            int64
}

func telemetryTest(t *testing.T, testFunc func(t *testing.T, tel testTelemetry, useOtel bool)) {
	t.Run("WithOC", func(t *testing.T) {
		testFunc(t, setupTelemetry(t, false), false)
	})

	t.Run("WithOTel", func(t *testing.T) {
		testFunc(t, setupTelemetry(t, true), true)
	})
}

func setupTelemetry(t *testing.T, useOtel bool) testTelemetry {
	// Unregister the views first since they are registered by the init, this way we reset them.
	views := metricViews()
	view.Unregister(views...)
	require.NoError(t, view.Register(views...))

	telemetry := testTelemetry{
		meter:   view.NewMeter(),
		useOtel: useOtel,
	}

	if useOtel {
		promReg := prometheus.NewRegistry()
		exporter, err := otelprom.New(otelprom.WithRegisterer(promReg), otelprom.WithoutUnits(), otelprom.WithoutScopeInfo())
		require.NoError(t, err)

		telemetry.meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(resource.Empty()),
			sdkmetric.WithReader(exporter),
			sdkmetric.WithView(filelogViews()...),
		)

		telemetry.promHandler = promhttp.HandlerFor(promReg, promhttp.HandlerOpts{})

		t.Cleanup(func() { assert.NoError(t, telemetry.meterProvider.Shutdown(context.Background())) })
	} else {
		promReg := prometheus.NewRegistry()

		ocExporter, err := ocprom.NewExporter(ocprom.Options{Registry: promReg})
		require.NoError(t, err)

		telemetry.promHandler = ocExporter

		view.RegisterExporter(ocExporter)
		t.Cleanup(func() { view.UnregisterExporter(ocExporter) })
	}

	return telemetry
}

func filelogViews() []sdkmetric.View {
	return []sdkmetric.View{
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: BuildMetricName("filelog", "file_read_delay")},
			sdkmetric.Stream{Aggregation: aggregation.ExplicitBucketHistogram{
				Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
					100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
					1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
			}},
		),
	}
}

func (tt *testTelemetry) assertMetrics(t *testing.T, expected expectedMetricsValue) {
	for _, v := range metricViews() {
		// Flush the view data.
		_, _ = view.RetrieveData(v.Name)
	}

	req, err := http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	tt.promHandler.ServeHTTP(rr, req)

	var parser expfmt.TextParser
	metrics, err := parser.TextToMetricFamilies(rr.Body)
	require.NoError(t, err)

	name := "receiver_filelog_file_read_delay"

	metricFamily, ok := metrics[name]

	if expected.count > 0 || expected.fileReadDelaySum > 0 {
		require.True(t, ok)
		require.Equal(t, io_prometheus_client.MetricType_HISTOGRAM, metricFamily.GetType())

		firstMetric := metricFamily.Metric[0]
		require.Equal(t, "receiver", firstMetric.Label[0].GetName())
		require.Equal(t, "filelog", firstMetric.Label[0].GetValue())
	} else {
		require.False(t, ok)
	}

	if expected.fileReadDelaySum > 0 {
		require.Equal(t, float64(expected.fileReadDelaySum), metricFamily.Metric[0].GetHistogram().GetSampleSum())
	}

	if expected.count > 0 {
		require.Equal(t, uint64(expected.count), metricFamily.Metric[0].GetHistogram().GetSampleCount())
	}

}
