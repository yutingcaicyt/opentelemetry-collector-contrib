// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel/attribute"
)

var (
	errFormatOTLPAttributes       = fmt.Errorf("value should be of the format key=\"value\"")
	errDoubleQuotesOTLPAttributes = fmt.Errorf("value should be a string wrapped in double quotes")
)

type KeyValue map[string]string

var _ pflag.Value = (*KeyValue)(nil)

func (v *KeyValue) String() string {
	return ""
}

func (v *KeyValue) Set(s string) error {
	kv := strings.SplitN(s, "=", 2)
	if len(kv) != 2 {
		return errFormatOTLPAttributes
	}
	val := kv[1]
	if len(val) < 2 || !strings.HasPrefix(val, "\"") || !strings.HasSuffix(val, "\"") {
		return errDoubleQuotesOTLPAttributes
	}

	(*v)[kv[0]] = val[1 : len(val)-1]
	return nil
}

func (v *KeyValue) Type() string {
	return "map[string]string"
}

type Config struct {
	WorkerCount       int
	Rate              int64
	TotalDuration     time.Duration
	ReportingInterval time.Duration

	// OTLP config
	Endpoint           string
	Insecure           bool
	UseHTTP            bool
	Headers            KeyValue
	ResourceAttributes KeyValue
}

func (c *Config) GetAttributes() []attribute.KeyValue {
	var attributes []attribute.KeyValue

	if len(c.ResourceAttributes) > 0 {
		for k, v := range c.ResourceAttributes {
			attributes = append(attributes, attribute.String(k, v))
		}
	}
	return attributes
}

// CommonFlags registers common config flags.
func (c *Config) CommonFlags(fs *pflag.FlagSet) {
	fs.IntVar(&c.WorkerCount, "workers", 1, "Number of workers (goroutines) to run")
	fs.Int64Var(&c.Rate, "rate", 0, "Approximately how many metrics per second each worker should generate. Zero means no throttling.")
	fs.DurationVar(&c.TotalDuration, "duration", 0, "For how long to run the test")
	fs.DurationVar(&c.ReportingInterval, "interval", 1*time.Second, "Reporting interval (default 1 second)")

	fs.StringVar(&c.Endpoint, "otlp-endpoint", "localhost:4317", "Target to which the exporter is going to send metrics. This MAY be configured to include a path (e.g. example.com/v1/metrics)")
	fs.BoolVar(&c.Insecure, "otlp-insecure", false, "Whether to enable client transport security for the exporter's grpc or http connection")
	fs.BoolVar(&c.UseHTTP, "otlp-http", false, "Whether to use HTTP exporter rather than a gRPC one")

	// custom headers
	c.Headers = make(map[string]string)
	fs.Var(&c.Headers, "otlp-header", "Custom header to be passed along with each OTLP request. The value is expected in the format key=value."+
		"Flag may be repeated to set multiple headers (e.g -otlp-header key1=value1 -otlp-header key2=value2)")

	// custom resource attributes
	c.ResourceAttributes = make(map[string]string)
	fs.Var(&c.ResourceAttributes, "otlp-attributes", "Custom resource attributes to use. The value is expected in the format key=\"value\"."+
		"Flag may be repeated to set multiple attributes (e.g -otlp-attributes key1=\"value1\" -otlp-attributes key2=\"value2\")")
}
