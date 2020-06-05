package assert

import "fmt"

func EQ(a, b interface{}) (ret bool) {
	if a != b {
		panic(fmt.Sprint(a, " != ", b))
	}
	return
}

func Holds(condition bool) (ret bool) {
	if ret = condition; !ret {
		panic("assertion error")
	}
	return
}
