package conflict_detector

import "github.com/emirpasic/gods/sets/linkedhashset"

type OperationType int

const (
	GET OperationType = iota
	SET
	ADD //commutative
	// TODO consider operands of the operators
	DEFAULT_INITIALIZE //idempotent
	OperationType_count uint = iota
)

type Author = interface{} // equals/hashcode required
type Authors = *linkedhashset.Set
type Key = string // equals/hashcode required
type Keys = *linkedhashset.Set
type ConflictRelations = map[OperationType][]OperationType
type OperationLogger func(OperationType, Key)
type Operation struct {
	Author Author
	Type   OperationType
	Key    Key
}
type OnConflict func(*ConflictDetector, *Operation, Authors)
type operationLog = []map[Key]Authors

var conflictRelations = func() ConflictRelations {
	ret := make(ConflictRelations)
	inConflict := func(left, right OperationType) {
		ret[left] = append(ret[left], right)
		ret[right] = append(ret[right], left)
	}
	inConflict(GET, SET)
	inConflict(GET, ADD)
	inConflict(GET, DEFAULT_INITIALIZE)
	inConflict(SET, SET)
	inConflict(SET, ADD)
	inConflict(SET, DEFAULT_INITIALIZE)
	inConflict(ADD, DEFAULT_INITIALIZE)
	return ret
}()
