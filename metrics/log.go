package metrics

type Logger interface {
	Printf(format string, v ...interface{})
}

// Output each metric in the given registry periodically using the given
// logger. Print timings in `scale` units (eg time.Millisecond) rather than nanos.
