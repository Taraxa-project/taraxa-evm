package vm

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
