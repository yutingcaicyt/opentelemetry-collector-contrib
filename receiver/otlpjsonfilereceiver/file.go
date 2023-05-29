// Copyright The OpenTelemetry Authors
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

package otlpjsonfilereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/otlpjsonfilereceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	rcvr "go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/otlpjsonfilereceiver/internal/metadata"
)

const (
	typeStr   = "otlpjsonfile"
	transport = "file"
)

// NewFactory creates a factory for file receiver
func NewFactory() rcvr.Factory {
	return rcvr.NewFactory(
		typeStr,
		createDefaultConfig,
		rcvr.WithMetrics(createMetricsReceiver, metadata.Stability),
		rcvr.WithLogs(createLogsReceiver, metadata.Stability),
		rcvr.WithTraces(createTracesReceiver, metadata.Stability))
}

type Config struct {
	fileconsumer.Config `mapstructure:",squash"`
	StorageID           *component.ID `mapstructure:"storage"`
}

func createDefaultConfig() component.Config {
	return &Config{
		Config: *fileconsumer.NewConfig(),
	}
}

type receiver struct {
	input     *fileconsumer.Manager
	id        component.ID
	storageID *component.ID
}

func (f *receiver) Start(ctx context.Context, host component.Host) error {
	storageClient, err := adapter.GetStorageClient(ctx, host, f.storageID, f.id)
	if err != nil {
		return err
	}
	return f.input.Start(storageClient)
}

func (f *receiver) Shutdown(ctx context.Context) error {
	return f.input.Stop()
}

func createLogsReceiver(_ context.Context, settings rcvr.CreateSettings, configuration component.Config, logs consumer.Logs) (rcvr.Logs, error) {
	logsUnmarshaler := &plog.JSONUnmarshaler{}
	obsrecv, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             settings.ID,
		Transport:              transport,
		ReceiverCreateSettings: settings,
	})
	if err != nil {
		return nil, err
	}
	cfg := configuration.(*Config)
	input, err := cfg.Config.Build(settings.Logger.Sugar(), func(ctx context.Context, attrs *fileconsumer.FileAttributes, token []byte) {
		ctx = obsrecv.StartLogsOp(ctx)
		var l plog.Logs
		l, err = logsUnmarshaler.UnmarshalLogs(token)
		if err != nil {
			obsrecv.EndLogsOp(ctx, typeStr, 0, err)
		} else {
			if l.ResourceLogs().Len() != 0 {
				err = logs.ConsumeLogs(ctx, l)
			}
			obsrecv.EndLogsOp(ctx, typeStr, l.LogRecordCount(), err)
		}
	})
	if err != nil {
		return nil, err
	}

	return &receiver{input: input, id: settings.ID, storageID: cfg.StorageID}, nil
}

func createMetricsReceiver(_ context.Context, settings rcvr.CreateSettings, configuration component.Config, metrics consumer.Metrics) (rcvr.Metrics, error) {
	metricsUnmarshaler := &pmetric.JSONUnmarshaler{}
	obsrecv, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             settings.ID,
		Transport:              transport,
		ReceiverCreateSettings: settings,
	})
	if err != nil {
		return nil, err
	}
	cfg := configuration.(*Config)
	input, err := cfg.Config.Build(settings.Logger.Sugar(), func(ctx context.Context, attrs *fileconsumer.FileAttributes, token []byte) {
		ctx = obsrecv.StartMetricsOp(ctx)
		var m pmetric.Metrics
		m, err = metricsUnmarshaler.UnmarshalMetrics(token)
		if err != nil {
			obsrecv.EndMetricsOp(ctx, typeStr, 0, err)
		} else {
			if m.ResourceMetrics().Len() != 0 {
				err = metrics.ConsumeMetrics(ctx, m)
			}
			obsrecv.EndMetricsOp(ctx, typeStr, m.MetricCount(), err)
		}
	})
	if err != nil {
		return nil, err
	}

	return &receiver{input: input, id: settings.ID, storageID: cfg.StorageID}, nil
}

func createTracesReceiver(ctx context.Context, settings rcvr.CreateSettings, configuration component.Config, traces consumer.Traces) (rcvr.Traces, error) {
	tracesUnmarshaler := &ptrace.JSONUnmarshaler{}
	obsrecv, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             settings.ID,
		Transport:              transport,
		ReceiverCreateSettings: settings,
	})
	if err != nil {
		return nil, err
	}
	cfg := configuration.(*Config)
	input, err := cfg.Config.Build(settings.Logger.Sugar(), func(ctx context.Context, attrs *fileconsumer.FileAttributes, token []byte) {
		ctx = obsrecv.StartTracesOp(ctx)
		var t ptrace.Traces
		t, err = tracesUnmarshaler.UnmarshalTraces(token)
		if err != nil {
			obsrecv.EndTracesOp(ctx, typeStr, 0, err)
		} else {
			if t.ResourceSpans().Len() != 0 {
				err = traces.ConsumeTraces(ctx, t)
			}
			obsrecv.EndTracesOp(ctx, typeStr, t.SpanCount(), err)
		}
	})
	if err != nil {
		return nil, err
	}

	return &receiver{input: input, id: settings.ID, storageID: cfg.StorageID}, nil
}
