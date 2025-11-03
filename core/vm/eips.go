package vm

// fixEIP1153 fixes EIP-1153
// https://eips.ethereum.org/EIPS/eip-1153
func fixEIP1153(jt *InstructionSet) {
	jt[TLOAD] = &operation{
		execute:       opTload,
		gasCost:       gasTLoad,
		validateStack: makeStackFunc(1, 1),
	}

	jt[TSTORE] = &operation{
			execute:       opTstore,
			gasCost:       gasTLoad,
			validateStack: makeStackFunc(2, 0),
	}
}

// enable5656 enables EIP-5656 (MCOPY opcode)
// https://eips.ethereum.org/EIPS/eip-5656
func enable5656(jt *InstructionSet) {
	jt[MCOPY] = &operation{
		execute:       opMcopy,
		gasCost:       gasMcopy,
		validateStack: makeStackFunc(3, 0),
		memorySize:    memoryMcopy,
	}
}

// opMcopy implements the MCOPY opcode (https://eips.ethereum.org/EIPS/eip-5656)
func opMcopy(pc *uint64, evm *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		dst    = stack.pop()
		src    = stack.pop()
		length = stack.pop()
	)
	// These values are checked for overflow during memory expansion calculation
	// (the memorySize function on the opcode).
	memory.Copy(dst.Uint64(), src.Uint64(), length.Uint64())
	return nil, nil
}
