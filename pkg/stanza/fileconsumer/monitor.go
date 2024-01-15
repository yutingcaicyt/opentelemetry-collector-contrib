// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package fileconsumer // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
)

const (
	// default as 1s
	defaultMonitorMaxDelay = 1000 * 1000 * 1000
)

// Monitor does two things:
// 1.Check whether some files have not been read within a certain time interval
// 2.Check the read delay when a reader reads the file
type MonitorManager struct {
	curPollStartTime time.Time
	// key: filePath, value: missCheckInfo
	fileMissCheckMap *sync.Map
	checkStart       sync.Once
	checkInterval    time.Duration
	maxDelay         time.Duration
	telemetry        *fileConsumerTelemetry
	// This channel is for reusing match files found in a poll
	matchFilesChan chan []string
}

type MonitorConfig struct {
	// Enabled indicates whether to not monitor the delay of file reading
	Enabled bool `mapstructure:"enabled, omitempty"`
	// MaxDelay means once the delay of file reading reaches the value, a log will be recorded
	MaxDelay time.Duration `mapstructure:"max_delay,omitempty"`
}

func (c *Config) telemetryInitialization(buildInfo *operator.BuildInfoInternal) *fileConsumerTelemetry {
	if buildInfo.CreateSettings == nil {
		return nil
	}
	telemetry, err := newFileConsumerTelemetry(buildInfo.CreateSettings, buildInfo.TelemetryUseOtel)
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
	m.monitorManager.fileMissCheckMap.Range(func(key, value interface{}) bool {
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
		m.monitorManager.telemetry.record(delay)
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
		matches = m.finder.FindFiles()
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

type ReaderDelayCheck struct {
	startPollTime    time.Time
	fileMissCheckMap *sync.Map
	telemetry        *fileConsumerTelemetry
	maxDelay         time.Duration
}

func (r *Reader) delayCheck(preFileSize int64, preTime time.Time) (int64, time.Time) {
	// the offset reaches the file size recorded before, record the delay
	if r.readerDelayCheck.telemetry == nil {
		return preFileSize, preTime
	}
	var curTime = time.Now()
	r.readerDelayCheck.fileMissCheckMap.Delete(r.file.Name())
	if preFileSize >= 0 && r.Offset >= preFileSize {
		delay := curTime.Sub(preTime).Milliseconds()
		if r.readerDelayCheck.maxDelay.Milliseconds() <= delay {
			r.Warnw("file "+r.file.Name()+"read delay is high",
				"delayTime(ms)", delay)
		}
		r.readerDelayCheck.telemetry.record(delay)
		preTime = curTime
		preFileSize = r.getFileSize()
	}
	return preFileSize, preTime
}

func (r *Reader) cancelMonitor(filePath string) {
	if r.readerDelayCheck.telemetry == nil {
		return
	}
	r.readerDelayCheck.fileMissCheckMap.Delete(filePath)
}

func (r *Reader) prepareBeforeDelayCheck() (int64, time.Time) {
	if r.readerDelayCheck.telemetry == nil {
		return 0, time.Time{}
	}
	return r.getFileSize(), r.readerDelayCheck.startPollTime
}
func (r *Reader) getFileSize() int64 {
	info, err := r.file.Stat()
	if err != nil {
		r.Errorw("Failed to get stat", zap.Error(err))
		return -1
	}
	return info.Size()
}
