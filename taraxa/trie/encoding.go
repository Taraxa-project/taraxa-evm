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

package trie

// Trie keys are dealt with in three distinct encodings:
//
// KEYBYTES encoding contains the actual key and nothing else. This encoding is the
// input to most API functions.
//
// HEX encoding contains one byte for each nibble of the key and an optional trailing
// 'terminator' byte of value 0x10 which indicates whether or not the node at the key
// contains a value. Hex key encoding is used for nodes loaded in memory because it's
// convenient to access.
//
// COMPACT encoding is defined by the Ethereum Yellow Paper (it's called "hex prefix
// encoding" there) and contains the bytes of the key and a flag. The high nibble of the
// first byte contains the flag; the lowest bit encoding the oddness of the length and
// the second-lowest encoding whether the node at the key is a value node. The low nibble
// of the first byte is zero in the case of an even number of nibbles and the first nibble
// in the case of an odd number. All remaining nibbles (now an even number) fit properly
// into the remaining bytes. Compact encoding is used for nodes stored on disk.

func hex_to_compact(in []byte, buf *hex_key_compact) (ret []byte) {
	terminator := byte(0)
	if hasTerm(in) {
		terminator = 1
		in = in[:len(in)-1]
	}
	ret = buf[:len(in)/2+1]
	ret[0] = terminator << 5 // the flag byte
	if len(in)&1 == 1 {
		ret[0] |= 1 << 4 // odd flag
		ret[0] |= in[0]  // first nibble is contained in the first byte
		in = in[1:]
	}
	decodeNibbles(in, ret[1:])
	return
}

func compact_to_hex(compact []byte) (ret []byte) {
	ret = keybytesToHex(compact)
	// delete terminator flag
	if ret[0] < 2 {
		ret = ret[:len(ret)-1]
	}
	// apply odd flag
	chop := 2 - ret[0]&1
	return ret[chop:]
}

func keybytes_to_hex(in, out []byte) {
	for i, b := range in {
		out[i*2] = b / 16
		out[i*2+1] = b % 16
	}
	out[len(out)-1] = 16
}

func keybytesToHex(str []byte) (nibbles []byte) {
	nibbles = make([]byte, len(str)*2+1)
	keybytes_to_hex(str, nibbles)
	return
}

// hexToKeybytes turns hex nibbles into key bytes.
// This can only be used for keys of even length.
func hex_to_keybytes(in, out []byte) {
	if hasTerm(in) {
		in = in[:len(in)-1]
	}
	if len(in)&1 != 0 {
		panic("can't convert hex key of odd length")
	}
	decodeNibbles(in, out)
}

func decodeNibbles(nibbles []byte, bytes []byte) {
	for bi, ni := 0, 0; ni < len(nibbles); bi, ni = bi+1, ni+2 {
		bytes[bi] = nibbles[ni]<<4 | nibbles[ni+1]
	}
}

// prefixLen returns the length of the common prefix of a and b.
func prefixLen(a, b []byte) int {
	var i, length = 0, len(a)
	if len(b) < length {
		length = len(b)
	}
	for ; i < length; i++ {
		if a[i] != b[i] {
			break
		}
	}
	return i
}

// hasTerm returns whether a hex key has the terminator flag.
func hasTerm(s []byte) bool {
	return len(s) > 0 && s[len(s)-1] == 16
}
