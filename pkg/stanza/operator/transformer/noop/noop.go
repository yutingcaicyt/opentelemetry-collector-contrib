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

package noop // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/noop"

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
)

const operatorType = "noop"

func init() {
	operator.Register(operatorType, func() operator.Builder { return NewConfig() })
}

// NewConfig creates a new noop operator config with default values
func NewConfig() *Config {
	return NewConfigWithID(operatorType)
}

// NewConfigWithID creates a new noop operator config with default values
func NewConfigWithID(operatorID string) *Config {
	return &Config{
		TransformerConfig: helper.NewTransformerConfig(operatorID, operatorType),
	}
}

// Config is the configuration of a noop operator.
type Config struct {
	helper.TransformerConfig `mapstructure:",squash"`
}

// Build will build a noop operator.
func (c Config) Build(buildInfo *operator.BuildInfoInternal) (operator.Operator, error) {
	transformerOperator, err := c.TransformerConfig.Build(buildInfo.Logger)
	if err != nil {
		return nil, err
	}

	return &Transformer{
		TransformerOperator: transformerOperator,
	}, nil
}

// Transformer is an operator that performs no operations on an entry.
type Transformer struct {
	helper.TransformerOperator
}

// Process will forward the entry to the next output without any alterations.
func (p *Transformer) Process(ctx context.Context, entry *entry.Entry) error {
	p.Write(ctx, entry)
	return nil
}
