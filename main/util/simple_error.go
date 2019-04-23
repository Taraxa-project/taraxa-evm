package util

type SimpleError string

func (this *SimpleError) Error() string {
	return string(*this)
}
