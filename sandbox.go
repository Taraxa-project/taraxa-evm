package main

import "fmt"

func Try(action func()) interface{} {
	return func() (recovered interface{}) {
		defer func() {
			recovered = recover()
		}()
		action()
		return
	}()
}

func ReadAll(chan) {

}

func main() {

	b := make(chan int, 1)
	close(b)
	fmt.Println(Try(func() {
		b <- 1
	}))
	fmt.Println(<-b)
}
