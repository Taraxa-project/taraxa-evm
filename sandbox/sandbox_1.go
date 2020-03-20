package main

import (
	"errors"
	"fmt"
	"time"
)

func f1_panic() (ret time.Time) {
	defer func() {
		ret = time.Now()
		if r := recover(); r != nil {
		}
	}()
	f2_panic()
	ret = time.Now()
	return
}

func f2_panic() {
	f3_panic()
}

func f3_panic() {
	f4_panic()
}

func f4_panic() {
	f5_panic()
}

func f5_panic() {
	panic(errors.New("foo"))
}

func f1() (ret time.Time) {
	if e := f2(); e != nil {
	}
	ret = time.Now()
	return
}

func f2() error {
	if err := f3(); err != nil {
		return err
	}
	return nil
}

func f3() error {
	if err := f4(); err != nil {
		return err
	}
	return nil
}

func f4() error {
	if err := f5(); err != nil {
		return err
	}
	return nil
}

func f5() error {
	return errors.New("foo")
}

func main() {
	N := time.Duration(10000000)

	var elapsed time.Duration
	for i := time.Duration(0); i < N; i++ {
		start := time.Now()
		finish := f1_panic()
		elapsed += finish.Sub(start)
	}
	fmt.Println(elapsed / N)
	elapsed = 0

	for i := time.Duration(0); i < N; i++ {
		start := time.Now()
		finish := f1()
		elapsed += finish.Sub(start)
	}
	fmt.Println(elapsed / N)
	elapsed = 0
}
