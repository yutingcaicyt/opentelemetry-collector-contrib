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

package filereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filereceiver"

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// stringReader is the only function we use from *bufio.Reader. We define it
// so that it can be swapped out for testing.
type stringReader interface {
	ReadString(delim byte) (string, error)
}

// fileReader
type fileReader struct {
	stringReader stringReader
	unm          pmetric.Unmarshaler
	consumer     consumer.Metrics
	timer        *replayTimer
}

func newFileReader(consumer consumer.Metrics, file *os.File, timer *replayTimer) fileReader {
	return fileReader{
		consumer:     consumer,
		stringReader: bufio.NewReader(file),
		unm:          &pmetric.JSONUnmarshaler{},
		timer:        timer,
	}
}

// readAll calls readline for each line in the file until all lines have been
// read or the context is cancelled.
func (fr fileReader) readAll(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := fr.readLine(ctx)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
		}
	}
}

// readLine reads the next line in the file, converting it into metrics and
// passing it to the the consumer member.
func (fr fileReader) readLine(ctx context.Context) error {
	line, err := fr.stringReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read line from input file: %w", err)
	}
	metrics, err := fr.unm.UnmarshalMetrics([]byte(line))
	if err != nil {
		return fmt.Errorf("failed to unmarshal metrics: %w", err)
	}
	err = fr.timer.wait(ctx, getFirstTimestamp(metrics))
	if err != nil {
		return fmt.Errorf("readLine interrupted while waiting for timer: %w", err)
	}
	return fr.consumer.ConsumeMetrics(ctx, metrics)
}

func getFirstTimestamp(metrics pmetric.Metrics) pcommon.Timestamp {
	resourceMetrics := metrics.ResourceMetrics()
	if resourceMetrics.Len() == 0 {
		return 0
	}
	scopeMetrics := resourceMetrics.At(0).ScopeMetrics()
	if scopeMetrics.Len() == 0 {
		return 0
	}
	metricSlice := scopeMetrics.At(0).Metrics()
	if metricSlice.Len() == 0 {
		return 0
	}
	return getFirstTimestampFromMetric(metricSlice.At(0))
}

func getFirstTimestampFromMetric(metric pmetric.Metric) pcommon.Timestamp {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		if dps.Len() == 0 {
			return 0
		}
		return dps.At(0).Timestamp()
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		if dps.Len() == 0 {
			return 0
		}
		return dps.At(0).Timestamp()
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		if dps.Len() == 0 {
			return 0
		}
		return dps.At(0).Timestamp()
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		if dps.Len() == 0 {
			return 0
		}
		return dps.At(0).Timestamp()
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		if dps.Len() == 0 {
			return 0
		}
		return dps.At(0).Timestamp()
	}
	return 0
}
