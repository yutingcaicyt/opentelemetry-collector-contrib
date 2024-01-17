// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fileconsumer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer/internal/metric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
)

const (
	// default as 1s
	defaultMonitorMaxDelay = 1000 * 1000 * 1000
)

// MonitorManager manages two things:
// 1.Check whether some files have not been read within a certain time interval
// 2.Check the read delay when a reader reads the file
type MonitorManager struct {
	curPollStartTime time.Time
	// key: filePath, value: missCheckInfo
	fileMissCheckMap *sync.Map
	checkStart       sync.Once
	checkInterval    time.Duration
	maxDelay         time.Duration
	telemetry        *metric.FileConsumerTelemetry
	// This channel is for reusing match files found in a poll
	matchFilesChan chan []string
}

type MonitorConfig struct {
	// Enabled indicates whether to not monitor the delay of file reading
	Enabled bool `mapstructure:"enabled, omitempty"`
	// MaxDelay means once the delay of file reading reaches the value, a log will be recorded
	MaxDelay time.Duration `mapstructure:"max_delay,omitempty"`
}

func (c *Config) telemetryInitialization(buildInfo *operator.BuildInfoInternal) *metric.FileConsumerTelemetry {
	if buildInfo.CreateSettings == nil {
		return nil
	}
	telemetry, err := metric.NewFileConsumerTelemetry(buildInfo.CreateSettings, buildInfo.TelemetryUseOtel)
	if err != nil {
		_ = fmt.Errorf("failed to create fileConsumer telemetry: %w", err)
	}
	return telemetry
}

func (c *Config) newMonitorManager(buildInfo *operator.BuildInfoInternal) *MonitorManager {
	var monitorManager *MonitorManager
	if c.Monitor.Enabled {
		monitorManager = &MonitorManager{
			fileMissCheckMap: &sync.Map{},
			checkInterval:    c.PollInterval,
			telemetry:        c.telemetryInitialization(buildInfo),
			maxDelay:         defaultMonitorMaxDelay,
			matchFilesChan:   make(chan []string, 1),
		}
		if c.Monitor.MaxDelay > 0 {
			monitorManager.maxDelay = c.Monitor.MaxDelay
		}
	}
	return monitorManager
}

// This info records how many consecutive checks a file has not been read
type missCheckInfo struct {
	missedTimes int
}

func (m *Manager) fileMissCheck() {

	matches := m.getMatchFiles()

	tmpMap := map[string]struct{}{}
	for _, v := range matches {
		tmpMap[v] = struct{}{}
	}
	m.monitorManager.fileMissCheckMap.Range(func(key, value any) bool {
		if _, ok := tmpMap[key.(string)]; !ok {
			m.monitorManager.fileMissCheckMap.Delete(key)
			return true
		}
		checkInfo := value.(*missCheckInfo)
		if checkInfo.missedTimes == 0 {
			checkInfo.missedTimes++
			return true
		}
		delay := m.monitorManager.checkInterval.Milliseconds() * int64(checkInfo.missedTimes)
		m.monitorManager.telemetry.Record(delay)
		// If delay reaches the value of MaxDelay, recording a log
		if delay >= m.monitorManager.maxDelay.Milliseconds() {
			m.Warnw("file "+key.(string)+" missed or read delay is high",
				"missedTime(ms)", delay)
		}
		checkInfo.missedTimes++
		return true
	})
	for _, path := range matches {
		if _, ok := m.monitorManager.fileMissCheckMap.Load(path); !ok {
			m.monitorManager.fileMissCheckMap.Store(path, &missCheckInfo{missedTimes: 0})
		}
	}
}

func (m *Manager) startCheck(ctx context.Context) {
	if m.monitorManager.telemetry == nil {
		return
	}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		checkTicker := time.NewTicker(m.monitorManager.checkInterval)
		defer checkTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-checkTicker.C:
			}
			m.fileMissCheck()
		}
	}()
}

func (m *Manager) syncMatchFiles(matchFiles []string) {
	select {
	case m.monitorManager.matchFilesChan <- matchFiles:
	default:
	}
}

func (m *Manager) getMatchFiles() []string {
	var matches []string

	select {
	case matches = <-m.monitorManager.matchFilesChan:
	default:
		var err error
		if matches, err = m.fileMatcher.MatchFiles(); err != nil {
			m.Warnf("finding files: %v", err)
		}
	}
	return matches
}

func (m *Manager) monitorStart(ctx context.Context, pollStartTime time.Time, matchFiles []string) {
	if m.monitorManager == nil {
		return
	}
	m.monitorManager.curPollStartTime = pollStartTime
	m.syncMatchFiles(matchFiles)
	m.monitorManager.checkStart.Do(func() {
		m.startCheck(ctx)
	})
}

func (m *Manager) cancelMonitor(filePath string) {
	if m.monitorManager == nil {
		return
	}
	m.monitorManager.fileMissCheckMap.Delete(filePath)
}
