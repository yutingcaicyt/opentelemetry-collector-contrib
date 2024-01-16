// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fileconsumer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	rcvr "go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer/internal/emittest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/testutil"
)

func testManager(t *testing.T, cfg *Config) (*Manager, *emittest.Sink) {
	sink := emittest.NewSink()
	return testManagerWithSink(t, cfg, sink), sink
}

func testManagerWithSink(t *testing.T, cfg *Config, sink *emittest.Sink) *Manager {
	input, err := cfg.Build(testutil.Logger(t), sink.Callback)
	require.NoError(t, err)
	t.Cleanup(func() { input.closePreviousFiles() })
	return input
}

func buildTestManagerWithEmit(t *testing.T, cfg *Config, emitChan chan *emitParams) *Manager {
	input, err := cfg.Build(&operator.BuildInfoInternal{Logger: testutil.Logger(t)}, testEmitFunc(emitChan))
	require.NoError(t, err)
	return input
}

func buildTestManagerWithOptionsAndTelemetry(t *testing.T, cfg *Config, useOtel bool, tel testTelemetry) (*Manager, chan *emitParams) {
	emicChan := make(chan *emitParams, 100)
	input, err := cfg.Build(&operator.BuildInfoInternal{CreateSettings: &rcvr.CreateSettings{
		ID: component.NewID("filelog"),
		TelemetrySettings: component.TelemetrySettings{
			MeterProvider: tel.meterProvider,
			MetricsLevel:  configtelemetry.LevelDetailed,
		},
	}, Logger: testutil.Logger(t), TelemetryUseOtel: useOtel}, testEmitFunc(emicChan))

	require.NoError(t, err)
	return input, emicChan
}
