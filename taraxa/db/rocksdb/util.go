package rocksdb

import "github.com/tecbot/gorocksdb"

type neverOverwrite uint8

var NeverOverwrite gorocksdb.MergeOperator = new(neverOverwrite)

func (this *neverOverwrite) FullMerge(key, existingValue []byte, operands [][]byte) ([]byte, bool) {
	if existingValue != nil || len(operands) == 0 {
		return nil, false
	}
	return operands[0], true
}

func (this *neverOverwrite) PartialMerge(key, leftOperand, rightOperand []byte) ([]byte, bool) {
	return nil, false
}

func (this *neverOverwrite) Name() string {
	return "neverOverwrite"
}
