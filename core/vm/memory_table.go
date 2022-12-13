// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"github.com/holiman/uint256"
)

func memoryKeccak256(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(0), stack.Back(1))
}

func memoryCallDataCopy(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(0), stack.Back(2))
}

func memoryReturnDataCopy(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(0), stack.Back(2))
}

func memoryCodeCopy(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(0), stack.Back(2))
}

func memoryExtCodeCopy(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(1), stack.Back(3))
}

func memoryMLoad(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(0), uint256.NewInt(32))
}

func memoryMStore8(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(0), uint256.NewInt(1))
}

func memoryMStore(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(0), uint256.NewInt(32))
}

func memoryCreate(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(1), stack.Back(2))
}

func memoryCreate2(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(1), stack.Back(2))
}

func memoryCall(stack *Stack) *uint256.Int {
	x := calcMemSize(stack.Back(5), stack.Back(6))
	y := calcMemSize(stack.Back(3), stack.Back(4))

	return bigMax(x, y)
}

func memoryDelegateCall(stack *Stack) *uint256.Int {
	x := calcMemSize(stack.Back(4), stack.Back(5))
	y := calcMemSize(stack.Back(2), stack.Back(3))

	return bigMax(x, y)
}

func memoryStaticCall(stack *Stack) *uint256.Int {
	x := calcMemSize(stack.Back(4), stack.Back(5))
	y := calcMemSize(stack.Back(2), stack.Back(3))

	return bigMax(x, y)
}

func memoryReturn(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(0), stack.Back(1))
}

func memoryRevert(stack *Stack) *uint256.Int {
	return calcMemSize(stack.Back(0), stack.Back(1))
}

func memoryLog(stack *Stack) *uint256.Int {
	mSize, mStart := stack.Back(1), stack.Back(0)
	return calcMemSize(mStart, mSize)
}

// BigMax returns the larger of x or y.
func bigMax(x, y *uint256.Int) *uint256.Int {
	if x.Lt(y) {
		return y
	}
	return x
}
