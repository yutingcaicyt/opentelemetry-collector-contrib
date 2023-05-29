// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awskinesisexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awskinesisexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awskinesisexporter/internal/batch"
)

const (
	// The value of "type" key in configuration.
	typeStr = "awskinesis"
	// The stability level of the exporter.
	stability = component.StabilityLevelBeta

	defaultEncoding    = "otlp"
	defaultCompression = "none"
)

// NewFactory creates a factory for Kinesis exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithTraces(NewTracesExporter, stability),
		exporter.WithMetrics(NewMetricsExporter, stability),
		exporter.WithLogs(NewLogsExporter, stability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		TimeoutSettings: exporterhelper.NewDefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.NewDefaultRetrySettings(),
		QueueSettings:   exporterhelper.NewDefaultQueueSettings(),
		Encoding: Encoding{
			Name:        defaultEncoding,
			Compression: defaultCompression,
		},
		AWS: AWSConfig{
			Region: "us-west-2",
		},
		MaxRecordsPerBatch: batch.MaxBatchedRecords,
		MaxRecordSize:      batch.MaxRecordSize,
	}
}

func NewTracesExporter(ctx context.Context, params exporter.CreateSettings, conf component.Config) (exporter.Traces, error) {
	exp, err := createExporter(ctx, conf, params.Logger)
	if err != nil {
		return nil, err
	}
	c := conf.(*Config)
	return exporterhelper.NewTracesExporter(
		ctx,
		params,
		conf,
		exp.consumeTraces,
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithTimeout(c.TimeoutSettings),
		exporterhelper.WithRetry(c.RetrySettings),
		exporterhelper.WithQueue(c.QueueSettings),
	)
}

func NewMetricsExporter(ctx context.Context, params exporter.CreateSettings, conf component.Config) (exporter.Metrics, error) {
	exp, err := createExporter(ctx, conf, params.Logger)
	if err != nil {
		return nil, err
	}
	c := conf.(*Config)
	return exporterhelper.NewMetricsExporter(
		ctx,
		params,
		c,
		exp.consumeMetrics,
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithTimeout(c.TimeoutSettings),
		exporterhelper.WithRetry(c.RetrySettings),
		exporterhelper.WithQueue(c.QueueSettings),
	)
}

func NewLogsExporter(ctx context.Context, params exporter.CreateSettings, conf component.Config) (exporter.Logs, error) {
	exp, err := createExporter(ctx, conf, params.Logger)
	if err != nil {
		return nil, err
	}
	c := conf.(*Config)
	return exporterhelper.NewLogsExporter(
		ctx,
		params,
		c,
		exp.consumeLogs,
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithTimeout(c.TimeoutSettings),
		exporterhelper.WithRetry(c.RetrySettings),
		exporterhelper.WithQueue(c.QueueSettings),
	)
}
