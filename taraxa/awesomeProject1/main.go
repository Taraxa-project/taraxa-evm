package main

import (
	"bytes"
	"fmt"
)

func main() {
	fmt.Println(bytes.Compare([]byte{1, 2}, []byte{0, 2}))
	fmt.Println([]byte(nil)[:8])
}
