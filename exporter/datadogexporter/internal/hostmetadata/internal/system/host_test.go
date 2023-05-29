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

package system

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetHostInfo(t *testing.T) {
	logger := zap.NewNop()

	hostInfo := GetHostInfo(logger)
	require.NotNil(t, hostInfo)

	osHostname, err := os.Hostname()
	require.NoError(t, err)
	assert.Equal(t, hostInfo.OS, osHostname)
}

func TestGetHostname(t *testing.T) {
	logger := zap.NewNop()

	hostInfoAll := &HostInfo{
		FQDN: "fqdn",
		OS:   "os",
	}
	assert.Equal(t, hostInfoAll.GetHostname(logger), "fqdn")

	hostInfoInvalid := &HostInfo{
		FQDN: "fqdn_invalid",
		OS:   "os",
	}
	assert.Equal(t, hostInfoInvalid.GetHostname(logger), "os")

	hostInfoMissingFQDN := &HostInfo{
		OS: "os",
	}
	assert.Equal(t, hostInfoMissingFQDN.GetHostname(logger), "os")

}
