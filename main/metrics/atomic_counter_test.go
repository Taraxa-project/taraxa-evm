package metrics

import (
	"fmt"
	"testing"
)

func Test(*testing.T) {
	var c AtomicCounter
	c.Add(10)
	fmt.Println(c + 1)
}
