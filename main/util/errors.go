package util

func RecoverErr(callback func(error)) {
	if recovered := recover(); recovered != nil {
		if err, isErr := recovered.(error); isErr {
			callback(err)
		} else {
			panic(recovered)
		}
	}
}

func FailOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
