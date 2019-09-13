package util

const (
	Offset32        = 2166136261
	Offset64        = 14695981039346656037
	Offset128Lower  = 0x62b821756295c58d
	Offset128Higher = 0x6c62272e07bb0142
	Prime32         = 16777619
	Prime64         = 1099511628211
)

func AppendFNV64(source, val uint64) uint64 {
	if source == 0 {
		source = Offset64
	}
	return (source * Prime64) ^ val
}

func FNV64(str string) (ret uint64) {
	ret = Offset64
	for _, rune := range str {
		ret = AppendFNV64(ret, uint64(rune))
	}
	return
}
