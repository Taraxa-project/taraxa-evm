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

package rlp

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/binary"
	"io"
	"math/big"
	"reflect"
	"sync"
)

const EmptyString = 0x80

var encoderInterface = reflect.TypeOf(new(RLPEncodable)).Elem()

// encbufs are pooled.
var encbufPool = sync.Pool{
	New: func() interface{} { return new(Encoder) },
}

// RLPEncodable is implemented by types that require custom
// encoding rules or want to encode private fields.
type RLPEncodable interface {
	// EncodeRLP should write the RLP encoding of its receiver to w.
	// If the implementation is a pointer method, it may also be
	// called for nil pointers.
	//
	// Implementations should generate valid RLP. The data written is
	// not verified at the moment, but a future version might. It is
	// recommended to write only a single value but writing multiple
	// values or no value at all is also permitted.
	EncodeRLP(io.Writer) error
}

// Encode writes the RLP encoding of val to w. Note that Encode may
// perform many small writes in some cases. Consider making w
// buffered.
//
// Encode uses the following type-dependent encoding rules:
//
// If the type implements the RLPEncodable interface, Encode calls
// EncodeRLP. This is true even for nil pointers, please see the
// documentation for RLPEncodable.
//
// To encode a pointer, the value being pointed to is encoded. For nil
// pointers, Encode will encode the zero value of the type. A nil
// pointer to a struct type always encodes as an empty RLP list.
// A nil pointer to an array encodes as an empty list (or empty string
// if the array has element type byte).
//
// Struct values are encoded as an RLP list of all their encoded
// public fields. Recursive struct types are supported.
//
// To encode slices and arrays, the elements are encoded as an RLP
// list of the value's elements. Note that arrays and slices with
// element type uint8 or byte are always encoded as an RLP string.
//
// A Go string is encoded as an RLP string.
//
// An unsigned integer value is encoded as an RLP string. Zero always
// encodes as an empty RLP string. Encode also supports *big.Int.
//
// An interface value encodes as the value contained in the interface.
//
// Boolean values are not supported, nor are signed integers, floating
// point numbers, maps, channels and functions.
func Encode(w io.Writer, val interface{}) (err error) {
	if outer, ok := w.(*Encoder); ok {
		// Encode was called by some type's EncodeRLP.
		// Avoid copying by writing to the outer Encoder directly.
		return outer.AppendAny(val)
	}
	eb := encbufPool.Get().(*Encoder)
	defer encbufPool.Put(eb)
	eb.Reset()
	if err = eb.AppendAny(val); err != nil {
		return
	}
	defer func() {
		if rec := recover(); rec != nil && rec != err {
			panic(rec)
		}
	}()
	eb.Flush(nil, func(b ...byte) {
		if _, err = w.Write(b); err != nil {
			panic(err)
		}
	})
	return
}

// EncodeToBytes returns the RLP encoding of val.
// Please see the documentation of Encode for the encoding rules.
func EncodeToBytes(val interface{}) ([]byte, error) {
	eb := encbufPool.Get().(*Encoder)
	defer encbufPool.Put(eb)
	eb.Reset()
	if err := eb.AppendAny(val); err != nil {
		return nil, err
	}
	return eb.ToBytes(nil), nil
}

func BytesAppender(b *[]byte) func(...byte) {
	return func(bs ...byte) {
		*b = append(*b, bs...)
	}
}

type Encoder struct {
	str    []byte      // string data, contains everything except list headers
	lheads []*ListHead // all list headers
	lhsize int         // sum of sizes of all encoded list headers
}
type EncoderConfig struct {
	StringBufferCap   int
	ListHeadBufferCap int
}

func NewEncoder(cfg EncoderConfig) *Encoder {
	return &Encoder{
		str:    make([]byte, 0, cfg.StringBufferCap),
		lheads: make([]*ListHead, 0, cfg.ListHeadBufferCap),
	}
}

// Encoder implements io.Writer so it can be passed it into EncodeRLP.
func (self *Encoder) Write(b []byte) (int, error) {
	self.AppendRaw(b...)
	return len(b), nil
}

type ListHead struct {
	ordinal     int
	strpos      int
	base_lhsize int
	elems_size  int
}

// append_size_to writes head to the given buffer, which must be at least
// 9 bytes long. It returns the encoded bytes.
func (self *ListHead) append_size_to(appender func(...byte)) {
	if self.elems_size < 0 {
		panic("the list is not finished")
	}
	puthead(appender, 0xC0, 0xF7, uint64(self.elems_size))
}

func (self *ListHead) Size() int {
	if self.elems_size == -1 {
		panic("list is not closed")
	}
	return self.elems_size + headsize(uint64(self.elems_size))
}

func (self *Encoder) ListStart() *ListHead {
	lh := &ListHead{ordinal: len(self.lheads), strpos: len(self.str), base_lhsize: self.lhsize, elems_size: -1}
	self.lheads = append(self.lheads, lh)
	return lh
}

func (self *Encoder) ListEnd(lh *ListHead) {
	if lh.elems_size != -1 {
		panic("list is already closed")
	}
	lh.elems_size = self.Size() - lh.strpos - lh.base_lhsize
	self.lhsize += headsize(uint64(lh.elems_size))
}

func (self *Encoder) EraseSince(lh *ListHead) {
	self.str = self.str[:lh.strpos]
	self.lheads = self.lheads[:lh.ordinal]
	self.lhsize = lh.base_lhsize
	// TODO maybe remove
	lh.ordinal = -2
}

func (self *Encoder) AppendRaw(b ...byte) {
	self.str = append(self.str, b...)
}

func (self *Encoder) AppendEmptyString() {
	self.AppendRaw(EmptyString)
}

func (self *Encoder) AppendString(str []byte) {
	if len(str) == 0 {
		self.AppendEmptyString()
		return
	}
	if len(str) == 1 && str[0] <= 0x7F {
		// fits single byte, no string header
		self.AppendRaw(str[0])
	} else {
		self.encodeStringHeader(len(str))
		self.AppendRaw(str...)
	}
}

func (self *Encoder) AppendUint(i uint64) {
	if i == 0 {
		self.AppendEmptyString()
	} else if i < 128 {
		self.AppendRaw(byte(i))
	} else {
		putint(self.AppendRaw, i, EmptyString)
	}
}

func (self *Encoder) AppendBigInt(i *big.Int) error {
	if cmp := i.Cmp(common.Big0); cmp == -1 {
		return fmt.Errorf("rlp: cannot encode negative *big.Int")
	} else if cmp == 0 {
		self.AppendEmptyString()
	} else {
		self.AppendString(i.Bytes())
	}
	return nil
}

func (self *Encoder) AppendAny(val interface{}) error {
	rval := reflect.ValueOf(val)
	ti, err := cachedTypeInfo(rval.Type(), tags{})
	if err != nil {
		return err
	}
	return ti.writer(rval, self)
}

func (self *Encoder) Size() int {
	return len(self.str) + self.lhsize
}

func (self *Encoder) Reset() {
	self.str = self.str[:0]
	self.lheads = self.lheads[:0]
	self.lhsize = 0
}

func (self *Encoder) Flush(since *ListHead, appender func(...byte)) {
	strpos := 0
	lheads_offset := 0
	if since != nil {
		strpos = since.strpos
		lheads_offset = since.ordinal + 1
		since.append_size_to(appender)
	}
	for _, head := range self.lheads[lheads_offset:] {
		appender(self.str[strpos:head.strpos]...)
		strpos = head.strpos
		head.append_size_to(appender)
	}
	appender(self.str[strpos:]...)
}

func (self *Encoder) FlushToBytes(since *ListHead, buf []byte) []byte {
	self.Flush(since, func(b ...byte) {
		buf = append(buf, b...)
	})
	return buf
}

func (self *Encoder) ToBytes(since *ListHead) []byte {
	capacity := self.Size()
	if since != nil {
		capacity -= (since.strpos + since.base_lhsize)
	}
	return self.FlushToBytes(since, make([]byte, 0, capacity))
}

func (self *Encoder) encodeStringHeader(size int) {
	if size < 56 {
		self.AppendRaw(EmptyString + byte(size))
	} else {
		putint(self.AppendRaw, uint64(size), 0xB7)
	}
}

// headsize returns the size of a list or string header
// for a value of the given size.
func headsize(size uint64) int {
	if size < 56 {
		return 1
	}
	return 1 + intsize(size)
}

// puthead writes a list or string header to buf.
// buf must be at least 9 bytes long.
func puthead(appender func(...byte), smalltag, largetag byte, size uint64) {
	if size < 56 {
		appender(smalltag + byte(size))
	} else {
		putint(appender, size, largetag)
	}
}

// putint writes i to the beginning of b in big endian byte
// order, using the least number of bytes needed to represent i.
func putint(appender func(...byte), i uint64, tag byte) {
	switch {
	case i < (1 << 8):
		appender(1+tag, byte(i))
	case i < (1 << 16):
		appender(2+tag, byte(i>>8), byte(i))
	case i < (1 << 24):
		appender(3+tag, byte(i>>16), byte(i>>8), byte(i))
	case i < (1 << 32):
		appender(4+tag, byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	case i < (1 << 40):
		appender(5+tag, byte(i>>32), byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	case i < (1 << 48):
		appender(6+tag, byte(i>>40), byte(i>>32), byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	case i < (1 << 56):
		appender(7+tag, byte(i>>48), byte(i>>40), byte(i>>32), byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	default:
		appender(8+tag, byte(i>>56), byte(i>>48), byte(i>>40), byte(i>>32), byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	}
}

// intsize computes the minimum number of bytes required to store i.
func intsize(i uint64) int {
	switch {
	case i < (1 << 8):
		return 1
	case i < (1 << 16):
		return 2
	case i < (1 << 24):
		return 3
	case i < (1 << 32):
		return 4
	case i < (1 << 40):
		return 5
	case i < (1 << 48):
		return 6
	case i < (1 << 56):
		return 7
	default:
		return 8
	}
}

// makeWriter creates a writer function for the given type.
func makeWriter(typ reflect.Type, ts tags) (writer, error) {
	kind := typ.Kind()
	switch {
	case typ == rawValueType:
		return writeRawValue, nil
	case typ.Implements(encoderInterface):
		return writeEncodable, nil
	case kind != reflect.Ptr && reflect.PtrTo(typ).Implements(encoderInterface):
		return writeEncodableNoPtr, nil
	case kind == reflect.Interface:
		return writeInterface, nil
	case typ.AssignableTo(reflect.PtrTo(bigInt)):
		return writeBigIntPtr, nil
	case typ.AssignableTo(bigInt):
		return writeBigIntNoPtr, nil
	case isUint(kind):
		return writeUint, nil
	case kind == reflect.Bool:
		return writeBool, nil
	case kind == reflect.String:
		return writeString, nil
	case kind == reflect.Slice && isByte(typ.Elem()):
		return writeBytes, nil
	case kind == reflect.Array && isByte(typ.Elem()):
		return writeByteArray, nil
	case kind == reflect.Slice || kind == reflect.Array:
		return makeSliceWriter(typ, ts)
	case kind == reflect.Struct:
		return makeStructWriter(typ)
	case kind == reflect.Ptr:
		return makePtrWriter(typ)
	default:
		return nil, fmt.Errorf("rlp: type %v is not RLP-serializable", typ)
	}
}

func isByte(typ reflect.Type) bool {
	return typ.Kind() == reflect.Uint8 && !typ.Implements(encoderInterface)
}

func writeRawValue(val reflect.Value, w *Encoder) error {
	w.AppendRaw(val.Bytes()...)
	return nil
}

func writeUint(val reflect.Value, w *Encoder) error {
	w.AppendUint(val.Uint())
	return nil
}

func writeBool(val reflect.Value, w *Encoder) error {
	if val.Bool() {
		w.AppendRaw(0x01)
	} else {
		w.AppendEmptyString()
	}
	return nil
}

func writeBigIntPtr(val reflect.Value, w *Encoder) error {
	ptr := val.Interface().(*big.Int)
	if ptr == nil {
		w.AppendEmptyString()
		return nil
	}
	return w.AppendBigInt(ptr)
}

func writeBigIntNoPtr(val reflect.Value, w *Encoder) error {
	i := val.Interface().(big.Int)
	return w.AppendBigInt(&i)
}

func writeBytes(val reflect.Value, w *Encoder) error {
	w.AppendString(val.Bytes())
	return nil
}

func writeByteArray(val reflect.Value, w *Encoder) error {
	if !val.CanAddr() {
		// Slice requires the value to be addressable.
		// Make it addressable by copying.
		copy := reflect.New(val.Type()).Elem()
		copy.Set(val)
		val = copy
	}
	size := val.Len()
	slice := val.Slice(0, size).Bytes()
	w.AppendString(slice)
	return nil
}

func writeString(val reflect.Value, w *Encoder) error {
	s := val.String()
	w.AppendString(binary.BytesView(s))
	return nil
}

func writeEncodable(val reflect.Value, w *Encoder) error {
	return val.Interface().(RLPEncodable).EncodeRLP(w)
}

// writeEncodableNoPtr handles non-pointer values that implement RLPEncodable
// with a pointer receiver.
func writeEncodableNoPtr(val reflect.Value, w *Encoder) error {
	if !val.CanAddr() {
		// We can't get the address. It would be possible to make the
		// value addressable by creating a shallow copy, but this
		// creates other problems so we're not doing it (yet).
		//
		// package json simply doesn't call MarshalJSON for cases like
		// this, but encodes the value as if it didn't implement the
		// interface. We don't want to handle it that way.
		return fmt.Errorf("rlp: game over: unadressable value of type %v, EncodeRLP is pointer method", val.Type())
	}
	return val.Addr().Interface().(RLPEncodable).EncodeRLP(w)
}

func writeInterface(val reflect.Value, w *Encoder) error {
	if val.IsNil() {
		// Write empty list. This is consistent with the previous RLP
		// encoder that we had and should therefore avoid any
		// problems.
		w.AppendRaw(0xC0)
		return nil
	}
	eval := val.Elem()
	ti, err := cachedTypeInfo(eval.Type(), tags{})
	if err != nil {
		return err
	}
	return ti.writer(eval, w)
}

func makeSliceWriter(typ reflect.Type, ts tags) (writer, error) {
	etypeinfo, err := cachedTypeInfo1(typ.Elem(), tags{})
	if err != nil {
		return nil, err
	}
	writer := func(val reflect.Value, w *Encoder) error {
		if !ts.tail {
			defer w.ListEnd(w.ListStart())
		}
		vlen := val.Len()
		for i := 0; i < vlen; i++ {
			if err := etypeinfo.writer(val.Index(i), w); err != nil {
				return err
			}
		}
		return nil
	}
	return writer, nil
}

func makeStructWriter(typ reflect.Type) (writer, error) {
	fields, err := structFields(typ)
	if err != nil {
		return nil, err
	}
	writer := func(val reflect.Value, w *Encoder) error {
		defer w.ListEnd(w.ListStart())
		for _, f := range fields {
			if err := f.info.writer(val.Field(f.index), w); err != nil {
				return err
			}
		}
		return nil
	}
	return writer, nil
}

func makePtrWriter(typ reflect.Type) (writer, error) {
	etypeinfo, err := cachedTypeInfo1(typ.Elem(), tags{})
	if err != nil {
		return nil, err
	}

	// determine nil pointer handler
	var nilfunc func(*Encoder) error
	kind := typ.Elem().Kind()
	switch {
	case kind == reflect.Array && isByte(typ.Elem().Elem()):
		nilfunc = func(w *Encoder) error {
			w.AppendRaw(EmptyString)
			return nil
		}
	case kind == reflect.Struct || kind == reflect.Array:
		nilfunc = func(w *Encoder) error {
			// encoding the zero value of a struct/array could trigger
			// infinite recursion, avoid that.
			w.ListEnd(w.ListStart())
			return nil
		}
	default:
		zero := reflect.Zero(typ.Elem())
		nilfunc = func(w *Encoder) error {
			return etypeinfo.writer(zero, w)
		}
	}

	writer := func(val reflect.Value, w *Encoder) error {
		if val.IsNil() {
			return nilfunc(w)
		}
		return etypeinfo.writer(val.Elem(), w)
	}
	return writer, err
}
