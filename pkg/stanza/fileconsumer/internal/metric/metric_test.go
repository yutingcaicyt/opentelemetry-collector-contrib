// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileLogReceiverMetrics(t *testing.T) {
	viewNames := []string{
		"file_read_delay",
	}
	views := metricViews()
	for i, viewName := range viewNames {
		assert.Equal(t, "receiver_filelog_"+viewName, views[i].Name)
	}
}
