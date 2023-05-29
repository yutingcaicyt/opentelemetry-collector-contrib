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

package bearertokenauthextension

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap/zaptest"
)

func TestPerRPCAuth(t *testing.T) {
	metadata := map[string]string{
		"authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
	}

	// test meta data is properly
	perRPCAuth := &PerRPCAuth{metadata: metadata}
	md, err := perRPCAuth.GetRequestMetadata(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, md, metadata)

	// always true
	ok := perRPCAuth.RequireTransportSecurity()
	assert.True(t, ok)
}

type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{}}
	for k, v := range req.Header {
		resp.Header.Set(k, v[0])
	}
	return resp, nil
}

func TestBearerAuthenticatorHttp(t *testing.T) {
	scheme := "TestScheme"
	cfg := createDefaultConfig().(*Config)
	cfg.BearerToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
	cfg.Scheme = scheme

	bauth := newBearerTokenAuth(cfg, nil)
	assert.NotNil(t, bauth)

	base := &mockRoundTripper{}
	c, err := bauth.RoundTripper(base)
	assert.NoError(t, err)
	assert.NotNil(t, c)

	request := &http.Request{Method: "Get"}
	resp, err := c.RoundTrip(request)
	assert.NoError(t, err)
	authHeaderValue := resp.Header.Get("Authorization")
	assert.Equal(t, authHeaderValue, fmt.Sprintf("%s %s", scheme, cfg.BearerToken))

}

func TestBearerAuthenticator(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.BearerToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

	bauth := newBearerTokenAuth(cfg, nil)
	assert.NotNil(t, bauth)

	assert.Nil(t, bauth.Start(context.Background(), componenttest.NewNopHost()))
	credential, err := bauth.PerRPCCredentials()

	assert.NoError(t, err)
	assert.NotNil(t, credential)

	md, err := credential.GetRequestMetadata(context.Background())
	expectedMd := map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", cfg.BearerToken),
	}
	assert.Equal(t, md, expectedMd)
	assert.NoError(t, err)
	assert.True(t, credential.RequireTransportSecurity())

	roundTripper, _ := bauth.RoundTripper(&mockRoundTripper{})
	orgHeaders := http.Header{
		"Foo": {"bar"},
	}
	expectedHeaders := http.Header{
		"Foo":           {"bar"},
		"Authorization": {bauth.bearerToken()},
	}

	resp, err := roundTripper.RoundTrip(&http.Request{Header: orgHeaders})
	assert.NoError(t, err)
	assert.Equal(t, expectedHeaders, resp.Header)
	assert.Nil(t, bauth.Shutdown(context.Background()))
}

func TestBearerStartWatchStop(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Filename = "test.token"

	bauth := newBearerTokenAuth(cfg, zaptest.NewLogger(t))
	assert.NotNil(t, bauth)

	assert.Nil(t, bauth.Start(context.Background(), componenttest.NewNopHost()))
	assert.Error(t, bauth.Start(context.Background(), componenttest.NewNopHost()))

	credential, err := bauth.PerRPCCredentials()
	assert.NoError(t, err)
	assert.NotNil(t, credential)

	token, err := os.ReadFile(bauth.filename)
	assert.NoError(t, err)

	tokenStr := fmt.Sprintf("Bearer %s", token)
	md, err := credential.GetRequestMetadata(context.Background())
	expectedMd := map[string]string{
		"authorization": tokenStr,
	}
	assert.Equal(t, md, expectedMd)
	assert.NoError(t, err)
	assert.True(t, credential.RequireTransportSecurity())

	// change file content once
	assert.Nil(t, os.WriteFile(bauth.filename, []byte(fmt.Sprintf("%stest", token)), 0600))
	time.Sleep(5 * time.Second)
	credential, _ = bauth.PerRPCCredentials()
	md, err = credential.GetRequestMetadata(context.Background())
	expectedMd["authorization"] = tokenStr + "test"
	assert.Equal(t, md, expectedMd)
	assert.NoError(t, err)

	// change file content back
	assert.Nil(t, os.WriteFile(bauth.filename, token, 0600))
	time.Sleep(5 * time.Second)
	credential, _ = bauth.PerRPCCredentials()
	md, err = credential.GetRequestMetadata(context.Background())
	expectedMd["authorization"] = tokenStr
	time.Sleep(5 * time.Second)
	assert.Equal(t, md, expectedMd)
	assert.NoError(t, err)

	assert.Nil(t, bauth.Shutdown(context.Background()))
	assert.Nil(t, bauth.shutdownCH)
}

func TestBearerTokenFileContentUpdate(t *testing.T) {
	scheme := "TestScheme"
	cfg := createDefaultConfig().(*Config)
	cfg.Filename = "test.token"
	cfg.Scheme = scheme

	bauth := newBearerTokenAuth(cfg, zaptest.NewLogger(t))
	assert.NotNil(t, bauth)

	assert.Nil(t, bauth.Start(context.Background(), componenttest.NewNopHost()))
	assert.Error(t, bauth.Start(context.Background(), componenttest.NewNopHost()))

	token, err := os.ReadFile(bauth.filename)
	assert.NoError(t, err)

	base := &mockRoundTripper{}
	rt, err := bauth.RoundTripper(base)
	assert.NoError(t, err)
	assert.NotNil(t, rt)

	request := &http.Request{Method: "Get"}
	resp, err := rt.RoundTrip(request)
	assert.NoError(t, err)
	authHeaderValue := resp.Header.Get("Authorization")
	assert.Equal(t, authHeaderValue, fmt.Sprintf("%s %s", scheme, string(token)))

	// change file content once
	assert.Nil(t, os.WriteFile(bauth.filename, []byte(fmt.Sprintf("%stest", token)), 0600))
	time.Sleep(5 * time.Second)

	tokenNew, err := os.ReadFile(bauth.filename)
	assert.NoError(t, err)

	// check if request is updated with the new token
	request = &http.Request{Method: "Get"}
	resp, err = rt.RoundTrip(request)
	assert.NoError(t, err)
	authHeaderValue = resp.Header.Get("Authorization")
	assert.Equal(t, authHeaderValue, fmt.Sprintf("%s %s", scheme, string(tokenNew)))

	// change file content back
	assert.Nil(t, os.WriteFile(bauth.filename, token, 0600))
	time.Sleep(5 * time.Second)

	// check if request is updated with the old token
	request = &http.Request{Method: "Get"}
	resp, err = rt.RoundTrip(request)
	assert.NoError(t, err)
	authHeaderValue = resp.Header.Get("Authorization")
	assert.Equal(t, authHeaderValue, fmt.Sprintf("%s %s", scheme, string(token)))
}
