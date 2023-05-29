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

package fileconsumer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/parser/regex"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/testutil"
)

func TestTokenization(t *testing.T) {
	testCases := []struct {
		testName    string
		fileContent []byte
		expected    [][]byte
	}{
		{
			"simple",
			[]byte("testlog1\ntestlog2\n"),
			[][]byte{
				[]byte("testlog1"),
				[]byte("testlog2"),
			},
		},
		{
			"empty_only",
			[]byte("\n"),
			[][]byte{
				[]byte(""),
			},
		},
		{
			"empty_first",
			[]byte("\ntestlog1\ntestlog2\n"),
			[][]byte{
				[]byte(""),
				[]byte("testlog1"),
				[]byte("testlog2"),
			},
		},
		{
			"empty_between_lines",
			[]byte("testlog1\n\ntestlog2\n"),
			[][]byte{
				[]byte("testlog1"),
				[]byte(""),
				[]byte("testlog2"),
			},
		},
		{
			"multiple_empty",
			[]byte("\n\ntestlog1\n\n\ntestlog2\n"),
			[][]byte{
				[]byte(""),
				[]byte(""),
				[]byte("testlog1"),
				[]byte(""),
				[]byte(""),
				[]byte("testlog2"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			f, emitChan := testReaderFactory(t)

			temp := openTemp(t, t.TempDir())
			_, err := temp.Write(tc.fileContent)
			require.NoError(t, err)

			r, err := f.newReaderBuilder().withFile(temp).build()
			require.NoError(t, err)

			r.ReadToEnd(context.Background())

			for _, expected := range tc.expected {
				require.Equal(t, expected, readToken(t, emitChan))
			}
		})
	}
}

func TestTokenizationTooLong(t *testing.T) {
	fileContent := []byte("aaaaaaaaaaaaaaaaaaaaaa\naaa\n")
	expected := [][]byte{
		[]byte("aaaaaaaaaa"),
		[]byte("aaaaaaaaaa"),
		[]byte("aa"),
		[]byte("aaa"),
	}

	f, emitChan := testReaderFactory(t)
	f.readerConfig.maxLogSize = 10

	temp := openTemp(t, t.TempDir())
	_, err := temp.Write(fileContent)
	require.NoError(t, err)

	r, err := f.newReaderBuilder().withFile(temp).build()
	require.NoError(t, err)

	r.ReadToEnd(context.Background())

	for _, expected := range expected {
		require.Equal(t, expected, readToken(t, emitChan))
	}
}

func TestTokenizationTooLongWithLineStartPattern(t *testing.T) {
	fileContent := []byte("aaa2023-01-01aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa 2023-01-01 2 2023-01-01")
	expected := [][]byte{
		[]byte("aaa"),
		[]byte("2023-01-01aaaaa"),
		[]byte("aaaaaaaaaaaaaaa"),
		[]byte("aaaaaaaaaaaaaaa"),
		[]byte("aaaaa"),
		[]byte("2023-01-01 2"),
	}

	f, emitChan := testReaderFactory(t)

	mlc := helper.NewMultilineConfig()
	mlc.LineStartPattern = `\d+-\d+-\d+`
	f.splitterFactory = newMultilineSplitterFactory(helper.SplitterConfig{
		EncodingConfig: helper.NewEncodingConfig(),
		Flusher:        helper.NewFlusherConfig(),
		Multiline:      mlc,
	})
	f.readerConfig.maxLogSize = 15

	temp := openTemp(t, t.TempDir())
	_, err := temp.Write(fileContent)
	require.NoError(t, err)

	r, err := f.newReaderBuilder().withFile(temp).build()
	require.NoError(t, err)

	r.ReadToEnd(context.Background())
	require.True(t, r.eof)

	for _, expected := range expected {
		require.Equal(t, expected, readToken(t, emitChan))
	}
}

func TestHeaderFingerprintIncluded(t *testing.T) {
	fileContent := []byte("#header-line\naaa\n")

	f, _ := testReaderFactory(t)
	f.readerConfig.maxLogSize = 10

	regexConf := regex.NewConfig()
	regexConf.Regex = "^#(?P<header>.*)"

	headerConf := &HeaderConfig{
		Pattern: "^#",
		MetadataOperators: []operator.Config{
			{
				Builder: regexConf,
			},
		},
	}

	enc, err := helper.EncodingConfig{
		Encoding: "utf-8",
	}.Build()
	require.NoError(t, err)

	h, err := headerConf.buildHeaderSettings(enc.Encoding)
	require.NoError(t, err)
	f.headerSettings = h

	temp := openTemp(t, t.TempDir())

	r, err := f.newReaderBuilder().withFile(temp).build()
	require.NoError(t, err)

	_, err = temp.Write(fileContent)
	require.NoError(t, err)

	r.ReadToEnd(context.Background())

	require.Equal(t, []byte("#header-line\naaa\n"), r.Fingerprint.FirstBytes)
}

func testReaderFactory(t *testing.T) (*readerFactory, chan *emitParams) {
	emitChan := make(chan *emitParams, 100)
	splitterConfig := helper.NewSplitterConfig()
	return &readerFactory{
		SugaredLogger: testutil.Logger(t),
		readerConfig: &readerConfig{
			fingerprintSize: DefaultFingerprintSize,
			maxLogSize:      defaultMaxLogSize,
			emit:            testEmitFunc(emitChan),
		},
		fromBeginning:   true,
		splitterFactory: newMultilineSplitterFactory(splitterConfig),
		encodingConfig:  splitterConfig.EncodingConfig,
	}, emitChan
}

func readToken(t *testing.T, c chan *emitParams) []byte {
	select {
	case call := <-c:
		return call.token
	case <-time.After(3 * time.Second):
		require.FailNow(t, "Timed out waiting for token")
	}
	return nil
}

func TestMapCopy(t *testing.T) {
	initMap := map[string]any{
		"mapVal": map[string]any{
			"nestedVal": "value1",
		},
		"intVal": 1,
		"strVal": "OrigStr",
	}

	copyMap := mapCopy(initMap)
	// Mutate values on the copied map
	copyMap["mapVal"].(map[string]any)["nestedVal"] = "overwrittenValue"
	copyMap["intVal"] = 2
	copyMap["strVal"] = "CopyString"

	// Assert that the original map should have the same values
	assert.Equal(t, "value1", initMap["mapVal"].(map[string]any)["nestedVal"])
	assert.Equal(t, 1, initMap["intVal"])
	assert.Equal(t, "OrigStr", initMap["strVal"])
}
