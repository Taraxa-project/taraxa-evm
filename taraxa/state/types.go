package state

type TxIndex = uint

type ConcurrentSchedule = struct {
	ParallelTransactions []TxIndex
}
