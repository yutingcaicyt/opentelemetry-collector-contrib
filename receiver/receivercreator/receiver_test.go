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

package receivercreator

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol/otelcoltest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/receivertest"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
	"go.uber.org/zap"
	zapObserver "go.uber.org/zap/zaptest/observer"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

type mockObserver struct {
}

func (m *mockObserver) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (m *mockObserver) Shutdown(ctx context.Context) error {
	return nil
}

var _ extension.Extension = (*mockObserver)(nil)

func (m *mockObserver) ListAndWatch(notify observer.Notify) {
	notify.OnAdd([]observer.Endpoint{portEndpoint})
}

func (m *mockObserver) Unsubscribe(_ observer.Notify) {}

var _ observer.Observable = (*mockObserver)(nil)

func TestMockedEndToEnd(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	factories, _ := otelcoltest.NopFactories()
	factories.Receivers[("nop")] = &nopWithEndpointFactory{Factory: receivertest.NewNopFactory()}
	factory := NewFactory()
	factories.Receivers[typeStr] = factory

	host := &mockHostFactories{Host: componenttest.NewNopHost(), factories: factories}
	host.extensions = map[component.ID]component.Component{
		component.NewID("mock_observer"):                      &mockObserver{},
		component.NewIDWithName("mock_observer", "with_name"): &mockObserver{},
	}

	cfg := factory.CreateDefaultConfig()
	sub, err := cm.Sub(component.NewIDWithName(typeStr, "1").String())
	require.NoError(t, err)
	require.NoError(t, component.UnmarshalConfig(sub, cfg))

	params := receivertest.NewNopCreateSettings()
	mockConsumer := new(consumertest.MetricsSink)

	rcvr, err := factory.CreateMetricsReceiver(context.Background(), params, cfg, mockConsumer)
	require.NoError(t, err)
	dyn := rcvr.(*receiverCreator)
	require.NoError(t, rcvr.Start(context.Background(), host))

	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			assert.NoError(t, rcvr.Shutdown(context.Background()))
		})
	}

	defer shutdown()

	require.Eventuallyf(t, func() bool {
		return dyn.observerHandler.receiversByEndpointID.Size() == 2
	}, 1*time.Second, 100*time.Millisecond, "expected 2 receiver but got %v", dyn.observerHandler.receiversByEndpointID)

	// Test that we can send metrics.
	for _, receiver := range dyn.observerHandler.receiversByEndpointID.Values() {
		example := receiver.(*nopWithEndpointReceiver)
		md := pmetric.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("attr", "1")
		rm.Resource().Attributes().PutStr(semconv.AttributeServiceName, "dynamictest")
		m := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		m.SetName("my-metric")
		m.SetDescription("My metric")
		m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(123)
		assert.NoError(t, example.ConsumeMetrics(context.Background(), md))
	}

	// TODO: Will have to rework once receivers are started asynchronously to Start().
	assert.Len(t, mockConsumer.AllMetrics(), 2)
}

func TestLoggingHost(t *testing.T) {
	core, obs := zapObserver.New(zap.ErrorLevel)
	host := &loggingHost{
		Host:   componenttest.NewNopHost(),
		logger: zap.New(core),
	}
	host.ReportFatalError(errors.New("runtime error"))
	require.Equal(t, 1, obs.Len())
	log := obs.All()[0]
	assert.Equal(t, "receiver reported a fatal error", log.Message)
	assert.Equal(t, "runtime error", log.ContextMap()["error"])
}
