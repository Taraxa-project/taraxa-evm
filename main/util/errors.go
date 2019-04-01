package util

import (
	"errors"
	"runtime/debug"
)

func Catch(callback func(error)) {
	if recovered := recover(); recovered != nil {
		if err, isErr := recovered.(error); isErr {
			callback(errors.New(err.Error() + ". Stack trace:\n" + string(debug.Stack())))
		} else {
			panic(recovered)
		}
	}
}

func PanicOn(err error) {
	if err != nil {
		panic(err)
	}
}
