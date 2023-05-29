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

package file // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/file"

import (
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
)

const operatorType = "file_input"

func init() {
	operator.Register(operatorType, func() operator.Builder { return NewConfig() })
}

// NewConfig creates a new input config with default values
func NewConfig() *Config {
	return NewConfigWithID(operatorType)
}

// NewConfigWithID creates a new input config with default values
func NewConfigWithID(operatorID string) *Config {
	return &Config{
		InputConfig: helper.NewInputConfig(operatorID, operatorType),
		Config:      *fileconsumer.NewConfig(),
	}
}

// Config is the configuration of a file input operator
type Config struct {
	helper.InputConfig  `mapstructure:",squash"`
	fileconsumer.Config `mapstructure:",squash"`
}

// Build will build a file input operator from the supplied configuration
func (c Config) Build(logger *zap.SugaredLogger) (operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(logger)
	if err != nil {
		return nil, err
	}

	var preEmitOptions []preEmitOption

	if fileconsumer.AllowHeaderMetadataParsing.IsEnabled() {
		preEmitOptions = append(preEmitOptions, setHeaderMetadata)
	}

	if c.IncludeFileName {
		preEmitOptions = append(preEmitOptions, setFileName)
	}
	if c.IncludeFilePath {
		preEmitOptions = append(preEmitOptions, setFilePath)
	}
	if c.IncludeFileNameResolved {
		preEmitOptions = append(preEmitOptions, setFileNameResolved)
	}
	if c.IncludeFilePathResolved {
		preEmitOptions = append(preEmitOptions, setFilePathResolved)
	}

	var toBody toBodyFunc = func(token []byte) interface{} {
		return string(token)
	}
	if helper.IsNop(c.Config.Splitter.EncodingConfig.Encoding) {
		toBody = func(token []byte) interface{} {
			copied := make([]byte, len(token))
			copy(copied, token)
			return copied
		}
	}

	input := &Input{
		InputOperator:  inputOperator,
		toBody:         toBody,
		preEmitOptions: preEmitOptions,
	}

	input.fileConsumer, err = c.Config.Build(logger, input.emit)
	if err != nil {
		return nil, err
	}

	return input, nil
}
