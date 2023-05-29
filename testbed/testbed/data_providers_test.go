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

package testbed

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const metricsPictPairsFile = "../../internal/goldendataset/testdata/generated_pict_pairs_metrics.txt"

func TestGoldenDataProvider(t *testing.T) {
	dp := NewGoldenDataProvider("", "", metricsPictPairsFile)
	dp.SetLoadGeneratorCounters(&atomic.Uint64{})
	var ms []pmetric.Metrics
	for {
		m, done := dp.GenerateMetrics()
		if done {
			break
		}
		ms = append(ms, m)
	}
	require.Equal(t, len(dp.(*goldenDataProvider).metricsGenerated), len(ms))
}
