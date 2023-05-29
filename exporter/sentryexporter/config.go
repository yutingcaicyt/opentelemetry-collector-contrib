// Copyright The OpenTelemetry Authors
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

package sentryexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sentryexporter"

import (
	"errors"
)

// Config defines the configuration for the Sentry Exporter.
type Config struct {
	// DSN to report transaction to Sentry. If the DSN is not set, no trace will be sent to Sentry.
	DSN string `mapstructure:"dsn"`
	// The deployment environment name, such as production or staging.
	// Environments are case sensitive. The environment name can't contain newlines, spaces or forward slashes,
	// can't be the string "None", or exceed 64 characters.
	Environment string `mapstructure:"environment"`
	// InsecureSkipVerify controls whether the client verifies the Sentry server certificate chain
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Environment == "None" || len(cfg.Environment) > 64 {
		return errors.New("can't be string \"None\" or exceed 64 characters")
	}
	return nil
}
