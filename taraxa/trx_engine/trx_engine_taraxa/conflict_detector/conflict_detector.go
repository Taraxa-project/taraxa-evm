package conflict_detector

type OperationType byte

const (
	GET OperationType = iota
	SET
	ADD //commutative
	// TODO consider arguments of the operations
	OperationType_count
)

const DELETE = SET

var conflictRelations = func() (ret [OperationType_count][]OperationType) {
	conflicting := func(left, right OperationType) {
		ret[left] = append(ret[left], right)
		ret[right] = append(ret[right], left)
	}
	conflicting(GET, SET)
	conflicting(GET, ADD)
	conflicting(SET, SET)
	conflicting(SET, ADD)
	return
}()

type Author = interface{}
type Authors = map[Author]bool
type Key = interface{}
type Keys = map[Key]bool
type Operation = struct {
	Author Author
	Type   OperationType
	Key    Key
}
type AuthorsByOperation = [OperationType_count]Authors
type OperationIndex = map[Key]*AuthorsByOperation

type ConflictDetector struct {
	OperationIndex    OperationIndex
	KeysInConflict    Keys
	AuthorsInConflict Authors
}

func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{make(OperationIndex), make(Keys), make(Authors)}
}

func (this *ConflictDetector) Process(op *Operation) (conflicts *AuthorsByOperation, hasCaused bool) {
	if this.AuthorsInConflict[op.Author] {
		return
	}
	if this.KeysInConflict[op.Key] {
		this.AuthorsInConflict[op.Author] = true
		return this.OperationIndex[op.Key], false
	}
	authorsByOp := this.OperationIndex[op.Key]
	if authorsByOp == nil {
		authorsByOp = new(AuthorsByOperation)
		authorsByOp[op.Type] = Authors{op.Author: true}
		this.OperationIndex[op.Key] = authorsByOp
		return
	}
	for _, conflictingOp := range conflictRelations[op.Type] {
		if authors := authorsByOp[conflictingOp]; len(authors) == 0 || len(authors) == 1 && authors[op.Author] {
			continue
		}
		this.KeysInConflict[op.Key] = true
		this.AuthorsInConflict[op.Key] = true
		for _, authors := range authorsByOp {
			for author := range authors {
				this.AuthorsInConflict[author] = true
			}
		}
		// TODO defensive copy or remove ?
		return authorsByOp, true
	}
	if authors := authorsByOp[op.Type]; authors != nil {
		authors[op.Author] = true
	} else {
		authorsByOp[op.Type] = Authors{op.Author: true}
	}
	return
}
