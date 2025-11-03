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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/common"
)

// precompiledTest defines the input/output pairs for precompiled contract tests.
type precompiledTest struct {
	Input, Expected string
	Name            string
	NoBenchmark     bool // Benchmark primarily the worst-cases
}

// precompiledFailureTest defines the input/error pairs for precompiled
// contract failure tests.
type precompiledFailureTest struct {
	Input         string
	ExpectedError string
	Name          string
}

var allPrecompiles = PrecompiledContractsCacti

// EIP-152 test vectors
var blake2FMalformedinputTests = []precompiledFailureTest{
	{
		Input:         "",
		ExpectedError: errBlake2FInvalidInputLength.Error(),
		Name:          "vector 0: empty input",
	},
	{
		Input:         "00000c48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000001",
		ExpectedError: errBlake2FInvalidInputLength.Error(),
		Name:          "vector 1: less than 213 bytes input",
	},
	{
		Input:         "000000000c48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000001",
		ExpectedError: errBlake2FInvalidInputLength.Error(),
		Name:          "vector 2: more than 213 bytes input",
	},
	{
		Input:         "0000000c48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000002",
		ExpectedError: errBlake2FInvalidFinalFlag.Error(),
		Name:          "vector 3: malformed final block indicator flag",
	},
}

func testPrecompiled(addr string, test precompiledTest, t *testing.T) {
	a := common.HexToAddress(addr)
	p := allPrecompiles.Get(&a)
	in := common.Hex2Bytes(test.Input)
	frame := CallFrame{Input: in}
	t.Run(fmt.Sprintf("%s-Gas=%d", test.Name, p.RequiredGas(frame, nil)), func(t *testing.T) {
		if res, err := p.Run(frame, nil); err != nil {
			t.Error(err)
		} else if common.Bytes2Hex(res) != test.Expected {
			t.Errorf("Expected %v, got %v", test.Expected, common.Bytes2Hex(res))
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func testPrecompiledOOG(addr string, test precompiledTest, t *testing.T) {
	a := common.HexToAddress(addr)
	p := allPrecompiles.Get(&a)
	in := common.Hex2Bytes(test.Input)
	frame := CallFrame{Input: in}
	reqGas := p.RequiredGas(frame, nil) - 1
	frame.Gas = reqGas
	t.Run(fmt.Sprintf("%s-Gas=%d", test.Name, reqGas), func(t *testing.T) {
		_, err := p.Run(frame, nil) // TODO this should first check for gas
		if err.Error() != "out of gas" {
			t.Errorf("Expected error [out of gas], got [%v]", err)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func testPrecompiledFailure(addr string, test precompiledFailureTest, t *testing.T) {
	a := common.HexToAddress(addr)
	p := allPrecompiles.Get(&a)
	in := common.Hex2Bytes(test.Input)
	frame := CallFrame{Input: in}
	t.Run(test.Name, func(t *testing.T) {
		_, err := p.Run(frame, nil)
		if err.Error() != test.ExpectedError {
			t.Errorf("Expected error [%v], got [%v]", test.ExpectedError, err)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func benchmarkPrecompiled(addr string, test precompiledTest, bench *testing.B) {
	if test.NoBenchmark {
		return
	}
	a := common.HexToAddress(addr)
	p := PrecompiledContractsCalifornicum.Get(&a)
	in := common.Hex2Bytes(test.Input)
	frame := CallFrame{Input: in}
	reqGas := p.RequiredGas(frame, nil)

	var (
		res []byte
		err error
	)

	bench.Run(fmt.Sprintf("%s-Gas=%d", test.Name, p.RequiredGas(frame, nil)), func(bench *testing.B) {
		bench.ReportAllocs()
		bench.ResetTimer()
		for i := 0; i < bench.N; i++ {
			res, err = p.Run(frame, nil)
		}
		bench.StopTimer()
		bench.ReportMetric(float64(reqGas), "gas/op")
		//Check if it is correct
		if err != nil {
			bench.Error(err)
			return
		}
		if common.Bytes2Hex(res) != test.Expected {
			bench.Error(fmt.Sprintf("Expected %v, got %v", test.Expected, common.Bytes2Hex(res)))
			return
		}
	})
}

// Benchmarks the sample inputs from the ECRECOVER precompile.
func BenchmarkPrecompiledEcrecover(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "000000000000000000000000ceaccac640adf55b2028469bd36ba501f28b699d",
		Name:     "",
	}
	benchmarkPrecompiled("01", t, bench)
}

// Benchmarks the sample inputs from the SHA256 precompile.
func BenchmarkPrecompiledSha256(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "811c7003375852fabd0d362e40e68607a12bdabae61a7d068fe5fdd1dbbf2a5d",
		Name:     "128",
	}
	benchmarkPrecompiled("02", t, bench)
}

// Benchmarks the sample inputs from the RIPEMD precompile.
func BenchmarkPrecompiledRipeMD(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "0000000000000000000000009215b8d9882ff46f0dfde6684d78e831467f65e6",
		Name:     "128",
	}
	benchmarkPrecompiled("03", t, bench)
}

// Benchmarks the sample inputs from the identiy precompile.
func BenchmarkPrecompiledIdentity(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Name:     "128",
	}
	benchmarkPrecompiled("04", t, bench)
}

// Tests the sample inputs from the ModExp EIP 198.
func TestPrecompiledModExp(t *testing.T)      { testJson("modexp", "05", t) }
func BenchmarkPrecompiledModExp(b *testing.B) { benchJson("modexp", "05", b) }

// Tests the sample inputs from the elliptic curve addition EIP 213.
func TestPrecompiledBn256Add(t *testing.T)      { testJson("bn256Add", "06", t) }
func BenchmarkPrecompiledBn256Add(b *testing.B) { benchJson("bn256Add", "06", b) }

// Tests OOG
// TODO this is not working because how we are calculating gas
// func TestPrecompiledModExpOOG(t *testing.T) {
// 	modexpTests, err := loadJson("modexp")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	for _, test := range modexpTests {
// 		testPrecompiledOOG("05", test, t)
// 	}
// }

// Tests the sample inputs from the elliptic curve scalar multiplication EIP 213.
func TestPrecompiledBn256ScalarMul(t *testing.T)      { testJson("bn256ScalarMul", "07", t) }
func BenchmarkPrecompiledBn256ScalarMul(b *testing.B) { benchJson("bn256ScalarMul", "07", b) }

// Tests the sample inputs from the elliptic curve pairing check EIP 197.
func TestPrecompiledBn256Pairing(t *testing.T)      { testJson("bn256Pairing", "08", t) }
func BenchmarkPrecompiledBn256Pairing(b *testing.B) { benchJson("bn256Pairing", "08", b) }

func TestPrecompiledBlake2F(t *testing.T)      { testJson("blake2F", "09", t) }
func BenchmarkPrecompiledBlake2F(b *testing.B) { benchJson("blake2F", "09", b) }

func TestPrecompileBlake2FMalformedinput(t *testing.T) {
	for _, test := range blake2FMalformedinputTests {
		testPrecompiledFailure("09", test, t)
	}
}

func TestPrecompiledEcrecover(t *testing.T) { testJson("ecRecover", "01", t) }

func testJson(name, addr string, t *testing.T) {
	tests, err := loadJson(name)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		testPrecompiled(addr, test, t)
	}
}

func testJsonFail(name, addr string, t *testing.T) {
	tests, err := loadJsonFail(name)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		testPrecompiledFailure(addr, test, t)
	}
}

func benchJson(name, addr string, b *testing.B) {
	tests, err := loadJson(name)
	if err != nil {
		b.Fatal(err)
	}
	for _, test := range tests {
		benchmarkPrecompiled(addr, test, b)
	}
}

func TestPrecompiledBLS12381G1Add(t *testing.T)      { testJson("blsG1Add", "0b", t) }
func TestPrecompiledBLS12381G1MultiExp(t *testing.T) { testJson("blsG1MultiExp", "0c", t) }
func TestPrecompiledBLS12381G2Add(t *testing.T)      { testJson("blsG2Add", "0d", t) }
func TestPrecompiledBLS12381G2MultiExp(t *testing.T) { testJson("blsG2MultiExp", "0e", t) }
func TestPrecompiledBLS12381Pairing(t *testing.T)    { testJson("blsPairing", "0f", t) }
func TestPrecompiledBLS12381MapG1(t *testing.T)      { testJson("blsMapG1", "10", t) }
func TestPrecompiledBLS12381MapG2(t *testing.T)      { testJson("blsMapG2", "11", t) }

func BenchmarkPrecompiledBLS12381G1Add(b *testing.B)      { benchJson("blsG1Add", "0b", b) }
func BenchmarkPrecompiledBLS12381G1MultiExp(b *testing.B) { benchJson("blsG1MultiExp", "0c", b) }
func BenchmarkPrecompiledBLS12381G2Add(b *testing.B)      { benchJson("blsG2Add", "0d", b) }
func BenchmarkPrecompiledBLS12381G2MultiExp(b *testing.B) { benchJson("blsG2MultiExp", "0e", b) }
func BenchmarkPrecompiledBLS12381Pairing(b *testing.B)    { benchJson("blsPairing", "0f", b) }
func BenchmarkPrecompiledBLS12381MapG1(b *testing.B)      { benchJson("blsMapG1", "10", b) }
func BenchmarkPrecompiledBLS12381MapG2(b *testing.B)      { benchJson("blsMapG2", "12", b) }

// Failure tests
func TestPrecompiledBLS12381G1AddFail(t *testing.T)      { testJsonFail("blsG1Add", "0b", t) }
func TestPrecompiledBLS12381G1MultiExpFail(t *testing.T) { testJsonFail("blsG1MultiExp", "0c", t) }
func TestPrecompiledBLS12381G2AddFail(t *testing.T)      { testJsonFail("blsG2Add", "0d", t) }
func TestPrecompiledBLS12381G2MultiExpFail(t *testing.T) { testJsonFail("blsG2MultiExp", "0e", t) }
func TestPrecompiledBLS12381PairingFail(t *testing.T)    { testJsonFail("blsPairing", "0f", t) }
func TestPrecompiledBLS12381MapG1Fail(t *testing.T)      { testJsonFail("blsMapG1", "10", t) }
func TestPrecompiledBLS12381MapG2Fail(t *testing.T)      { testJsonFail("blsMapG2", "11", t) }

func loadJson(name string) ([]precompiledTest, error) {
	data, err := os.ReadFile(fmt.Sprintf("testdata/precompiles/%v.json", name))
	if err != nil {
		return nil, err
	}
	var testcases []precompiledTest
	err = json.Unmarshal(data, &testcases)
	return testcases, err
}

func loadJsonFail(name string) ([]precompiledFailureTest, error) {
	data, err := os.ReadFile(fmt.Sprintf("testdata/precompiles/fail-%v.json", name))
	if err != nil {
		return nil, err
	}
	var testcases []precompiledFailureTest
	err = json.Unmarshal(data, &testcases)
	return testcases, err
}

// BenchmarkPrecompiledBLS12381G1MultiExpWorstCase benchmarks the worst case we could find that still fits a gaslimit of 10MGas.
func BenchmarkPrecompiledBLS12381G1MultiExpWorstCase(b *testing.B) {
	task := "0000000000000000000000000000000008d8c4a16fb9d8800cce987c0eadbb6b3b005c213d44ecb5adeed713bae79d606041406df26169c35df63cf972c94be1" +
		"0000000000000000000000000000000011bc8afe71676e6730702a46ef817060249cd06cd82e6981085012ff6d013aa4470ba3a2c71e13ef653e1e223d1ccfe9" +
		"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"
	input := task
	for i := 0; i < 4787; i++ {
		input = input + task
	}
	testcase := precompiledTest{
		Input:       input,
		Expected:    "0000000000000000000000000000000005a6310ea6f2a598023ae48819afc292b4dfcb40aabad24a0c2cb6c19769465691859eeb2a764342a810c5038d700f18000000000000000000000000000000001268ac944437d15923dc0aec00daa9250252e43e4b35ec7a19d01f0d6cd27f6e139d80dae16ba1c79cc7f57055a93ff5",
		Name:        "WorstCaseG1",
		NoBenchmark: false,
	}
	benchmarkPrecompiled("f0c", testcase, b)
}

// BenchmarkPrecompiledBLS12381G2MultiExpWorstCase benchmarks the worst case we could find that still fits a gaslimit of 10MGas.
func BenchmarkPrecompiledBLS12381G2MultiExpWorstCase(b *testing.B) {
	task := "000000000000000000000000000000000d4f09acd5f362e0a516d4c13c5e2f504d9bd49fdfb6d8b7a7ab35a02c391c8112b03270d5d9eefe9b659dd27601d18f" +
		"000000000000000000000000000000000fd489cb75945f3b5ebb1c0e326d59602934c8f78fe9294a8877e7aeb95de5addde0cb7ab53674df8b2cfbb036b30b99" +
		"00000000000000000000000000000000055dbc4eca768714e098bbe9c71cf54b40f51c26e95808ee79225a87fb6fa1415178db47f02d856fea56a752d185f86b" +
		"000000000000000000000000000000001239b7640f416eb6e921fe47f7501d504fadc190d9cf4e89ae2b717276739a2f4ee9f637c35e23c480df029fd8d247c7" +
		"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"
	input := task
	for i := 0; i < 1040; i++ {
		input = input + task
	}

	testcase := precompiledTest{
		Input:       input,
		Expected:    "0000000000000000000000000000000018f5ea0c8b086095cfe23f6bb1d90d45de929292006dba8cdedd6d3203af3c6bbfd592e93ecb2b2c81004961fdcbb46c00000000000000000000000000000000076873199175664f1b6493a43c02234f49dc66f077d3007823e0343ad92e30bd7dc209013435ca9f197aca44d88e9dac000000000000000000000000000000000e6f07f4b23b511eac1e2682a0fc224c15d80e122a3e222d00a41fab15eba645a700b9ae84f331ae4ed873678e2e6c9b000000000000000000000000000000000bcb4849e460612aaed79617255fd30c03f51cf03d2ed4163ca810c13e1954b1e8663157b957a601829bb272a4e6c7b8",
		Name:        "WorstCaseG2",
		NoBenchmark: false,
	}
	benchmarkPrecompiled("f0f", testcase, b)
}

// Benchmarks the sample inputs from the P256VERIFY precompile.
func BenchmarkPrecompiledP256Verify(bench *testing.B) {
	t := precompiledTest{
		Input:    "4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "p256Verify",
	}
	benchmarkPrecompiled("0x100", t, bench)
}

func TestPrecompiledP256Verify(t *testing.T) { testJson("p256Verify", "0x100", t) }

// EIP-7213 test vectors
var falcon512MalformedInputTests = []precompiledFailureTest{
	{
		Input:         "",
		ExpectedError: errFalcon512InvalidMethodSignatureLength.Error(),
		Name:          "vector 0: empty input",
	},
	{
		Input:         "111111110000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000032000000000000000000000000000000000000000000000000000000000000006e0",
		ExpectedError: errFalconInvalidMethodSignature.Error(),
		Name:          "vector 1: 4 method signature bytes is not de8f50a1",
	},
}

var falcon512InvalidSignatureTests = []precompiledTest{
	{
		Input:    "de8f50a1",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 2: No data after method signature",
	},
	{
		Input:    "de8f50a1abc",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 3: No enough data after method signature",
	},
	{
		Input:    "de8f50a100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000e0",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 4: Signature offset is zero",
	},
	{
		Input:    "de8f50a10000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e0",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 5: Public key offset is zero",
	},
	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 6: data offset is zero",
	},

	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e0",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 7: signature length not present in input",
	},
	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000000",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 8: signature length is present but its value is zero",
	},
	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000600000000",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 9: Signature indicated length is too long",
	},

	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000000201111111111111111111111111111111111111111111111111111111111111111",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 10: public key length not present in input",
	},
	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000020111111111111111111111111111111111111111111111111111111111111111100000000000000000000000000000000000000000000000000000000000000000",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 11: public key length is present but its value is zero",
	},
	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000020111111111111111111111111111111111111111111111111111111111111111100000000000000000000000000000000000000000000000000800000000000000",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 12: Public Key indicated length is too long",
	},

	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000020111111111111111111111111111111111111111111111111111111111111111100000000000000000000000000000000000000000000000000000000000000202222222222222222222222222222222222222222222222222222222222222222",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 13: data length not present in input",
	},
	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000000201111111111111111111111111111111111111111111111111111111111111111000000000000000000000000000000000000000000000000000000000000002022222222222222222222222222222222222222222222222222222222222222220000000000000000000000000000000000000000000000000000000000000000",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 14: data length is present but its value is zero",
	},
	{
		Input:    "de8f50a1000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000002011111111111111111111111111111111111111111111111111111111111111110000000000000000000000000000000000000000000000000000000000000020222222222222222222222222222222222222222222222222222222222222222200000000000000000000000000000000000000000000000050000000000000000",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 15: Data indicated length is too long",
	},
	{
		Input:    "de8f50a10000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000031200000000000000000000000000000000000000000000000000000000000006b30000000000000000000000000000000000000000000000000000000000000292390191ef48486eb9d9a6823d8e6ff0d7f4df8bed13af7fa55a7e8dfa0d19725842a5451bcf4c061982d021c63a7d666c2024fe57033b1a1bda8c2179c518c94d434947eacc109efe792857ff6450cc853e8bb9d5d951b3ddb1397fadc2210762b479e386e660a68eb2ad034a58a3d0cce2370edf257ff4f81fe97c9bb10c1a96ee87d2414fdf4ea39f9f7c0604d1a3aebe5d73a6a1f7221a99eb2389b905da94766cd27a9a89f13d56b97cdd346689cfc5b3bdbbdc5102d3b5f93a2a95be40afcaac416d831391474c2298da0a359bc958a5f227de991a338944d6ebe4963f1a4ba5bb3d5c672526992329c84c5770c03a5e43504a28fa86f3b9bd72354c0b1921d1b568e5212856d8436f79a8c409e631082a4e28b7cb99d89c2a9ab3c196c56a77e6d05718196b316d8e9bedcfc214e2a33c02f0afd821dbfcb966c8bb8dde01c812a2045b6bc8de767f97b739dc8ed262f43dfc092e49f34c7728839895e4620620e4a193e94dbf2a9d439e249e6ec3377ed6734030acd43d138b982c36df814a4257f971a53aa5d92a7670915469013e68124edae3fc11b4ca470938ee2f218bc0e11468a6092f233de914790c972f77d95f968ac5f6e0968e0efa8d628825f3fbad2254e04e9ba34defc698b2e19e9629290f4e5682270595035b07404b091f5325293ed4021c619d7f0914ac47219b9e5df6fda5acbb398adb5c8540dc3390a548d82be42fac8f6ef9963546bcf278e63fdd49d0abbe62208e3956468771df4a50dd7a3aa197d81f58c14425ee166d29aef3eb2c171ff9b24f575e0ae9a65227d1e57ab6c5df11a369645dac717ab671a4c10aef72824134d0e68676bb138ca20eb5e08a0d1f90ee48af1cce1f21c72394ac427f382ced545183689cdb72cec8ea30c77389a16204356259e4b1142d000000000000000000000000000000000000000000000000000000000000038109b7107f987f937ea7566bca38e1578b8d0b59294e68bd208bfa90133101f5efd8755e9f1fd20564762585baa5a4f165ebec6df80ab5248a22bba940a7754abe5329b5c60345f395ad2a33ac106e65f14b91d0dca308ae4cc6db5fe69eee9fe4a392ff64f52865070eb5587e7f83ab6c187cc0584b3552920a3d4b50ab0a4a1e8d9dea3d4b5eb015a2f4ab59ae3d0ad2c081d8d2428a83806de7973ec65975ff8fd7287c642b6a83250255b61838cb4d68c52600b8c5330fce48aaeb806d4fb99a9add5e56577454a5a5ad5699711dd04854bf11484713dea5d9abd17f9214c33c4f4d47a6600bc241011f52e29d9820ac85ab70dfccb1d08c003b489c0826bf28ccee71945637b7161e6a99584451bf8351a43a0b0755ce3044f0840b7ad0489e6572c896666463e2cfc8ebe1258e3a963ab1433b173865705c15f044bcfde1b780d29e422604a9081d2349f6d6b40671b7c6ae77f44c16a22412e9e32cb116363d99ca4d2c3ace6730fd45fc6612d389edcd1c9b2201ba32a4705fac61005e184b89a4c90983acd7afee694ac9d904473eb512ec2d4875c1c954b791506f02c9e65f5d04976ea4e81d22d4884eb1c47eeb1a7ee109e12e61ce0ee4dfa88fdacb78ed61b0a327c2069d8cd33d184e68a60c22f6804faceca968cf5c1c276c7d16386f38bb82d5ea1e11d801f5ef33d3a3b0171dc870741ce8373c779ae8935211348c436285703681f1e6b0adc05c35c56196c246731ee2a4a998ef918a165023a76d324c58419cd9e76ebca0e13823d90b2ebc641717b404e2ee2937d48b38441e88f1086c15c95de8a48632bee5fb56f99f07ac31037323000317c291e2eace7865ece23548e804679241f1366748b1656cd58c28b86d5e08e269d0e3a668a834f4178a188dd63384042773fac10b3d96f533ecabe3a8a27e091d5846d6ead8ac9241437240ad4f7d274b78403402210ad042ddf73d59e02abf657aa41e101455df638d44c181e4ca219f2c6679088ff11af439115d8ef38f3614b957e1ebb9cc2e6bee0c0664da7ba3f1268404a5bac8ed45854881972382908861cc7f14f5d03b112273917854617590aad70eeedf398cb206ff5c7f7a9c5390dfa27e14b1148518833b3375cdfaf5a73680cdd7d0ea5f664672fe91ae6700032a3ee21aeb3ab7b6a0f66b0f65597a7fb6f5c1a9b1459d48885db3734abec9918c0f3a8135bbb2279984a054115e9c12a8f10ca25b93be8a3acfb94dc6a90d4da0e0a7ada80000000000000000000000000000000000000000000000000000000000000064486c8e99de7f81a3b0f4610bb555be687d67e079f5b03ee9d18c21d766fe3d2d36ea378a89294f839dc9bcd9a8251f92cfd39aacc1e228f44442e95b3c59ef904613ece9d312028b07a3cb26b3c9844dcdb2699299d7d47fd63f7889d6c5ce34626b583b",
		Expected: "0000000000000000000000000000000000000000000000000000000000000001",
		Name:     "vector 16: Correct format but invalid Signature",
	},
}

func BenchmarkPrecompiledFalcon512(b *testing.B) { benchJson("falconBenchVectors", "14", b) }

func TestPrecompileFalcon512MalformedInput(t *testing.T) {
	for _, test := range falcon512MalformedInputTests {
		testPrecompiledFailure("fa1c", test, t)
	}
}

func TestPrecompiledFalcon512InvalidSignature(t *testing.T) {
	for _, test := range falcon512InvalidSignatureTests {
		testPrecompiled("fa1c", test, t)
	}
}
