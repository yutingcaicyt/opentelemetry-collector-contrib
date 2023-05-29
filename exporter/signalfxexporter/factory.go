// Copyright 2019, OpenTelemetry Authors
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

package signalfxexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/signalfxexporter"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/signalfxexporter/internal/correlation"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchperresourceattr"
)

const (
	// The value of "type" key in configuration.
	typeStr = "signalfx"
	// The stability level of the exporter.
	stability = component.StabilityLevelBeta

	defaultHTTPTimeout = time.Second * 5

	defaultMaxConns = 100
)

// NewFactory creates a factory for SignalFx exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, stability),
		exporter.WithLogs(createLogsExporter, stability),
		exporter.WithTraces(createTracesExporter, stability),
	)
}

func createDefaultConfig() component.Config {
	maxConnCount := defaultMaxConns
	idleConnTimeout := 30 * time.Second

	return &Config{
		RetrySettings: exporterhelper.NewDefaultRetrySettings(),
		QueueSettings: exporterhelper.NewDefaultQueueSettings(),
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Timeout:             defaultHTTPTimeout,
			MaxIdleConns:        &maxConnCount,
			MaxIdleConnsPerHost: &maxConnCount,
			IdleConnTimeout:     &idleConnTimeout,
		},
		AccessTokenPassthroughConfig: splunk.AccessTokenPassthroughConfig{
			AccessTokenPassthrough: true,
		},
		DeltaTranslationTTL:           3600,
		Correlation:                   correlation.DefaultConfig(),
		NonAlphanumericDimensionChars: "_-.",
	}
}

func createTracesExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	eCfg component.Config,
) (exporter.Traces, error) {
	cfg := eCfg.(*Config)
	corrCfg := cfg.Correlation

	if corrCfg.HTTPClientSettings.Endpoint == "" {
		apiURL, err := cfg.getAPIURL()
		if err != nil {
			return nil, fmt.Errorf("unable to create API URL: %w", err)
		}
		corrCfg.HTTPClientSettings.Endpoint = apiURL.String()
	}
	if cfg.AccessToken == "" {
		return nil, errors.New("access_token is required")
	}
	set.Logger.Info("Correlation tracking enabled", zap.String("endpoint", corrCfg.HTTPClientSettings.Endpoint))
	tracker := correlation.NewTracker(corrCfg, cfg.AccessToken, set)

	return exporterhelper.NewTracesExporter(
		ctx,
		set,
		cfg,
		tracker.AddSpans,
		exporterhelper.WithStart(tracker.Start),
		exporterhelper.WithShutdown(tracker.Shutdown))
}

func createMetricsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	config component.Config,
) (exporter.Metrics, error) {
	cfg := config.(*Config)

	exp, err := newSignalFxExporter(cfg, set)
	if err != nil {
		return nil, err
	}

	me, err := exporterhelper.NewMetricsExporter(
		ctx,
		set,
		cfg,
		exp.pushMetrics,
		// explicitly disable since we rely on http.Client timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(cfg.RetrySettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown))

	if err != nil {
		return nil, err
	}

	// If AccessTokenPassthrough enabled, split the incoming Metrics data by splunk.SFxAccessTokenLabel,
	// this ensures that we get batches of data for the same token when pushing to the backend.
	if cfg.AccessTokenPassthrough {
		me = &baseMetricsExporter{
			Component: me,
			Metrics:   batchperresourceattr.NewBatchPerResourceMetrics(splunk.SFxAccessTokenLabel, me),
		}
	}

	return &signalfMetadataExporter{
		Metrics:  me,
		exporter: exp,
	}, nil
}

func createLogsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Logs, error) {
	expCfg := cfg.(*Config)

	exp, err := newEventExporter(expCfg, set)
	if err != nil {
		return nil, err
	}

	le, err := exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		exp.pushLogs,
		// explicitly disable since we rely on http.Client timeout logic.
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(expCfg.RetrySettings),
		exporterhelper.WithQueue(expCfg.QueueSettings),
		exporterhelper.WithStart(exp.startLogs))

	if err != nil {
		return nil, err
	}

	// If AccessTokenPassthrough enabled, split the incoming Metrics data by splunk.SFxAccessTokenLabel,
	// this ensures that we get batches of data for the same token when pushing to the backend.
	if expCfg.AccessTokenPassthrough {
		le = &baseLogsExporter{
			Component: le,
			Logs:      batchperresourceattr.NewBatchPerResourceLogs(splunk.SFxAccessTokenLabel, le),
		}
	}

	return le, nil
}
