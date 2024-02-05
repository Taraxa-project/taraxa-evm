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
	"io"
	"math"
	"math/big"
	"reflect"
	"sync"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

const EmptyString = byte(0x80)

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
// Not supported: signed integers, floating point numbers, channels and functions.
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
	eb.Flush(-1, func(b ...byte) {
		if _, err = w.Write(b); err != nil {
			panic(err)
		}
	})
	return
}

// EncodeToBytes returns the RLP encoding of val.
// Please see the documentation of Encode for the encoding rules.
func EncodeToBytes(val interface{}) (ret []byte, err error) {
	eb := encbufPool.Get().(*Encoder)
	defer encbufPool.Put(eb)
	eb.Reset()
	if err = eb.AppendAny(val); err == nil {
		ret = eb.ToBytes(-1)
	}
	return
}

func MustEncodeToBytes(val interface{}) (ret []byte) {
	var err error
	ret, err = EncodeToBytes(val)
	util.PanicIfNotNil(err)
	return
}

func BytesAppender(b *[]byte) func(...byte) {
	return func(bs ...byte) {
		*b = append(*b, bs...)
	}
}

type Encoder struct {
	str    []byte     // string data, contains everything except list headers
	lheads []ListHead // all list headers
	lhsize uint32     // sum of sizes of all encoded list headers
}
type ListHead struct {
	strpos      uint32
	base_lhsize uint32
	elems_size  uint32
	closed      bool
}

func (self *Encoder) Reset() {
	self.str = self.str[:0]
	self.lheads = self.lheads[:0]
	self.lhsize = 0
}

func (self *Encoder) ResizeReset(string_buf_cap, list_buf_cap int) {
	self.str = make([]byte, 0, string_buf_cap)
	self.lheads = make([]ListHead, 0, list_buf_cap)
	self.lhsize = 0
}

func (self *Encoder) BufferSizes() (strbuf_size, listbuf_size int) {
	return len(self.str), len(self.lheads)
}

func (self *Encoder) ListsCap() int {
	return cap(self.lheads)
}

func (self *Encoder) Size() uint32 {
	return uint32(len(self.str)) + self.lhsize
}

func (self *Encoder) ListSize(list_pos int) uint32 {
	if lh := self.lheads[list_pos]; lh.closed {
		return lh.elems_size + uint32(headsize(uint64(lh.elems_size)))
	}
	panic("list is not closed")
}

func (self *Encoder) ListStart() (list_pos int) {
	list_pos = len(self.lheads)
	self.lheads = append(self.lheads, ListHead{strpos: uint32(len(self.str)), base_lhsize: self.lhsize})
	return
}

func (self *Encoder) ListEnd(list_pos int) {
	if lh := &self.lheads[list_pos]; !lh.closed {
		lh.elems_size = self.Size() - lh.strpos - lh.base_lhsize
		self.lhsize += uint32(headsize(uint64(lh.elems_size)))
		lh.closed = true
		return
	}
	panic("list is already closed")
}

func (self *Encoder) RevertToListStart(list_pos int) {
	lh := self.lheads[list_pos]
	self.lheads = self.lheads[:list_pos]
	self.str = self.str[:lh.strpos]
	self.lhsize = lh.base_lhsize
}

func (self *Encoder) Flush(list_pos int, appender func(...byte)) {
	strpos := uint32(0)
	if list_pos != -1 {
		strpos = self.lheads[list_pos].strpos
	} else {
		list_pos = 0
	}
	for _, lh := range self.lheads[list_pos:] {
		appender(self.str[strpos:lh.strpos]...)
		strpos = lh.strpos
		if !lh.closed {
			panic("the list is not closed")
		}
		puthead(appender, 0xC0, 0xF7, uint64(lh.elems_size))
	}
	appender(self.str[strpos:]...)
}

func (self *Encoder) FlushToBytes(since_list_pos int, buf *[]byte) {
	self.Flush(since_list_pos, func(b ...byte) {
		*buf = append(*buf, b...)
	})
}

func (self *Encoder) ToBytes(since_list_pos int) (ret []byte) {
	capacity := self.Size()
	if since_list_pos != -1 {
		lhead := self.lheads[since_list_pos]
		capacity -= (lhead.strpos + lhead.base_lhsize)
	}
	ret = make([]byte, 0, capacity)
	self.FlushToBytes(since_list_pos, &ret)
	return
}

// Encoder implements io.Writer so it can be passed it into EncodeRLP.
func (self *Encoder) Write(b []byte) (int, error) {
	self.AppendRaw(b...)
	return len(b), nil
}

func (self *Encoder) AppendRaw(b ...byte) {
	asserts.Holds(math.MaxUint32-len(self.str) > len(b))
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
	ToRLPString(str, self.AppendRaw)
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

func (self *Encoder) AppendBigInt(i *big.Int) (err error) {
	if i == nil {
		self.AppendEmptyString()
	} else if sign := i.Sign(); sign == 0 {
		self.AppendEmptyString()
	} else if sign == 1 {
		self.AppendString(i.Bytes())
	} else {
		err = fmt.Errorf("rlp: cannot encode negative *big.Int")
	}
	return
}

func (self *Encoder) AppendAny(val interface{}) error {
	rval := reflect.ValueOf(val)
	ti, err := cachedTypeInfo(rval.Type(), tags{})
	if err != nil {
		return err
	}
	return ti.writer(rval, self)
}

func ToRLPStringSimple(str []byte) (ret []byte) {
	size := len(str)
	ret = make([]byte, 0, size+int(bin.SizeInBytes(uint64(size))+1))
	ToRLPString(str, func(b ...byte) {
		ret = append(ret, b...)
	})
	return ret
}

func ToRLPString(str []byte, appender func(...byte)) {
	if size := len(str); size == 1 && str[0] <= 0x7F {
		// fits single byte, no string header
		appender(str[0])
	} else {
		if size < 56 {
			appender(EmptyString + byte(size))
		} else {
			putint(appender, uint64(size), 0xB7)
		}
		appender(str...)
	}
}

// headsize returns the size of a list or string header
// for a value of the given size.
func headsize(size uint64) byte {
	if size < 56 {
		return 1
	}
	return 1 + bin.SizeInBytes(size)
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
	case typ.AssignableTo(bigint_ptr_t):
		return writeBigIntPtr, nil
	case typ.ConvertibleTo(bigint_ptr_t):
		return func(value reflect.Value, encoder *Encoder) error {
			return writeBigIntPtr(value.Convert(bigint_ptr_t), encoder)
		}, nil
	case typ.AssignableTo(bigInt):
		return func(value reflect.Value, encoder *Encoder) error {
			val := value.Interface().(big.Int)
			return encoder.AppendBigInt(&val)
		}, nil
	case typ.ConvertibleTo(bigInt):
		return func(value reflect.Value, encoder *Encoder) error {
			val := value.Convert(bigInt).Interface().(big.Int)
			return encoder.AppendBigInt(&val)
		}, nil
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
	case kind == reflect.Map:
		return makeMapWriter(typ)
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
	return w.AppendBigInt(val.Interface().(*big.Int))
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
	w.AppendString(bin.BytesView(s))
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

func makeMapWriter(typ reflect.Type) (writer, error) {
	k_type_info, err_0 := cachedTypeInfo1(typ.Key(), tags{})
	util.PanicIfNotNil(err_0)
	v_type_info, err_1 := cachedTypeInfo1(typ.Elem(), tags{})
	util.PanicIfNotNil(err_1)
	return func(value reflect.Value, encoder *Encoder) error {
		list_start := encoder.ListStart()
		for i := value.MapRange(); i.Next(); {
			list_start := encoder.ListStart()
			if err := k_type_info.writer(i.Key(), encoder); err != nil {
				return err
			}
			if err := v_type_info.writer(i.Value(), encoder); err != nil {
				return err
			}
			encoder.ListEnd(list_start)
		}
		encoder.ListEnd(list_start)
		return nil
	}, nil
}
