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

package syslog // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/syslog"

import (
	"errors"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/tcp"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/udp"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/parser/syslog"
)

const operatorType = "syslog_input"

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
	}
}

type Config struct {
	helper.InputConfig `mapstructure:",squash"`
	syslog.BaseConfig  `mapstructure:",squash"`
	TCP                *tcp.BaseConfig `mapstructure:"tcp"`
	UDP                *udp.BaseConfig `mapstructure:"udp"`
}

func (c Config) Build(buildInfo *operator.BuildInfoInternal) (operator.Operator, error) {
	logger := buildInfo.Logger
	inputBase, err := c.InputConfig.Build(logger)
	if err != nil {
		return nil, err
	}

	syslogParserCfg := syslog.NewConfigWithID(inputBase.ID() + "_internal_tcp")
	syslogParserCfg.BaseConfig = c.BaseConfig
	syslogParserCfg.SetID(inputBase.ID() + "_internal_parser")
	syslogParserCfg.OutputIDs = c.OutputIDs
	syslogParser, err := syslogParserCfg.Build(buildInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve syslog config: %w", err)
	}

	if c.TCP != nil {
		tcpInputCfg := tcp.NewConfigWithID(inputBase.ID() + "_internal_tcp")
		tcpInputCfg.BaseConfig = *c.TCP

		tcpInput, err := tcpInputCfg.Build(buildInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve tcp config: %w", err)
		}

		tcpInput.SetOutputIDs([]string{syslogParser.ID()})
		if err := tcpInput.SetOutputs([]operator.Operator{syslogParser}); err != nil {
			return nil, fmt.Errorf("failed to set outputs")
		}

		return &Input{
			InputOperator: inputBase,
			tcp:           tcpInput.(*tcp.Input),
			parser:        syslogParser.(*syslog.Parser),
		}, nil
	}

	if c.UDP != nil {

		udpInputCfg := udp.NewConfigWithID(inputBase.ID() + "_internal_udp")
		udpInputCfg.BaseConfig = *c.UDP

		// Octet counting and Non-Transparent-Framing are invalid for UDP connections
		if syslogParserCfg.EnableOctetCounting || syslogParserCfg.NonTransparentFramingTrailer != nil {
			return nil, errors.New("octet_counting and non_transparent_framing is not compatible with UDP")
		}

		udpInput, err := udpInputCfg.Build(&operator.BuildInfoInternal{Logger: logger})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve upd config: %w", err)
		}

		udpInput.SetOutputIDs([]string{syslogParser.ID()})
		if err := udpInput.SetOutputs([]operator.Operator{syslogParser}); err != nil {
			return nil, fmt.Errorf("failed to set outputs")
		}

		return &Input{
			InputOperator: inputBase,
			udp:           udpInput.(*udp.Input),
			parser:        syslogParser.(*syslog.Parser),
		}, nil
	}

	return nil, fmt.Errorf("need tcp config or udp config")
}

// Input is an operator that listens for log entries over tcp.
type Input struct {
	helper.InputOperator
	tcp    *tcp.Input
	udp    *udp.Input
	parser *syslog.Parser
}

// Start will start listening for log entries over tcp or udp.
func (t *Input) Start(p operator.Persister) error {
	if t.tcp != nil {
		return t.tcp.Start(p)
	}
	return t.udp.Start(p)
}

// Stop will stop listening for messages.
func (t *Input) Stop() error {
	if t.tcp != nil {
		return t.tcp.Stop()
	}
	return t.udp.Stop()
}

// SetOutputs will set the outputs of the internal syslog parser.
func (t *Input) SetOutputs(operators []operator.Operator) error {
	t.parser.SetOutputIDs(t.GetOutputIDs())
	return t.parser.SetOutputs(operators)
}
