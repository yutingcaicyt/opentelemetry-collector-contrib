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

package ottlfuncs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func Test_truncateAll(t *testing.T) {
	input := pcommon.NewMap()
	input.PutStr("test", "hello world")
	input.PutInt("test2", 3)
	input.PutBool("test3", true)

	target := &ottl.StandardTypeGetter[pcommon.Map, pcommon.Map]{
		Getter: func(ctx context.Context, tCtx pcommon.Map) (interface{}, error) {
			return tCtx, nil
		},
	}

	tests := []struct {
		name   string
		target ottl.PMapGetter[pcommon.Map]
		limit  int64
		want   func(pcommon.Map)
	}{
		{
			name:   "truncate map",
			target: target,
			limit:  1,
			want: func(expectedMap pcommon.Map) {
				expectedMap.PutStr("test", "h")
				expectedMap.PutInt("test2", 3)
				expectedMap.PutBool("test3", true)
			},
		},
		{
			name:   "truncate map to zero",
			target: target,
			limit:  0,
			want: func(expectedMap pcommon.Map) {
				expectedMap.PutStr("test", "")
				expectedMap.PutInt("test2", 3)
				expectedMap.PutBool("test3", true)
			},
		},
		{
			name:   "truncate nothing",
			target: target,
			limit:  100,
			want: func(expectedMap pcommon.Map) {
				expectedMap.PutStr("test", "hello world")
				expectedMap.PutInt("test2", 3)
				expectedMap.PutBool("test3", true)
			},
		},
		{
			name:   "truncate exact",
			target: target,
			limit:  11,
			want: func(expectedMap pcommon.Map) {
				expectedMap.PutStr("test", "hello world")
				expectedMap.PutInt("test2", 3)
				expectedMap.PutBool("test3", true)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenarioMap := pcommon.NewMap()
			input.CopyTo(scenarioMap)

			exprFunc, err := TruncateAll(tt.target, tt.limit)
			assert.NoError(t, err)

			result, err := exprFunc(nil, scenarioMap)
			assert.NoError(t, err)
			assert.Nil(t, result)

			expected := pcommon.NewMap()
			tt.want(expected)

			assert.Equal(t, expected, scenarioMap)
		})
	}
}

func Test_truncateAll_validation(t *testing.T) {
	_, err := TruncateAll[interface{}](&ottl.StandardTypeGetter[interface{}, pcommon.Map]{}, -1)
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid limit for truncate_all function, -1 cannot be negative")
}

func Test_truncateAll_bad_input(t *testing.T) {
	input := pcommon.NewValueStr("not a map")
	target := &ottl.StandardTypeGetter[interface{}, pcommon.Map]{
		Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
			return tCtx, nil
		},
	}

	exprFunc, err := TruncateAll[interface{}](target, 1)
	assert.NoError(t, err)

	_, err = exprFunc(nil, input)
	assert.Error(t, err)
}

func Test_truncateAll_get_nil(t *testing.T) {
	target := &ottl.StandardTypeGetter[interface{}, pcommon.Map]{
		Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
			return tCtx, nil
		},
	}

	exprFunc, err := TruncateAll[interface{}](target, 1)
	assert.NoError(t, err)

	_, err = exprFunc(nil, nil)
	assert.Error(t, err)
}
