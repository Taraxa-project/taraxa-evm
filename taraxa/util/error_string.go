package util

type ErrorString string

func (this *ErrorString) Error() string {
	return string(*this)
}
