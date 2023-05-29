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

package datadogreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/datadogreceiver"

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/stretchr/testify/assert"
	vmsgp "github.com/vmihailenco/msgpack/v4"
)

var data = [2]interface{}{
	0: []string{
		0:  "baggage",
		1:  "item",
		2:  "elasticsearch.version",
		3:  "7.0",
		4:  "my-name",
		5:  "X",
		6:  "my-service",
		7:  "my-resource",
		8:  "_dd.sampling_rate_whatever",
		9:  "value whatever",
		10: "sql",
		11: "service.name",
	},
	1: [][][12]interface{}{
		{
			{
				6,
				4,
				7,
				uint64(12345678901234561234),
				uint64(2),
				uint64(3),
				int64(123),
				int64(456),
				1,
				map[interface{}]interface{}{
					8:  9,
					0:  1,
					2:  3,
					11: 6,
				},
				map[interface{}]float64{
					5: 1.2,
				},
				10,
			},
		},
	},
}

func TestTracePayloadV05Unmarshalling(t *testing.T) {
	var traces pb.Traces
	payload, err := vmsgp.Marshal(&data)
	assert.NoError(t, err)
	if err := traces.UnmarshalMsgDictionary(payload); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, "/v0.5/traces", io.NopCloser(bytes.NewReader(payload)))
	translated := toTraces(&pb.TracerPayload{
		LanguageName:    req.Header.Get("Datadog-Meta-Lang"),
		LanguageVersion: req.Header.Get("Datadog-Meta-Lang-Version"),
		Chunks:          traceChunksFromTraces(traces),
		TracerVersion:   req.Header.Get("Datadog-Meta-Tracer-Version"),
	}, req)
	assert.Equal(t, 1, translated.SpanCount(), "Span Count wrong")
	span := translated.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	assert.NotNil(t, span)
	assert.Equal(t, 4, span.Attributes().Len(), "missing tags")
	value, exists := span.Attributes().Get("service.name")
	assert.True(t, exists, "service.name missing")
	assert.Equal(t, "my-service", value.AsString(), "service.name tag value incorrect")
	assert.Equal(t, span.Name(), "my-resource")
}

func TestTracePayloadV07Unmarshalling(t *testing.T) {
	var traces pb.Traces
	payload, err := vmsgp.Marshal(&data)
	assert.NoError(t, err)
	if err2 := traces.UnmarshalMsgDictionary(payload); err2 != nil {
		t.Fatal(err2)
	}
	apiPayload := pb.TracerPayload{
		LanguageName:    "1",
		LanguageVersion: "1",
		Chunks:          traceChunksFromTraces(traces),
		TracerVersion:   "1",
	}
	var reqBytes []byte
	bytez, _ := apiPayload.MarshalMsg(reqBytes)
	req, _ := http.NewRequest(http.MethodPost, "/v0.7/traces", io.NopCloser(bytes.NewReader(bytez)))

	translated, _ := handlePayload(req)
	span := translated.GetChunks()[0].GetSpans()[0]
	assert.NotNil(t, span)
	assert.Equal(t, 4, len(span.GetMeta()), "missing tags")
	value, exists := span.GetMeta()["service.name"]
	assert.True(t, exists, "service.name missing")
	assert.Equal(t, "my-service", value, "service.name tag value incorrect")
	assert.Equal(t, "my-name", span.GetName())
}

func BenchmarkTranslatorv05(b *testing.B) {
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		TestTracePayloadV05Unmarshalling(&testing.T{})
	}
	b.StopTimer()
}

func BenchmarkTranslatorv07(b *testing.B) {
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		TestTracePayloadV07Unmarshalling(&testing.T{})
	}
	b.StopTimer()
}
