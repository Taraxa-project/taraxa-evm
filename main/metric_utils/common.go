package metric_utils

import "time"

type OnDuration func(duration time.Duration)

func NewTimeRecorder(callbacks ...OnDuration) func() time.Duration {
	start := time.Now()
	return func() time.Duration {
		duration := time.Since(start)
		for _, cb := range callbacks {
			cb(duration)
		}
		return duration
	}
}
