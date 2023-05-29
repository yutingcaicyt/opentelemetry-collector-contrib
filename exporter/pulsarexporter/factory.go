// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pulsarexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/pulsarexporter"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	typeStr = "pulsar"
	// The stability level of the exporter.
	stability = component.StabilityLevelAlpha

	defaultTracesTopic  = "otlp_spans"
	defaultMetricsTopic = "otlp_metrics"
	defaultLogsTopic    = "otlp_logs"
	defaultEncoding     = "otlp_proto"
	defaultBroker       = "pulsar://localhost:6650"
)

// FactoryOption applies changes to pulsarExporterFactory.
type FactoryOption func(factory *pulsarExporterFactory)

// WithTracesMarshalers adds tracesMarshalers.
func WithTracesMarshalers(tracesMarshalers ...TracesMarshaler) FactoryOption {
	return func(factory *pulsarExporterFactory) {
		for _, marshaler := range tracesMarshalers {
			factory.tracesMarshalers[marshaler.Encoding()] = marshaler
		}
	}
}

// NewFactory creates Pulsar exporter factory.
func NewFactory(options ...FactoryOption) exporter.Factory {
	f := &pulsarExporterFactory{
		tracesMarshalers:  tracesMarshalers(),
		metricsMarshalers: metricsMarshalers(),
		logsMarshalers:    logsMarshalers(),
	}
	for _, o := range options {
		o(f)
	}
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithTraces(f.createTracesExporter, stability),
		exporter.WithMetrics(f.createMetricsExporter, stability),
		exporter.WithLogs(f.createLogsExporter, stability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		TimeoutSettings: exporterhelper.NewDefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.NewDefaultRetrySettings(),
		QueueSettings:   exporterhelper.NewDefaultQueueSettings(),
		Endpoint:        defaultBroker,
		// using an empty topic to track when it has not been set by user, default is based on traces or metrics.
		Topic:                   "",
		Encoding:                defaultEncoding,
		Authentication:          Authentication{},
		MaxConnectionsPerBroker: 1,
		ConnectionTimeout:       5 * time.Second,
		OperationTimeout:        30 * time.Second,
	}
}

type pulsarExporterFactory struct {
	tracesMarshalers  map[string]TracesMarshaler
	metricsMarshalers map[string]MetricsMarshaler
	logsMarshalers    map[string]LogsMarshaler
}

func (f *pulsarExporterFactory) createTracesExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Traces, error) {
	oCfg := *(cfg.(*Config))
	if oCfg.Topic == "" {
		oCfg.Topic = defaultTracesTopic
	}
	if oCfg.Encoding == "otlp_json" {
		set.Logger.Info("otlp_json is considered experimental and should not be used in a production environment")
	}
	exp, err := newTracesExporter(oCfg, set, f.tracesMarshalers)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewTracesExporter(
		ctx,
		set,
		cfg,
		exp.tracesPusher,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable exporterhelper Timeout, because we cannot pass a Context to the Producer,
		// and will rely on the Pulsar Producer Timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(oCfg.RetrySettings),
		exporterhelper.WithQueue(oCfg.QueueSettings),
		exporterhelper.WithShutdown(exp.Close))
}

func (f *pulsarExporterFactory) createMetricsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Metrics, error) {
	oCfg := *(cfg.(*Config))
	if oCfg.Topic == "" {
		oCfg.Topic = defaultMetricsTopic
	}
	if oCfg.Encoding == "otlp_json" {
		set.Logger.Info("otlp_json is considered experimental and should not be used in a production environment")
	}
	exp, err := newMetricsExporter(oCfg, set, f.metricsMarshalers)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewMetricsExporter(
		ctx,
		set,
		cfg,
		exp.metricsDataPusher,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable exporterhelper Timeout, because we cannot pass a Context to the Producer,
		// and will rely on the sarama Pulsar Timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(oCfg.RetrySettings),
		exporterhelper.WithQueue(oCfg.QueueSettings),
		exporterhelper.WithShutdown(exp.Close))
}

func (f *pulsarExporterFactory) createLogsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Logs, error) {
	oCfg := *(cfg.(*Config))
	if oCfg.Topic == "" {
		oCfg.Topic = defaultLogsTopic
	}
	if oCfg.Encoding == "otlp_json" {
		set.Logger.Info("otlp_json is considered experimental and should not be used in a production environment")
	}
	exp, err := newLogsExporter(oCfg, set, f.logsMarshalers)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		exp.logsDataPusher,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable exporterhelper Timeout, because we cannot pass a Context to the Producer,
		// and will rely on the Pulsar Producer Timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(oCfg.RetrySettings),
		exporterhelper.WithQueue(oCfg.QueueSettings),
		exporterhelper.WithShutdown(exp.Close))
}
