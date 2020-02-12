package state

type State = interface {
	CommitBlock(state_change StateChange) (block_ordinal BlockOrdinal, digest []byte, err error)
	Get(block_ordinal BlockOrdinal, k []byte) ([]byte, error)
	GetWithProof(block_ordinal BlockOrdinal, k []byte) (ValueProof, error)
	Close()
}

type BlockOrdinal = uint64

type StateEntry = struct {
	K, V []byte
}

type StateChange = []StateEntry

type ValueProof = interface {
	Verify(digest, key []byte) (value []byte, err error)
}
