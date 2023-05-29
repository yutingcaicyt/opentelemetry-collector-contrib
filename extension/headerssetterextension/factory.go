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

package headerssetterextension // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension/internal/metadata"
)

const (
	typeStr = "headers_setter"
)

// NewFactory creates a factory for the headers setter extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		typeStr,
		createDefaultConfig,
		createExtension,
		metadata.Stability,
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createExtension(
	_ context.Context,
	settings extension.CreateSettings,
	cfg component.Config,
) (extension.Extension, error) {
	return newHeadersSetterExtension(cfg.(*Config), settings.Logger)
}
