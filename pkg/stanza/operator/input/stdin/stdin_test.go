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

package stdin

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/testutil"
)

func TestStdin(t *testing.T) {
	cfg := NewConfig("")
	cfg.OutputIDs = []string{"fake"}

	op, err := cfg.Build(&operator.BuildInfoInternal{Logger: testutil.Logger(t)})
	require.NoError(t, err)

	fake := testutil.NewFakeOutput(t)
	require.NoError(t, op.SetOutputs([]operator.Operator{fake}))

	r, w, err := os.Pipe()
	require.NoError(t, err)

	stdin := op.(*Input)
	stdin.stdin = r

	require.NoError(t, stdin.Start(testutil.NewMockPersister("test")))
	defer func() {
		require.NoError(t, stdin.Stop())
	}()

	_, err = w.WriteString("test")
	require.NoError(t, err)
	w.Close()
	fake.ExpectBody(t, "test")
}
