// Copyright 2014 The go-ethereum Authors
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
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc"
	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"github.com/ethereum/go-ethereum/crypto/blake2b"
	"github.com/pornin/go-fn-dsa/fndsa"

	"golang.org/x/crypto/ripemd160"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/math"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/crypto/bn256"
	"github.com/Taraxa-project/taraxa-evm/crypto/secp256r1"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

// PrecompiledContract is the basic interface for native Go contracts. The implementation
// requires a deterministic gas count based on the input size of the Run method of the
// contract.
type PrecompiledContract interface {
	RequiredGas(ctx CallFrame, evm *EVM) uint64
	Run(ctx CallFrame, evm *EVM) ([]byte, error)
}

type Precompiles map[common.Address]PrecompiledContract

var PrecompiledContractAddrPrefix = make([]byte, common.AddressLength-1)

func (p *Precompiles) Get(address *common.Address) PrecompiledContract {
	return (*p)[*address]
}

func (p *Precompiles) Put(address *common.Address, contract PrecompiledContract) {
	(*p)[*address] = contract
}

// PrecompiledContractsCalifornicum contains the default set of pre-compiled Ethereum
// contracts used in the Californicum release.
var PrecompiledContractsCalifornicum = Precompiles{
	common.BytesToAddress([]byte{0x01}): &ecrecover{},
	common.BytesToAddress([]byte{0x02}): &sha256hash{},
	common.BytesToAddress([]byte{0x03}): &ripemd160hash{},
	common.BytesToAddress([]byte{0x04}): &dataCopy{},
	common.BytesToAddress([]byte{0x05}): &bigModExp{},
	common.BytesToAddress([]byte{0x06}): &bn256Add{},
	common.BytesToAddress([]byte{0x07}): &bn256ScalarMul{},
	common.BytesToAddress([]byte{0x08}): &bn256Pairing{},
}

var PrecompiledContractsFicus = Precompiles{
	common.BytesToAddress([]byte{0x01}): &ecrecover{},
	common.BytesToAddress([]byte{0x02}): &sha256hash{},
	common.BytesToAddress([]byte{0x03}): &ripemd160hash{},
	common.BytesToAddress([]byte{0x04}): &dataCopy{},
	common.BytesToAddress([]byte{0x05}): &bigModExp{},
	common.BytesToAddress([]byte{0x06}): &bn256Add{},
	common.BytesToAddress([]byte{0x07}): &bn256ScalarMul{},
	common.BytesToAddress([]byte{0x08}): &bn256Pairing{},
	common.BytesToAddress([]byte{0x09}): &blake2F{},
	// common.BytesToAddress([]byte{0x0a}): &kzgPointEvaluation{},
	common.BytesToAddress([]byte{0x0b}): &bls12381G1Add{},
	common.BytesToAddress([]byte{0x0c}): &bls12381G1Mul{},
	common.BytesToAddress([]byte{0x0d}): &bls12381G1MultiExp{},
	common.BytesToAddress([]byte{0x0e}): &bls12381G2Add{},
	common.BytesToAddress([]byte{0x0f}): &bls12381G2Mul{},
	common.BytesToAddress([]byte{0x10}): &bls12381G2MultiExp{},
	common.BytesToAddress([]byte{0x11}): &bls12381Pairing{},
	common.BytesToAddress([]byte{0x12}): &bls12381MapG1{},
	common.BytesToAddress([]byte{0x13}): &bls12381MapG2{},
}

var PrecompiledContractsCacti = Precompiles{
	common.BytesToAddress([]byte{0x01}): &ecrecover{},
	common.BytesToAddress([]byte{0x02}): &sha256hash{},
	common.BytesToAddress([]byte{0x03}): &ripemd160hash{},
	common.BytesToAddress([]byte{0x04}): &dataCopy{},
	common.BytesToAddress([]byte{0x05}): &bigModExp{},
	common.BytesToAddress([]byte{0x06}): &bn256Add{},
	common.BytesToAddress([]byte{0x07}): &bn256ScalarMul{},
	common.BytesToAddress([]byte{0x08}): &bn256Pairing{},
	common.BytesToAddress([]byte{0x09}): &blake2F{},
	//common.BytesToAddress([]byte{0x0a}):       &kzgPointEvaluation{},
	common.BytesToAddress([]byte{0x0b}):       &bls12381G1Add{},
	common.BytesToAddress([]byte{0x0c}):       &bls12381G1MultiExp{},
	common.BytesToAddress([]byte{0x0d}):       &bls12381G2Add{},
	common.BytesToAddress([]byte{0x0e}):       &bls12381G2MultiExp{},
	common.BytesToAddress([]byte{0x0f}):       &bls12381Pairing{},
	common.BytesToAddress([]byte{0x10}):       &bls12381MapG1{},
	common.BytesToAddress([]byte{0x11}):       &bls12381MapG2{},
	common.BytesToAddress([]byte{0x1, 0x00}):  &p256Verify{},
	common.BytesToAddress([]byte{0xfa, 0x1c}): &falcon512{},
}

// ECRECOVER implemented as a native contract.
type ecrecover struct{}

func (c ecrecover) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return EcrecoverGas
}

func (c ecrecover) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	const ecRecoverInputLength = 128
	input := ctx.Input
	input = common.RightPadBytes(input, ecRecoverInputLength)
	// "input" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(input[64:96])
	s := new(big.Int).SetBytes(input[96:128])
	v := input[63] - 27

	// tighter sig s values input homestead only apply to tx sigs
	if !allZero(input[32:63]) || !crypto.ValidateSignatureValues(v, r, s, false) {
		return nil, nil
	}
	// We must make sure not to modify the 'input', so placing the 'v' along with
	// the signature needs to be done on a new allocation
	sig := make([]byte, 65)
	copy(sig, input[64:128])
	sig[64] = v
	// v needs to be at the end for libsecp256k1
	pubKey, err := crypto.Ecrecover(input[:32], sig)
	// make sure the public key is a valid one
	if err != nil {
		return nil, nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(keccak256.Hash(pubKey[1:])[12:], 32), nil
}

// SHA256 implemented as a native contract.
type sha256hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c sha256hash) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return uint64(len(ctx.Input)+31)/32*Sha256PerWordGas + Sha256BaseGas
}
func (c sha256hash) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	h := sha256.Sum256(ctx.Input)
	return h[:], nil
}

// RIPEMD160 implemented as a native contract.
type ripemd160hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c ripemd160hash) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return uint64(len(ctx.Input)+31)/32*Ripemd160PerWordGas + Ripemd160BaseGas
}
func (c ripemd160hash) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	ripemd := ripemd160.New()
	ripemd.Write(ctx.Input)
	return common.LeftPadBytes(ripemd.Sum(nil), 32), nil
}

// data copy implemented as a native contract.
type dataCopy struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c dataCopy) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return uint64(len(ctx.Input)+31)/32*IdentityPerWordGas + IdentityBaseGas
}
func (c dataCopy) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	return ctx.Input, nil
}

// bigModExp implements a native big integer exponential modular operation.
type bigModExp struct{}

var (
	big1      = big.NewInt(1)
	big4      = big.NewInt(4)
	big8      = big.NewInt(8)
	big16     = big.NewInt(16)
	big32     = big.NewInt(32)
	big64     = big.NewInt(64)
	big96     = big.NewInt(96)
	big480    = big.NewInt(480)
	big1024   = big.NewInt(1024)
	big3072   = big.NewInt(3072)
	big199680 = big.NewInt(199680)
)

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bigModExp) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	input := ctx.Input
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32))
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32))
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32))
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Retrieve the head 32 bytes of exp for the adjusted exponent length
	var expHead *big.Int
	if big.NewInt(int64(len(input))).Cmp(baseLen) <= 0 {
		expHead = new(big.Int)
	} else {
		if expLen.Cmp(big32) > 0 {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), 32))
		} else {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), expLen.Uint64()))
		}
	}
	// Calculate the adjusted exponent length
	var msb int
	if bitlen := expHead.BitLen(); bitlen > 0 {
		msb = bitlen - 1
	}
	adjExpLen := new(big.Int)
	if expLen.Cmp(big32) > 0 {
		adjExpLen.Sub(expLen, big32)
		adjExpLen.Mul(big8, adjExpLen)
	}
	adjExpLen.Add(adjExpLen, big.NewInt(int64(msb)))

	// Calculate the gas cost of the operation
	gas := new(big.Int).Set(math.BigMax(modLen, baseLen))
	switch {
	case gas.Cmp(big64) <= 0:
		gas.Mul(gas, gas)
	case gas.Cmp(big1024) <= 0:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big4),
			new(big.Int).Sub(new(big.Int).Mul(big96, gas), big3072),
		)
	default:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big16),
			new(big.Int).Sub(new(big.Int).Mul(big480, gas), big199680),
		)
	}
	gas.Mul(gas, math.BigMax(adjExpLen, big1))
	gas.Div(gas, new(big.Int).SetUint64(ModExpQuadCoeffDiv))

	if gas.BitLen() > 64 {
		return math.MaxUint64
	}
	return gas.Uint64()
}

func (c bigModExp) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32)).Uint64()
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32)).Uint64()
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32)).Uint64()
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Handle a special case when both the base and mod length is zero
	if baseLen == 0 && modLen == 0 {
		return []byte{}, nil
	}
	// Retrieve the operands and execute the exponentiation
	var (
		base = new(big.Int).SetBytes(getData(input, 0, baseLen))
		exp  = new(big.Int).SetBytes(getData(input, baseLen, expLen))
		mod  = new(big.Int).SetBytes(getData(input, baseLen+expLen, modLen))
	)
	if mod.BitLen() == 0 {
		// Modulo 0 is undefined, return zero
		return common.LeftPadBytes([]byte{}, int(modLen)), nil
	}
	return common.LeftPadBytes(base.Exp(base, exp, mod).Bytes(), int(modLen)), nil
}

// newCurvePoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newCurvePoint(blob []byte) (*bn256.G1, error) {
	p := new(bn256.G1)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

// newTwistPoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newTwistPoint(blob []byte) (*bn256.G2, error) {
	p := new(bn256.G2)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

// bn256Add implements a native elliptic curve point addition.
type bn256Add struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bn256Add) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return Bn256AddGas
}

func (c bn256Add) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	x, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	y, err := newCurvePoint(getData(input, 64, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.Add(x, y)
	return res.Marshal(), nil
}

// bn256ScalarMul implements a native elliptic curve scalar multiplication.
type bn256ScalarMul struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bn256ScalarMul) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return Bn256ScalarMulGas
}

func (c bn256ScalarMul) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	p, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.ScalarMult(p, new(big.Int).SetBytes(getData(input, 64, 32)))
	return res.Marshal(), nil
}

var (
	// true32Byte is returned if the bn256 pairing check succeeds.
	true32Byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	// false32Byte is returned if the bn256 pairing check fails.
	false32Byte = make([]byte, 32)

	// errBadPairingInput is returned if the bn256 pairing input is invalid.
	errBadPairingInput = errors.New("bad elliptic curve pairing size")
)

// bn256Pairing implements a pairing pre-compile for the bn256 curve
type bn256Pairing struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bn256Pairing) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return Bn256PairingBaseGas + uint64(len(ctx.Input)/192)*Bn256PairingPerPointGas
}

func (c bn256Pairing) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Handle some corner cases cheaply
	if len(input)%192 > 0 {
		return nil, errBadPairingInput
	}
	// Convert the input into a set of coordinates
	var (
		cs []*bn256.G1
		ts []*bn256.G2
	)
	for i := 0; i < len(input); i += 192 {
		c, err := newCurvePoint(input[i : i+64])
		if err != nil {
			return nil, err
		}
		t, err := newTwistPoint(input[i+64 : i+192])
		if err != nil {
			return nil, err
		}
		cs = append(cs, c)
		ts = append(ts, t)
	}
	// Execute the pairing checks and return the results
	if bn256.PairingCheck(cs, ts) {
		return true32Byte, nil
	}
	return false32Byte, nil
}

type blake2F struct{}

func (c blake2F) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	input := ctx.Input
	// If the input is malformed, we can't calculate the gas, return 0 and let the
	// actual call choke and fault.
	if len(input) != blake2FInputLength {
		return 0
	}
	return uint64(binary.BigEndian.Uint32(input[0:4]))
}

const (
	blake2FInputLength        = 213
	blake2FFinalBlockBytes    = byte(1)
	blake2FNonFinalBlockBytes = byte(0)
)

var (
	errBlake2FInvalidInputLength = errors.New("invalid input length")
	errBlake2FInvalidFinalFlag   = errors.New("invalid final flag")
)

func (c blake2F) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Make sure the input is valid (correct length and final flag)
	if len(input) != blake2FInputLength {
		return nil, errBlake2FInvalidInputLength
	}
	if input[212] != blake2FNonFinalBlockBytes && input[212] != blake2FFinalBlockBytes {
		return nil, errBlake2FInvalidFinalFlag
	}
	// Parse the input into the Blake2b call parameters
	var (
		rounds = binary.BigEndian.Uint32(input[0:4])
		final  = (input[212] == blake2FFinalBlockBytes)

		h [8]uint64
		m [16]uint64
		t [2]uint64
	)
	for i := 0; i < 8; i++ {
		offset := 4 + i*8
		h[i] = binary.LittleEndian.Uint64(input[offset : offset+8])
	}
	for i := 0; i < 16; i++ {
		offset := 68 + i*8
		m[i] = binary.LittleEndian.Uint64(input[offset : offset+8])
	}
	t[0] = binary.LittleEndian.Uint64(input[196:204])
	t[1] = binary.LittleEndian.Uint64(input[204:212])

	// Execute the compression function, extract and return the result
	blake2b.F(&h, m, t, final, rounds)

	output := make([]byte, 64)
	for i := 0; i < 8; i++ {
		offset := i * 8
		binary.LittleEndian.PutUint64(output[offset:offset+8], h[i])
	}
	return output, nil
}

var (
	errBLS12381InvalidInputLength          = errors.New("invalid input length")
	errBLS12381InvalidFieldElementTopBytes = errors.New("invalid field element top bytes")
	errBLS12381G1PointSubgroup             = errors.New("g1 point is not on correct subgroup")
	errBLS12381G2PointSubgroup             = errors.New("g2 point is not on correct subgroup")
)

// bls12381G1Add implements EIP-2537 G1Add precompile.
type bls12381G1Add struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bls12381G1Add) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return Bls12381G1AddGas
}

func (c bls12381G1Add) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Implements EIP-2537 G1Add precompile.
	// > G1 addition call expects `256` bytes as an input that is interpreted as byte concatenation of two G1 points (`128` bytes each).
	// > Output is an encoding of addition operation result - single G1 point (`128` bytes).
	if len(input) != 256 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	var p0, p1 *bls12381.G1Affine

	// Decode G1 point p_0
	if p0, err = decodePointG1(input[:128]); err != nil {
		return nil, err
	}
	// Decode G1 point p_1
	if p1, err = decodePointG1(input[128:]); err != nil {
		return nil, err
	}

	// Compute r = p_0 + p_1
	p0.Add(p0, p1)

	// Encode the G1 point result into 128 bytes
	return encodePointG1(p0), nil
}

// bls12381G1Mul implements EIP-2537 G1Mul precompile.
type bls12381G1Mul struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bls12381G1Mul) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return Bls12381G1MulGas
}

func (c bls12381G1Mul) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Implements EIP-2537 G1Mul precompile.
	// > G1 multiplication call expects `160` bytes as an input that is interpreted as byte concatenation of encoding of G1 point (`128` bytes) and encoding of a scalar value (`32` bytes).
	// > Output is an encoding of multiplication operation result - single G1 point (`128` bytes).
	if len(input) != 160 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	var p0 *bls12381.G1Affine

	// Decode G1 point
	if p0, err = decodePointG1(input[:128]); err != nil {
		return nil, err
	}
	// Decode scalar value
	e := new(big.Int).SetBytes(input[128:])

	// Compute r = e * p_0
	r := new(bls12381.G1Affine)
	r.ScalarMultiplication(p0, e)

	// Encode the G1 point into 128 bytes
	return encodePointG1(r), nil
}

// bls12381G1MultiExp implements EIP-2537 G1MultiExp precompile.
type bls12381G1MultiExp struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bls12381G1MultiExp) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	input := ctx.Input
	// Calculate G1 point, scalar value pair length
	k := len(input) / 160
	if k == 0 {
		// Return 0 gas for small input length
		return 0
	}
	// Lookup discount value for G1 point, scalar value pair length
	maxDiscountLen := len(Bls12381MultiExpDiscountTable)
	if k > maxDiscountLen {
		k = maxDiscountLen
	}
	discount := Bls12381MultiExpDiscountTable[k-1]
	// Calculate gas and return the result
	return (uint64(k) * Bls12381G1MulGas * discount) / 1000
}

func (c bls12381G1MultiExp) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Implements EIP-2537 G1MultiExp precompile.
	// G1 multiplication call expects `160*k` bytes as an input that is interpreted as byte concatenation of `k` slices each of them being a byte concatenation of encoding of G1 point (`128` bytes) and encoding of a scalar value (`32` bytes).
	// Output is an encoding of multiexponentiation operation result - single G1 point (`128` bytes).
	k := len(input) / 160
	if len(input) == 0 || len(input)%160 != 0 {
		return nil, errBLS12381InvalidInputLength
	}
	points := make([]bls12381.G1Affine, k)
	scalars := make([]fr.Element, k)

	// Decode point scalar pairs
	for i := 0; i < k; i++ {
		off := 160 * i
		t0, t1, t2 := off, off+128, off+160
		// Decode G1 point
		p, err := decodePointG1(input[t0:t1])
		if err != nil {
			return nil, err
		}
		points[i] = *p
		// Decode scalar value
		scalars[i] = *new(fr.Element).SetBytes(input[t1:t2])
	}

	// Compute r = e_0 * p_0 + e_1 * p_1 + ... + e_(k-1) * p_(k-1)
	r := new(bls12381.G1Affine)
	r.MultiExp(points, scalars, ecc.MultiExpConfig{})

	// Encode the G1 point to 128 bytes
	return encodePointG1(r), nil
}

// bls12381G2Add implements EIP-2537 G2Add precompile.
type bls12381G2Add struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bls12381G2Add) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return Bls12381G2AddGas
}

func (c bls12381G2Add) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Implements EIP-2537 G2Add precompile.
	// > G2 addition call expects `512` bytes as an input that is interpreted as byte concatenation of two G2 points (`256` bytes each).
	// > Output is an encoding of addition operation result - single G2 point (`256` bytes).
	if len(input) != 512 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	var p0, p1 *bls12381.G2Affine

	// Decode G2 point p_0
	if p0, err = decodePointG2(input[:256]); err != nil {
		return nil, err
	}
	// Decode G2 point p_1
	if p1, err = decodePointG2(input[256:]); err != nil {
		return nil, err
	}

	// Compute r = p_0 + p_1
	r := new(bls12381.G2Affine)
	r.Add(p0, p1)

	// Encode the G2 point into 256 bytes
	return encodePointG2(r), nil
}

// bls12381G2Mul implements EIP-2537 G2Mul precompile.
type bls12381G2Mul struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bls12381G2Mul) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return Bls12381G2MulGas
}

func (c bls12381G2Mul) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Implements EIP-2537 G2MUL precompile logic.
	// > G2 multiplication call expects `288` bytes as an input that is interpreted as byte concatenation of encoding of G2 point (`256` bytes) and encoding of a scalar value (`32` bytes).
	// > Output is an encoding of multiplication operation result - single G2 point (`256` bytes).
	if len(input) != 288 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	var p0 *bls12381.G2Affine

	// Decode G2 point
	if p0, err = decodePointG2(input[:256]); err != nil {
		return nil, err
	}
	// Decode scalar value
	e := new(big.Int).SetBytes(input[256:])

	// Compute r = e * p_0
	r := new(bls12381.G2Affine)
	r.ScalarMultiplication(p0, e)

	// Encode the G2 point into 256 bytes
	return encodePointG2(r), nil
}

// bls12381G2MultiExp implements EIP-2537 G2MultiExp precompile.
type bls12381G2MultiExp struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bls12381G2MultiExp) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	input := ctx.Input
	// Calculate G2 point, scalar value pair length
	k := len(input) / 288
	if k == 0 {
		// Return 0 gas for small input length
		return 0
	}
	// Lookup discount value for G2 point, scalar value pair length
	maxDiscountLen := len(Bls12381MultiExpDiscountTable)
	if k > maxDiscountLen {
		k = maxDiscountLen
	}
	discount := Bls12381MultiExpDiscountTable[k-1]
	// Calculate gas and return the result
	return (uint64(k) * Bls12381G2MulGas * discount) / 1000
}

func (c bls12381G2MultiExp) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Implements EIP-2537 G2MultiExp precompile logic
	// > G2 multiplication call expects `288*k` bytes as an input that is interpreted as byte concatenation of `k` slices each of them being a byte concatenation of encoding of G2 point (`256` bytes) and encoding of a scalar value (`32` bytes).
	// > Output is an encoding of multiexponentiation operation result - single G2 point (`256` bytes).
	k := len(input) / 288
	if len(input) == 0 || len(input)%288 != 0 {
		return nil, errBLS12381InvalidInputLength
	}
	points := make([]bls12381.G2Affine, k)
	scalars := make([]fr.Element, k)

	// Decode point scalar pairs
	for i := 0; i < k; i++ {
		off := 288 * i
		t0, t1, t2 := off, off+256, off+288
		// Decode G2 point
		p, err := decodePointG2(input[t0:t1])
		if err != nil {
			return nil, err
		}
		points[i] = *p
		// Decode scalar value
		scalars[i] = *new(fr.Element).SetBytes(input[t1:t2])
	}

	// Compute r = e_0 * p_0 + e_1 * p_1 + ... + e_(k-1) * p_(k-1)
	r := new(bls12381.G2Affine)
	r.MultiExp(points, scalars, ecc.MultiExpConfig{})

	// Encode the G2 point to 256 bytes.
	return encodePointG2(r), nil
}

// bls12381Pairing implements EIP-2537 Pairing precompile.
type bls12381Pairing struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bls12381Pairing) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	input := ctx.Input
	return Bls12381PairingBaseGas + uint64(len(input)/384)*Bls12381PairingPerPairGas
}

func (c bls12381Pairing) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Implements EIP-2537 Pairing precompile logic.
	// > Pairing call expects `384*k` bytes as an inputs that is interpreted as byte concatenation of `k` slices. Each slice has the following structure:
	// > - `128` bytes of G1 point encoding
	// > - `256` bytes of G2 point encoding
	// > Output is a `32` bytes where last single byte is `0x01` if pairing result is equal to multiplicative identity in a pairing target field and `0x00` otherwise
	// > (which is equivalent of Big Endian encoding of Solidity values `uint256(1)` and `uin256(0)` respectively).
	k := len(input) / 384
	if len(input) == 0 || len(input)%384 != 0 {
		return nil, errBLS12381InvalidInputLength
	}

	var (
		p []bls12381.G1Affine
		q []bls12381.G2Affine
	)

	// Decode pairs
	for i := 0; i < k; i++ {
		off := 384 * i
		t0, t1, t2 := off, off+128, off+384

		// Decode G1 point
		p1, err := decodePointG1(input[t0:t1])
		if err != nil {
			return nil, err
		}
		// Decode G2 point
		p2, err := decodePointG2(input[t1:t2])
		if err != nil {
			return nil, err
		}

		// 'point is on curve' check already done,
		// Here we need to apply subgroup checks.
		if !p1.IsInSubGroup() {
			return nil, errBLS12381G1PointSubgroup
		}
		if !p2.IsInSubGroup() {
			return nil, errBLS12381G2PointSubgroup
		}
		p = append(p, *p1)
		q = append(q, *p2)
	}
	// Prepare 32 byte output
	out := make([]byte, 32)

	// Compute pairing and set the result
	ok, err := bls12381.PairingCheck(p, q)
	if err == nil && ok {
		out[31] = 1
	}
	return out, nil
}

func decodePointG1(in []byte) (*bls12381.G1Affine, error) {
	if len(in) != 128 {
		return nil, errors.New("invalid g1 point length")
	}
	// decode x
	x, err := decodeBLS12381FieldElement(in[:64])
	if err != nil {
		return nil, err
	}
	// decode y
	y, err := decodeBLS12381FieldElement(in[64:])
	if err != nil {
		return nil, err
	}
	elem := bls12381.G1Affine{X: x, Y: y}
	if !elem.IsOnCurve() {
		return nil, errors.New("invalid point: not on curve")
	}

	return &elem, nil
}

// decodePointG2 given encoded (x, y) coordinates in 256 bytes returns a valid G2 Point.
func decodePointG2(in []byte) (*bls12381.G2Affine, error) {
	if len(in) != 256 {
		return nil, errors.New("invalid g2 point length")
	}
	x0, err := decodeBLS12381FieldElement(in[:64])
	if err != nil {
		return nil, err
	}
	x1, err := decodeBLS12381FieldElement(in[64:128])
	if err != nil {
		return nil, err
	}
	y0, err := decodeBLS12381FieldElement(in[128:192])
	if err != nil {
		return nil, err
	}
	y1, err := decodeBLS12381FieldElement(in[192:])
	if err != nil {
		return nil, err
	}

	p := bls12381.G2Affine{X: bls12381.E2{A0: x0, A1: x1}, Y: bls12381.E2{A0: y0, A1: y1}}
	if !p.IsOnCurve() {
		return nil, errors.New("invalid point: not on curve")
	}
	return &p, err
}

// decodeBLS12381FieldElement decodes BLS12-381 elliptic curve field element.
// Removes top 16 bytes of 64 byte input.
func decodeBLS12381FieldElement(in []byte) (fp.Element, error) {
	if len(in) != 64 {
		return fp.Element{}, errors.New("invalid field element length")
	}
	// check top bytes
	for i := 0; i < 16; i++ {
		if in[i] != byte(0x00) {
			return fp.Element{}, errBLS12381InvalidFieldElementTopBytes
		}
	}
	var res [48]byte
	copy(res[:], in[16:])

	return fp.BigEndian.Element(&res)
}

// encodePointG1 encodes a point into 128 bytes.
func encodePointG1(p *bls12381.G1Affine) []byte {
	out := make([]byte, 128)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[16:]), p.X)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[64+16:]), p.Y)
	return out
}

// encodePointG2 encodes a point into 256 bytes.
func encodePointG2(p *bls12381.G2Affine) []byte {
	out := make([]byte, 256)
	// encode x
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[16:16+48]), p.X.A0)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[80:80+48]), p.X.A1)
	// encode y
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[144:144+48]), p.Y.A0)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[208:208+48]), p.Y.A1)
	return out
}

// bls12381MapG1 implements EIP-2537 MapG1 precompile.
type bls12381MapG1 struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bls12381MapG1) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return Bls12381MapG1Gas
}

func (c bls12381MapG1) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Implements EIP-2537 Map_To_G1 precompile.
	// > Field-to-curve call expects `64` bytes an an input that is interpreted as a an element of the base field.
	// > Output of this call is `128` bytes and is G1 point following respective encoding rules.
	if len(input) != 64 {
		return nil, errBLS12381InvalidInputLength
	}

	// Decode input field element
	fe, err := decodeBLS12381FieldElement(input)
	if err != nil {
		return nil, err
	}

	// Compute mapping
	r := bls12381.MapToG1(fe)
	if err != nil {
		return nil, err
	}

	// Encode the G1 point to 128 bytes
	return encodePointG1(&r), nil
}

// bls12381MapG2 implements EIP-2537 MapG2 precompile.
type bls12381MapG2 struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c bls12381MapG2) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return Bls12381MapG2Gas
}

func (c bls12381MapG2) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	// Implements EIP-2537 Map_FP2_TO_G2 precompile logic.
	// > Field-to-curve call expects `128` bytes an an input that is interpreted as a an element of the quadratic extension field.
	// > Output of this call is `256` bytes and is G2 point following respective encoding rules.
	if len(input) != 128 {
		return nil, errBLS12381InvalidInputLength
	}

	// Decode input field element
	c0, err := decodeBLS12381FieldElement(input[:64])
	if err != nil {
		return nil, err
	}
	c1, err := decodeBLS12381FieldElement(input[64:])
	if err != nil {
		return nil, err
	}

	// Compute mapping
	r := bls12381.MapToG2(bls12381.E2{A0: c0, A1: c1})
	if err != nil {
		return nil, err
	}

	// Encode the G2 point to 256 bytes
	return encodePointG2(&r), nil
}

// P256VERIFY (secp256r1 signature verification)
// implemented as a native contract
type p256Verify struct{}

// RequiredGas returns the gas required to execute the precompiled contract
func (c *p256Verify) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return P256VerifyGas
}

// Run executes the precompiled contract with given 160 bytes of param, returning the output and the used gas
func (c *p256Verify) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input
	const p256VerifyInputLength = 160
	if len(input) != p256VerifyInputLength {
		return nil, nil
	}

	// Extract hash, r, s, x, y from the input.
	hash := input[0:32]
	r, s := new(big.Int).SetBytes(input[32:64]), new(big.Int).SetBytes(input[64:96])
	x, y := new(big.Int).SetBytes(input[96:128]), new(big.Int).SetBytes(input[128:160])

	// Verify the signature.
	if secp256r1.Verify(hash, r, s, x, y) {
		return true32Byte, nil
	}
	return nil, nil
}

// FALCON512 implementation precompile
type falcon512 struct{}

func (c *falcon512) RequiredGas(ctx CallFrame, evm *EVM) uint64 {
	return uint64(len(ctx.Input)+31)/32*Falcon512PerWordGas + Falcon512BaseGas
}

const (
	falcon512MethodSignature = 0xde8f50a1 // verify(bytes,bytes,bytes)
)

var (
	// fndsa exposes fixed sizes per degree; compute them at init to avoid hardcoding.
	falcon512LogN      uint = 9 // 2^9 = 512
	falcon512SigLen         = fndsa.SignatureSize(falcon512LogN)
	falcon512PubKeyLen      = fndsa.VerifyingKeySize(falcon512LogN)

	errFalcon512InvalidMethodSignatureLength = errors.New("invalid input format")
	errFalconInvalidMethodSignature          = errors.New("invalid method signature")
)

// returns bytes32(0) if signature is valid, bytes32(1) otherwise
func (c *falcon512) Run(ctx CallFrame, evm *EVM) ([]byte, error) {
	input := ctx.Input

	// 32-byte digest per your convention: 0x...01 means invalid; 0x...00 means valid
	digest := make([]byte, 32)
	digest[31] = 1

	if len(input) < 4 { // verifies method signature
		return nil, errFalcon512InvalidMethodSignatureLength
	}
	if binary.BigEndian.Uint32(getData(input, 0, 4)) != uint32(falcon512MethodSignature) {
		return nil, errFalconInvalidMethodSignature
	}

	input = getData(input, 4, uint64(len(input)))
	if len(input) < 96 {
		return digest, nil
	}

	signatureOffset := new(big.Int).SetBytes(getData(input, 0, 32)).Uint64()
	pubKeyOffset := new(big.Int).SetBytes(getData(input, 32, 32)).Uint64()
	dataOffset := new(big.Int).SetBytes(getData(input, 64, 32)).Uint64()

	if signatureOffset == 0 || pubKeyOffset == 0 || dataOffset == 0 {
		return digest, nil
	}

	// --- signature ---
	if len(input) < int(signatureOffset)+32 {
		return digest, nil
	}
	signatureLength := new(big.Int).SetBytes(getData(input, signatureOffset, 32)).Uint64()
	if signatureLength == 0 || len(input) < int(signatureOffset)+32+int(signatureLength) {
		return digest, nil
	}
	signatureSlice := getData(input, signatureOffset+32, signatureLength)
	// Enforce fixed-size FN-DSA-512 signatures.
	if int(signatureLength) != falcon512SigLen {
		return digest, nil
	}

	// --- public key ---
	if len(input) < int(pubKeyOffset)+32 {
		return digest, nil
	}
	pubKeyLength := new(big.Int).SetBytes(getData(input, pubKeyOffset, 32)).Uint64()
	if pubKeyLength == 0 || len(input) < int(pubKeyOffset)+32+int(pubKeyLength) {
		return digest, nil
	}
	pubKeySlice := getData(input, pubKeyOffset+32, pubKeyLength)
	// Enforce fixed-size FN-DSA-512 verifying key.
	if int(pubKeyLength) != falcon512PubKeyLen {
		return digest, nil
	}

	// --- message data (raw; no prehash) ---
	if len(input) < int(dataOffset)+32 {
		return digest, nil
	}
	dataLength := new(big.Int).SetBytes(getData(input, dataOffset, 32)).Uint64()
	if dataLength == 0 || len(input) < int(dataOffset)+32+int(dataLength) {
		return digest, nil
	}
	dataSlice := getData(input, dataOffset+32, dataLength)

	// Pure-Go FN-DSA verify (ctx = NONE, id = 0 => raw message).
	ok := fndsa.Verify(pubKeySlice, fndsa.DOMAIN_NONE, 0, dataSlice, signatureSlice)
	if ok {
		digest[31] = 0
	}
	return digest, nil
}
