// Code generated by protoc-gen-go. DO NOT EDIT.
// source: common.proto

package grpc_go

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type VmId struct {
	ContractAddr         []byte   `protobuf:"bytes,1,opt,name=contractAddr,proto3" json:"contractAddr,omitempty"`
	ProcessId            string   `protobuf:"bytes,2,opt,name=processId,proto3" json:"processId,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *VmId) Reset()         { *m = VmId{} }
func (m *VmId) String() string { return proto.CompactTextString(m) }
func (*VmId) ProtoMessage()    {}
func (*VmId) Descriptor() ([]byte, []int) {
	return fileDescriptor_common_894cc49d171d3f58, []int{0}
}
func (m *VmId) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VmId.Unmarshal(m, b)
}
func (m *VmId) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VmId.Marshal(b, m, deterministic)
}
func (dst *VmId) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VmId.Merge(dst, src)
}
func (m *VmId) XXX_Size() int {
	return xxx_messageInfo_VmId.Size(m)
}
func (m *VmId) XXX_DiscardUnknown() {
	xxx_messageInfo_VmId.DiscardUnknown(m)
}

var xxx_messageInfo_VmId proto.InternalMessageInfo

func (m *VmId) GetContractAddr() []byte {
	if m != nil {
		return m.ContractAddr
	}
	return nil
}

func (m *VmId) GetProcessId() string {
	if m != nil {
		return m.ProcessId
	}
	return ""
}

type BytesMessage struct {
	VmId                 *VmId    `protobuf:"bytes,1,opt,name=vmId,proto3" json:"vmId,omitempty"`
	Value                []byte   `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BytesMessage) Reset()         { *m = BytesMessage{} }
func (m *BytesMessage) String() string { return proto.CompactTextString(m) }
func (*BytesMessage) ProtoMessage()    {}
func (*BytesMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_common_894cc49d171d3f58, []int{1}
}
func (m *BytesMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BytesMessage.Unmarshal(m, b)
}
func (m *BytesMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BytesMessage.Marshal(b, m, deterministic)
}
func (dst *BytesMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BytesMessage.Merge(dst, src)
}
func (m *BytesMessage) XXX_Size() int {
	return xxx_messageInfo_BytesMessage.Size(m)
}
func (m *BytesMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_BytesMessage.DiscardUnknown(m)
}

var xxx_messageInfo_BytesMessage proto.InternalMessageInfo

func (m *BytesMessage) GetVmId() *VmId {
	if m != nil {
		return m.VmId
	}
	return nil
}

func (m *BytesMessage) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

type BoolMessage struct {
	VmId                 *VmId    `protobuf:"bytes,1,opt,name=vmId,proto3" json:"vmId,omitempty"`
	Value                bool     `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BoolMessage) Reset()         { *m = BoolMessage{} }
func (m *BoolMessage) String() string { return proto.CompactTextString(m) }
func (*BoolMessage) ProtoMessage()    {}
func (*BoolMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_common_894cc49d171d3f58, []int{2}
}
func (m *BoolMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BoolMessage.Unmarshal(m, b)
}
func (m *BoolMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BoolMessage.Marshal(b, m, deterministic)
}
func (dst *BoolMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BoolMessage.Merge(dst, src)
}
func (m *BoolMessage) XXX_Size() int {
	return xxx_messageInfo_BoolMessage.Size(m)
}
func (m *BoolMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_BoolMessage.DiscardUnknown(m)
}

var xxx_messageInfo_BoolMessage proto.InternalMessageInfo

func (m *BoolMessage) GetVmId() *VmId {
	if m != nil {
		return m.VmId
	}
	return nil
}

func (m *BoolMessage) GetValue() bool {
	if m != nil {
		return m.Value
	}
	return false
}

func init() {
	proto.RegisterType((*VmId)(nil), "taraxa.vm.statedb.VmId")
	proto.RegisterType((*BytesMessage)(nil), "taraxa.vm.statedb.BytesMessage")
	proto.RegisterType((*BoolMessage)(nil), "taraxa.vm.statedb.BoolMessage")
}

func init() { proto.RegisterFile("common.proto", fileDescriptor_common_894cc49d171d3f58) }

var fileDescriptor_common_894cc49d171d3f58 = []byte{
	// 193 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x49, 0xce, 0xcf, 0xcd,
	0xcd, 0xcf, 0xd3, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0x2c, 0x49, 0x2c, 0x4a, 0xac, 0x48,
	0xd4, 0x2b, 0xcb, 0xd5, 0x2b, 0x2e, 0x49, 0x2c, 0x49, 0x4d, 0x49, 0x52, 0xf2, 0xe0, 0x62, 0x09,
	0xcb, 0xf5, 0x4c, 0x11, 0x52, 0x02, 0x29, 0xcd, 0x2b, 0x29, 0x4a, 0x4c, 0x2e, 0x71, 0x4c, 0x49,
	0x29, 0x92, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x09, 0x42, 0x11, 0x13, 0x92, 0xe1, 0xe2, 0x2c, 0x28,
	0xca, 0x4f, 0x4e, 0x2d, 0x2e, 0xf6, 0x4c, 0x91, 0x60, 0x52, 0x60, 0xd4, 0xe0, 0x0c, 0x42, 0x08,
	0x28, 0x05, 0x72, 0xf1, 0x38, 0x55, 0x96, 0xa4, 0x16, 0xfb, 0xa6, 0x16, 0x17, 0x27, 0xa6, 0xa7,
	0x0a, 0x69, 0x73, 0xb1, 0x94, 0xe5, 0x7a, 0xa6, 0x80, 0x4d, 0xe2, 0x36, 0x12, 0xd7, 0xc3, 0xb0,
	0x5b, 0x0f, 0x64, 0x71, 0x10, 0x58, 0x91, 0x90, 0x08, 0x17, 0x6b, 0x59, 0x62, 0x4e, 0x69, 0x2a,
	0xd8, 0x58, 0x9e, 0x20, 0x08, 0x47, 0x29, 0x80, 0x8b, 0xdb, 0x29, 0x3f, 0x3f, 0x87, 0x72, 0x13,
	0x39, 0xa0, 0x26, 0x3a, 0x71, 0x46, 0xb1, 0xa7, 0x17, 0x15, 0x24, 0xc7, 0xa7, 0xe7, 0x27, 0xb1,
	0x81, 0xc3, 0xc4, 0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0x4f, 0xd4, 0x25, 0x56, 0x23, 0x01, 0x00,
	0x00,
}