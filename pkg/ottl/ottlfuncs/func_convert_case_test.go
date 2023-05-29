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

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func Test_convertCase(t *testing.T) {
	tests := []struct {
		name     string
		target   ottl.StringGetter[interface{}]
		toCase   string
		expected interface{}
	}{
		// snake case
		{
			name: "snake simple convert",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "simpleString", nil
				},
			},
			toCase:   "snake",
			expected: "simple_string",
		},
		{
			name: "snake noop already snake case",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "simple_string", nil
				},
			},
			toCase:   "snake",
			expected: "simple_string",
		},
		{
			name: "snake multiple uppercase",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "CPUUtilizationMetric", nil
				},
			},
			toCase:   "snake",
			expected: "cpu_utilization_metric",
		},
		{
			name: "snake hyphens",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "simple-string", nil
				},
			},
			toCase:   "snake",
			expected: "simple_string",
		},
		{
			name: "snake empty string",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "", nil
				},
			},
			toCase:   "snake",
			expected: "",
		},
		// camel case
		{
			name: "camel simple convert",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "simple_string", nil
				},
			},
			toCase:   "camel",
			expected: "SimpleString",
		},
		{
			name: "snake noop already snake case",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "SimpleString", nil
				},
			},
			toCase:   "camel",
			expected: "SimpleString",
		},
		{
			name: "snake hyphens",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "simple-string", nil
				},
			},
			toCase:   "camel",
			expected: "SimpleString",
		},
		{
			name: "snake empty string",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "", nil
				},
			},
			toCase:   "camel",
			expected: "",
		},
		// upper case
		{
			name: "upper simple",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "simple", nil
				},
			},
			toCase:   "upper",
			expected: "SIMPLE",
		},
		{
			name: "upper complex",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "complex_SET-of.WORDS1234", nil
				},
			},
			toCase:   "upper",
			expected: "COMPLEX_SET-OF.WORDS1234",
		},
		{
			name: "upper empty string",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "", nil
				},
			},
			toCase:   "upper",
			expected: "",
		},
		// lower case
		{
			name: "lower simple",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "SIMPLE", nil
				},
			},
			toCase:   "lower",
			expected: "simple",
		},
		{
			name: "lower complex",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "complex_SET-of.WORDS1234", nil
				},
			},
			toCase:   "lower",
			expected: "complex_set-of.words1234",
		},
		{
			name: "lower empty string",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "", nil
				},
			},
			toCase:   "lower",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exprFunc, err := ConvertCase(tt.target, tt.toCase)
			assert.NoError(t, err)
			result, err := exprFunc(nil, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_convertCaseError(t *testing.T) {
	tests := []struct {
		name   string
		target ottl.StringGetter[interface{}]
		toCase string
	}{
		{
			name: "error bad case",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return "simpleString", nil
				},
			},
			toCase: "unset",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ConvertCase(tt.target, tt.toCase)
			require.Error(t, err)
			assert.ErrorContains(t, err, "invalid case: unset, allowed cases are: lower, upper, snake, camel")
		})
	}
}

func Test_convertCaseRuntimeError(t *testing.T) {
	tests := []struct {
		name          string
		target        ottl.StringGetter[interface{}]
		toCase        string
		expectedError string
	}{
		{
			name: "non-string",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return 10, nil
				},
			},
			toCase:        "upper",
			expectedError: "expected string but got int",
		},
		{
			name: "nil",
			target: &ottl.StandardTypeGetter[interface{}, string]{
				Getter: func(ctx context.Context, tCtx interface{}) (interface{}, error) {
					return nil, nil
				},
			},
			toCase:        "snake",
			expectedError: "expected string but got nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exprFunc, err := ConvertCase[any](tt.target, tt.toCase)
			require.NoError(t, err)
			_, err = exprFunc(context.Background(), nil)
			assert.ErrorContains(t, err, tt.expectedError)
		})
	}
}
