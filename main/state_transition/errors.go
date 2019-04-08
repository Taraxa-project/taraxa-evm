package state_transition

type ErrorCode uint64

const (

)

type StateTransitionError struct {
	uint64 code
	Error error
}