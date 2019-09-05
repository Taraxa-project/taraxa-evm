package metrics

// Write sorts writes each metric in the given registry periodically to the
// given io.Writer.

// WriteOnce sorts and writes metrics in the given registry to the given
// io.Writer.

type namedMetric struct {
	name string
	m    interface{}
}

// namedMetricSlice is a slice of namedMetrics that implements sort.Interface.
type namedMetricSlice []namedMetric

func (nms namedMetricSlice) Len() int { return len(nms) }

func (nms namedMetricSlice) Swap(i, j int) { nms[i], nms[j] = nms[j], nms[i] }

func (nms namedMetricSlice) Less(i, j int) bool {
	return nms[i].name < nms[j].name
}
