package conflict_detector

import "github.com/emirpasic/gods/sets/linkedhashset"

type OperationType int

const (
	GET OperationType = iota
	SET
	ADD  //commutative
	// TODO consider operands of the operators
	DEFAULT_INITIALIZE  //idempotent
	OperationType_count uint = iota
)

type Author = interface{} // equals/hashcode required
type Authors = *linkedhashset.Set
type Key = string // equals/hashcode required
type Keys = *linkedhashset.Set
type OperationLogger func(OperationType, Key)

type conflictRelationsMap = map[OperationType][]OperationType

var conflictRelations = func() conflictRelationsMap {
	ret := make(conflictRelationsMap)
	inConflict := func(left, right OperationType) {
		ret[left] = append(ret[left], right)
		ret[right] = append(ret[right], left)
	}
	inConflict(GET, SET)
	inConflict(GET, ADD)
	inConflict(SET, SET)
	inConflict(SET, ADD)
	inConflict(DEFAULT_INITIALIZE, GET)
	inConflict(DEFAULT_INITIALIZE, ADD)
	inConflict(DEFAULT_INITIALIZE, SET)
	return ret
}()

type Operation struct {
	Author Author
	Type   OperationType
	Key    Key
}
type operationLog = []map[Key]Authors

type OperationLoggerFactory func(Author) OperationLogger

var NoopLogger OperationLogger = func(operationType OperationType, key Key) {}

var NoopLoggerFactory OperationLoggerFactory = func(author Author) OperationLogger {
	return NoopLogger
}
