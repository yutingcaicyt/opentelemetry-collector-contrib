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

package mongodbatlasreceiver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap/confmaptest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mongodbatlasreceiver/internal/metadata"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name        string
		input       Config
		expectedErr string
	}{
		{
			name:  "Empty config",
			input: Config{},
		},
		{
			name: "Valid alerts config",
			input: Config{
				Alerts: AlertConfig{
					Enabled:  true,
					Endpoint: "0.0.0.0:7706",
					Secret:   "some_secret",
					Mode:     alertModeListen,
				},
			},
		},
		{
			name: "Alerts missing endpoint",
			input: Config{
				Alerts: AlertConfig{
					Enabled: true,
					Secret:  "some_secret",
					Mode:    alertModeListen,
				},
			},
			expectedErr: errNoEndpoint.Error(),
		},
		{
			name: "Alerts missing secret",
			input: Config{
				Alerts: AlertConfig{
					Enabled:  true,
					Endpoint: "0.0.0.0:7706",
					Mode:     alertModeListen,
				},
			},
			expectedErr: errNoSecret.Error(),
		},
		{
			name: "Invalid endpoint",
			input: Config{
				Alerts: AlertConfig{
					Enabled:  true,
					Endpoint: "7706",
					Secret:   "some_secret",
					Mode:     alertModeListen,
				},
			},
			expectedErr: "failed to split endpoint into 'host:port' pair",
		},
		{
			name: "TLS config missing key",
			input: Config{
				Alerts: AlertConfig{
					Enabled:  true,
					Endpoint: "0.0.0.0:7706",
					Secret:   "some_secret",
					Mode:     alertModeListen,
					TLS: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "some_cert_file",
						},
					},
				},
			},
			expectedErr: errNoKey.Error(),
		},
		{
			name: "TLS config missing cert",
			input: Config{
				Alerts: AlertConfig{
					Enabled:  true,
					Endpoint: "0.0.0.0:7706",
					Secret:   "some_secret",
					Mode:     alertModeListen,
					TLS: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							KeyFile: "some_key_file",
						},
					},
				},
			},
			expectedErr: errNoCert.Error(),
		},
		{
			name: "Valid Logs Config",
			input: Config{
				Logs: LogConfig{
					Enabled: true,
					Projects: []*ProjectConfig{
						{
							Name:            "Project1",
							EnableAuditLogs: false,
						},
					},
				},
			},
		},
		{
			name: "Invalid Logs Config",
			input: Config{
				Logs: LogConfig{
					Enabled: true,
				},
			},
			expectedErr: errNoProjects.Error(),
		},
		{
			name: "Invalid ProjectConfig",
			input: Config{
				Logs: LogConfig{
					Enabled: true,
					Projects: []*ProjectConfig{
						{
							Name:            "Project1",
							EnableAuditLogs: false,
							ExcludeClusters: []string{"cluster1"},
							IncludeClusters: []string{"cluster2"},
						},
					},
				},
			},
			expectedErr: errClusterConfig.Error(),
		},
		{
			name: "Invalid Alerts Retrieval ProjectConfig",
			input: Config{
				Alerts: AlertConfig{
					Enabled: true,
					Mode:    alertModePoll,
					Projects: []*ProjectConfig{
						{
							Name:            "Project1",
							EnableAuditLogs: false,
							ExcludeClusters: []string{"cluster1"},
							IncludeClusters: []string{"cluster2"},
						},
					},
					PageSize: defaultAlertsPageSize,
				},
			},
			expectedErr: errClusterConfig.Error(),
		},
		{
			name: "Invalid Alerts Poll No Projects",
			input: Config{
				Alerts: AlertConfig{
					Enabled:  true,
					Mode:     alertModePoll,
					Projects: []*ProjectConfig{},
					PageSize: defaultAlertsPageSize,
				},
			},
			expectedErr: errNoProjects.Error(),
		},
		{
			name: "Valid Alerts Config",
			input: Config{
				Alerts: AlertConfig{
					Enabled: true,
					Mode:    alertModePoll,
					Projects: []*ProjectConfig{
						{
							Name: "Project1",
						},
					},
					PageSize: defaultAlertsPageSize,
				},
			},
		},
		{
			name: "Invalid Alerts Mode",
			input: Config{
				Alerts: AlertConfig{
					Enabled:  true,
					Mode:     "invalid type",
					Projects: []*ProjectConfig{},
				},
			},
			expectedErr: errNoModeRecognized.Error(),
		},
		{
			name: "Invalid Page Size",
			input: Config{
				Alerts: AlertConfig{
					Enabled: true,
					Mode:    alertModePoll,
					Projects: []*ProjectConfig{
						{
							Name: "Test",
						},
					},
					PageSize: -1,
				},
			},
			expectedErr: errPageSizeIncorrect.Error(),
		},
		{
			name: "Invalid events config - no projects",
			input: Config{
				Events: &EventsConfig{
					Projects: []*ProjectConfig{},
				},
			},
			expectedErr: errNoEvents.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(component.NewIDWithName(typeStr, "").String())
	require.NoError(t, err)
	require.NoError(t, component.UnmarshalConfig(sub, cfg))

	expected := factory.CreateDefaultConfig().(*Config)
	expected.MetricsBuilderConfig = metadata.DefaultMetricsBuilderConfig()
	expected.PrivateKey = "my-private-key"
	expected.PublicKey = "my-public-key"
	expected.Logs = LogConfig{
		Enabled: true,
		Projects: []*ProjectConfig{
			{
				Name: "Project 0",
			},
		},
	}
	expected.Alerts = AlertConfig{
		Enabled: true,
		Mode:    alertModePoll,
		Projects: []*ProjectConfig{
			{
				Name:            "Project 0",
				IncludeClusters: []string{"Cluster0"},
			},
		},
		PageSize:     defaultAlertsPageSize,
		MaxPages:     defaultAlertsMaxPages,
		PollInterval: time.Minute,
	}

	expected.Events = &EventsConfig{
		Projects: []*ProjectConfig{
			{
				Name: "Project 0",
			},
		},
		Organizations: []*OrgConfig{
			{
				ID: "5b478b3afc4625789ce616a3",
			},
		},
		PollInterval: time.Minute,
		MaxPages:     defaultEventsMaxPages,
		PageSize:     defaultEventsPageSize,
	}
	require.Equal(t, expected, cfg)
}
