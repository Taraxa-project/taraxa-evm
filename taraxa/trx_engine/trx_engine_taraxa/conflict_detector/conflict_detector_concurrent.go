package conflict_detector

type ConcurrentConflictDetector struct {
	operationIndex    OperationIndex
	keysInConflict    Keys
	authorsInConflict Authors
}

func NewConcurrentConflictDetector() *ConcurrentConflictDetector {
	return &ConcurrentConflictDetector{make(OperationIndex), make(Keys), make(Authors)}
}

func (this *ConcurrentConflictDetector) Process(op *Operation) (conflicts *AuthorsByOperation) {
	if this.authorsInConflict[op.Author] {
		return
	}
	if this.keysInConflict[op.Key] {
		this.authorsInConflict[op.Author] = true
		return
	}
	authorsByOp := this.operationIndex[op.Key]
	if authorsByOp == nil {
		authorsByOp = new(AuthorsByOperation)
		authorsByOp[op.Type] = Authors{op.Author: true}
		this.operationIndex[op.Key] = authorsByOp
		return
	}
	for _, conflictingOp := range conflictRelations[op.Type] {
		if authors := authorsByOp[conflictingOp]; len(authors) == 0 || len(authors) == 1 && authors[op.Author] {
			continue
		}
		this.keysInConflict[op.Key] = true
		this.authorsInConflict[op.Key] = true
		for _, authors := range authorsByOp {
			for author := range authors {
				this.authorsInConflict[author] = true
			}
		}
		// TODO defensive copy or remove ?
		return authorsByOp
	}
	if authors := authorsByOp[op.Type]; authors != nil {
		authors[op.Author] = true
	} else {
		authorsByOp[op.Type] = Authors{op.Author: true}
	}
	return
}
