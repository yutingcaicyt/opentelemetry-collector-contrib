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

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"encoding/hex"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type ExprFunc[K any] func(ctx context.Context, tCtx K) (interface{}, error)

type Expr[K any] struct {
	exprFunc ExprFunc[K]
}

func (e Expr[K]) Eval(ctx context.Context, tCtx K) (interface{}, error) {
	return e.exprFunc(ctx, tCtx)
}

type Getter[K any] interface {
	Get(ctx context.Context, tCtx K) (interface{}, error)
}

type Setter[K any] interface {
	Set(ctx context.Context, tCtx K, val interface{}) error
}

type GetSetter[K any] interface {
	Getter[K]
	Setter[K]
}

type StandardGetSetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
	Setter func(ctx context.Context, tCtx K, val interface{}) error
}

func (path StandardGetSetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	return path.Getter(ctx, tCtx)
}

func (path StandardGetSetter[K]) Set(ctx context.Context, tCtx K, val interface{}) error {
	return path.Setter(ctx, tCtx, val)
}

type literal[K any] struct {
	value interface{}
}

func (l literal[K]) Get(context.Context, K) (interface{}, error) {
	return l.value, nil
}

type exprGetter[K any] struct {
	expr Expr[K]
}

func (g exprGetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	return g.expr.Eval(ctx, tCtx)
}

type listGetter[K any] struct {
	slice []Getter[K]
}

func (l *listGetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	evaluated := make([]any, len(l.slice))

	for i, v := range l.slice {
		val, err := v.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		evaluated[i] = val
	}

	return evaluated, nil
}

// StringGetter is a Getter that must return a string.
type StringGetter[K any] interface {
	// Get retrieves a string value.  If the value is not a string, an error is returned.
	Get(ctx context.Context, tCtx K) (string, error)
}

type IntGetter[K any] interface {
	Get(ctx context.Context, tCtx K) (int64, error)
}

type PMapGetter[K any] interface {
	Get(ctx context.Context, tCtx K) (pcommon.Map, error)
}

type StandardTypeGetter[K any, T any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

func (g StandardTypeGetter[K, T]) Get(ctx context.Context, tCtx K) (T, error) {
	var v T
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return v, err
	}
	if val == nil {
		return v, fmt.Errorf("expected %T but got nil", v)
	}
	v, ok := val.(T)
	if !ok {
		return v, fmt.Errorf("expected %T but got %T", v, val)
	}
	return v, nil
}

// StringLikeGetter is a Getter that returns a string by converting the underlying value to a string if necessary.
type StringLikeGetter[K any] interface {
	// Get retrieves a string value.
	// Unlike `StringGetter`, the expectation is that the underlying value is converted to a string if possible.
	// If the value cannot be converted to a string, nil and an error are returned.
	// If the value is nil, nil is returned without an error.
	Get(ctx context.Context, tCtx K) (*string, error)
}

type StandardStringLikeGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

func (g StandardStringLikeGetter[K]) Get(ctx context.Context, tCtx K) (*string, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	var result string
	switch v := val.(type) {
	case string:
		result = v
	case []byte:
		result = hex.EncodeToString(v)
	case pcommon.Map:
		result, err = jsoniter.MarshalToString(v.AsRaw())
		if err != nil {
			return nil, err
		}
	case pcommon.Slice:
		result, err = jsoniter.MarshalToString(v.AsRaw())
		if err != nil {
			return nil, err
		}
	case pcommon.Value:
		result = v.AsString()
	default:
		result, err = jsoniter.MarshalToString(v)
		if err != nil {
			return nil, fmt.Errorf("unsupported type: %T", v)
		}
	}
	return &result, nil
}

func (p *Parser[K]) newGetter(val value) (Getter[K], error) {
	if val.IsNil != nil && *val.IsNil {
		return &literal[K]{value: nil}, nil
	}

	if s := val.String; s != nil {
		return &literal[K]{value: *s}, nil
	}
	if b := val.Bool; b != nil {
		return &literal[K]{value: bool(*b)}, nil
	}
	if b := val.Bytes; b != nil {
		return &literal[K]{value: ([]byte)(*b)}, nil
	}

	if val.Enum != nil {
		enum, err := p.enumParser(val.Enum)
		if err != nil {
			return nil, err
		}
		return &literal[K]{value: int64(*enum)}, nil
	}

	if eL := val.Literal; eL != nil {
		if f := eL.Float; f != nil {
			return &literal[K]{value: *f}, nil
		}
		if i := eL.Int; i != nil {
			return &literal[K]{value: *i}, nil
		}
		if eL.Path != nil {
			return p.pathParser(eL.Path)
		}
		if eL.Converter != nil {
			call, err := p.newFunctionCall(invocation{
				Function:  eL.Converter.Function,
				Arguments: eL.Converter.Arguments,
			})
			if err != nil {
				return nil, err
			}
			return &exprGetter[K]{
				expr: call,
			}, nil
		}
	}

	if val.List != nil {
		lg := listGetter[K]{slice: make([]Getter[K], len(val.List.Values))}
		for i, v := range val.List.Values {
			getter, err := p.newGetter(v)
			if err != nil {
				return nil, err
			}
			lg.slice[i] = getter
		}
		return &lg, nil
	}

	if val.MathExpression == nil {
		// In practice, can't happen since the DSL grammar guarantees one is set
		return nil, fmt.Errorf("no value field set. This is a bug in the OpenTelemetry Transformation Language")
	}
	return p.evaluateMathExpression(val.MathExpression)
}
