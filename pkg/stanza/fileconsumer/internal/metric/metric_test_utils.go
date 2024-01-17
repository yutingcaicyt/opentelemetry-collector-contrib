// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

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
	"go.opentelemetry.io/otel/sdk/resource"
)

type TestTelemetry struct {
	meter         view.Meter
	promHandler   http.Handler
	useOtel       bool
	MeterProvider *sdkmetric.MeterProvider
}

type ExpectedMetricsValue struct {
	// receiver_filelog_file_read_delay
	FileReadDelaySum int64
	Count            int64
}

func TelemetryTest(t *testing.T, testFunc func(t *testing.T, tel TestTelemetry, useOtel bool)) {
	t.Run("WithOC", func(t *testing.T) {
		testFunc(t, SetupTelemetry(t, false), false)
	})

	t.Run("WithOTel", func(t *testing.T) {
		testFunc(t, SetupTelemetry(t, true), true)
	})
}

func SetupTelemetry(t *testing.T, useOtel bool) TestTelemetry {
	// Unregister the views first since they are registered by the init, this way we reset them.
	views := metricViews()
	view.Unregister(views...)
	require.NoError(t, view.Register(views...))

	telemetry := TestTelemetry{
		meter:   view.NewMeter(),
		useOtel: useOtel,
	}

	if useOtel {
		promReg := prometheus.NewRegistry()
		exporter, err := otelprom.New(otelprom.WithRegisterer(promReg), otelprom.WithoutUnits(), otelprom.WithoutScopeInfo())
		require.NoError(t, err)

		telemetry.MeterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(resource.Empty()),
			sdkmetric.WithReader(exporter),
			sdkmetric.WithView(filelogViews()...),
		)

		telemetry.promHandler = promhttp.HandlerFor(promReg, promhttp.HandlerOpts{})

		t.Cleanup(func() { assert.NoError(t, telemetry.MeterProvider.Shutdown(context.Background())) })
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
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
					100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
					1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
			}},
		),
	}
}

func (tt *TestTelemetry) AssertMetrics(t *testing.T, expected ExpectedMetricsValue) {
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

	if expected.Count > 0 || expected.FileReadDelaySum > 0 {
		require.True(t, ok)
		require.Equal(t, io_prometheus_client.MetricType_HISTOGRAM, metricFamily.GetType())

		firstMetric := metricFamily.Metric[0]
		require.Equal(t, "receiver", firstMetric.Label[0].GetName())
		require.Equal(t, "filelog", firstMetric.Label[0].GetValue())
	} else {
		require.False(t, ok)
	}

	if expected.FileReadDelaySum > 0 {
		require.Equal(t, float64(expected.FileReadDelaySum), metricFamily.Metric[0].GetHistogram().GetSampleSum())
	}

	if expected.Count > 0 {
		require.Equal(t, uint64(expected.Count), metricFamily.Metric[0].GetHistogram().GetSampleCount())
	}

}
