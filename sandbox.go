package main

import "fmt"

type s struct {
	i uint64
	v *uint64
}

func main()  {

	m := make(map[string]s)
	v := m["foo"]
	fmt.Println(v)
}