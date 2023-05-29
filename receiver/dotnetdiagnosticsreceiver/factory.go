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

package dotnetdiagnosticsreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dotnetdiagnosticsreceiver"

import (
	"context"
	"io"
	"math"
	"net"
	"path/filepath"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	rcvr "go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dotnetdiagnosticsreceiver/network"
)

const (
	typeStr   = "dotnet_diagnostics"
	stability = component.StabilityLevelDeprecated
)

func NewFactory() rcvr.Factory {
	return rcvr.NewFactory(
		typeStr,
		createDefaultConfig,
		rcvr.WithMetrics(createMetricsReceiver, stability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		ScraperControllerSettings: scraperhelper.ScraperControllerSettings{
			CollectionInterval: time.Second,
		},
		Counters: []string{"System.Runtime", "Microsoft.AspNetCore.Hosting"},
	}
}

func createMetricsReceiver(
	ctx context.Context,
	params rcvr.CreateSettings,
	baseConfig component.Config,
	consumer consumer.Metrics,
) (rcvr.Metrics, error) {
	cfg := baseConfig.(*Config)
	bw := network.NewBlobWriter(cfg.LocalDebugDir, cfg.MaxLocalDebugFiles, params.Logger)
	sec := int(math.Round(cfg.CollectionInterval.Seconds()))
	return NewReceiver(
		ctx,
		consumer,
		mkConnectionSupplier(cfg.PID, net.Dial, filepath.Glob),
		cfg.Counters,
		sec,
		params.Logger,
		bw,
	)
}

func mkConnectionSupplier(pid int, df network.DialFunc, gf network.GlobFunc) connectionSupplier {
	return func() (io.ReadWriter, error) {
		return network.Connect(pid, df, gf)
	}
}
