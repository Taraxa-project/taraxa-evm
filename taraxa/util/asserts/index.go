package asserts

import (
	"fmt"
	"strings"
)

func EQ(a, b interface{}) (ret bool) {
	if a != b {
		panic(fmt.Sprint(a, " != ", b))
	}
	return
}

func Holds(condition bool, msg ...string) (ret bool) {
	if ret = condition; !ret {
		if len(msg) == 0 {
			panic("assertion error")
		}
		panic(strings.Join(msg, " "))
	}
	return
}
