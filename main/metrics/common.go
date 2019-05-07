package metrics

import "time"

type OnDuration func(duration time.Duration)

func NewTimeRecorder(callback OnDuration) func() {
	start := time.Now()
	return func() {
		callback(time.Since(start))
	}
}
