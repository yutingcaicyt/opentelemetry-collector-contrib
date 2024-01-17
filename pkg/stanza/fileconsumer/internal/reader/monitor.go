// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package reader

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer/internal/metric"
)

type DelayCheck struct {
	StartPollTime    time.Time
	FileMissCheckMap *sync.Map
	Telemetry        *metric.FileConsumerTelemetry
	MaxDelay         time.Duration
}

func (r *Reader) delayCheck(preFileSize int64, preTime time.Time) (int64, time.Time) {
	// the offset reaches the file size recorded before, record the delay
	if r.ReaderDelayCheck.Telemetry == nil {
		return preFileSize, preTime
	}
	var curTime = time.Now()
	r.ReaderDelayCheck.FileMissCheckMap.Delete(r.file.Name())
	if preFileSize >= 0 && r.Offset >= preFileSize {
		delay := curTime.Sub(preTime).Milliseconds()
		if r.ReaderDelayCheck.MaxDelay.Milliseconds() <= delay {
			r.logger.Warnw("file "+r.file.Name()+"read delay is high",
				"delayTime(ms)", delay)
		}
		r.ReaderDelayCheck.Telemetry.Record(delay)
		preTime = curTime
		preFileSize = r.getFileSize()
	}
	return preFileSize, preTime
}

func (r *Reader) cancelMonitor(filePath string) {
	if r.ReaderDelayCheck.Telemetry == nil {
		return
	}
	r.ReaderDelayCheck.FileMissCheckMap.Delete(filePath)
}

func (r *Reader) prepareBeforeDelayCheck() (int64, time.Time) {
	if r.ReaderDelayCheck.Telemetry == nil {
		return 0, time.Time{}
	}
	return r.getFileSize(), r.ReaderDelayCheck.StartPollTime
}
func (r *Reader) getFileSize() int64 {
	info, err := r.file.Stat()
	if err != nil {
		r.logger.Errorw("Failed to get stat", zap.Error(err))
		return -1
	}
	return info.Size()
}
