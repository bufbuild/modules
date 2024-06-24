// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: state/v1alpha1/state.proto

package v1alpha1

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// GlobalState is a sorted array managed modules, each one with the latest
// reference from its local array of references. This is kept updated in a
// global state file at the root sync directory.
type GlobalState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Modules []*GlobalStateReference `protobuf:"bytes,1,rep,name=modules,proto3" json:"modules,omitempty"`
}

func (x *GlobalState) Reset() {
	*x = GlobalState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_v1alpha1_state_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GlobalState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GlobalState) ProtoMessage() {}

func (x *GlobalState) ProtoReflect() protoreflect.Message {
	mi := &file_state_v1alpha1_state_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GlobalState.ProtoReflect.Descriptor instead.
func (*GlobalState) Descriptor() ([]byte, []int) {
	return file_state_v1alpha1_state_proto_rawDescGZIP(), []int{0}
}

func (x *GlobalState) GetModules() []*GlobalStateReference {
	if x != nil {
		return x.Modules
	}
	return nil
}

// GlobalReference is a single managed module reference with the latest
// reference from its local array of references.
type GlobalStateReference struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ModuleName      string `protobuf:"bytes,1,opt,name=module_name,json=moduleName,proto3" json:"module_name,omitempty"`
	LatestReference string `protobuf:"bytes,2,opt,name=latest_reference,json=latestReference,proto3" json:"latest_reference,omitempty"`
}

func (x *GlobalStateReference) Reset() {
	*x = GlobalStateReference{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_v1alpha1_state_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GlobalStateReference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GlobalStateReference) ProtoMessage() {}

func (x *GlobalStateReference) ProtoReflect() protoreflect.Message {
	mi := &file_state_v1alpha1_state_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GlobalStateReference.ProtoReflect.Descriptor instead.
func (*GlobalStateReference) Descriptor() ([]byte, []int) {
	return file_state_v1alpha1_state_proto_rawDescGZIP(), []int{1}
}

func (x *GlobalStateReference) GetModuleName() string {
	if x != nil {
		return x.ModuleName
	}
	return ""
}

func (x *GlobalStateReference) GetLatestReference() string {
	if x != nil {
		return x.LatestReference
	}
	return ""
}

// ModuleState is an array of references that will be synced to a BSR cluster for a
// managed module. This is kept updated in a state file at the managed module
// directory.
type ModuleState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	References []*ModuleReference `protobuf:"bytes,1,rep,name=references,proto3" json:"references,omitempty"`
}

func (x *ModuleState) Reset() {
	*x = ModuleState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_v1alpha1_state_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModuleState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleState) ProtoMessage() {}

func (x *ModuleState) ProtoReflect() protoreflect.Message {
	mi := &file_state_v1alpha1_state_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModuleState.ProtoReflect.Descriptor instead.
func (*ModuleState) Descriptor() ([]byte, []int) {
	return file_state_v1alpha1_state_proto_rawDescGZIP(), []int{2}
}

func (x *ModuleState) GetReferences() []*ModuleReference {
	if x != nil {
		return x.References
	}
	return nil
}

// ModuleReference is a single git reference of a managed module that will be
// synced to a BSR cluster.
type ModuleReference struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name   string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Digest string `protobuf:"bytes,2,opt,name=digest,proto3" json:"digest,omitempty"`
}

func (x *ModuleReference) Reset() {
	*x = ModuleReference{}
	if protoimpl.UnsafeEnabled {
		mi := &file_state_v1alpha1_state_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModuleReference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleReference) ProtoMessage() {}

func (x *ModuleReference) ProtoReflect() protoreflect.Message {
	mi := &file_state_v1alpha1_state_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModuleReference.ProtoReflect.Descriptor instead.
func (*ModuleReference) Descriptor() ([]byte, []int) {
	return file_state_v1alpha1_state_proto_rawDescGZIP(), []int{3}
}

func (x *ModuleReference) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ModuleReference) GetDigest() string {
	if x != nil {
		return x.Digest
	}
	return ""
}

var File_state_v1alpha1_state_proto protoreflect.FileDescriptor

var file_state_v1alpha1_state_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x2f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0e, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x1b, 0x62, 0x75,
	0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64,
	0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xbd, 0x03, 0x0a, 0x0b, 0x47, 0x6c,
	0x6f, 0x62, 0x61, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x3e, 0x0a, 0x07, 0x6d, 0x6f, 0x64,
	0x75, 0x6c, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x73, 0x74, 0x61,
	0x74, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x6c, 0x6f, 0x62,
	0x61, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65,
	0x52, 0x07, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x73, 0x3a, 0xed, 0x02, 0xba, 0x48, 0xe9, 0x02,
	0x1a, 0xe6, 0x02, 0x0a, 0x20, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x2e, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x5f, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f,
	0x6e, 0x61, 0x6d, 0x65, 0x73, 0x1a, 0xc1, 0x02, 0x73, 0x69, 0x7a, 0x65, 0x28, 0x74, 0x68, 0x69,
	0x73, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72,
	0x28, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2c, 0x20, 0x21, 0x74, 0x68, 0x69,
	0x73, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x73, 0x2e, 0x6d, 0x61, 0x70, 0x28, 0x69, 0x2c,
	0x20, 0x69, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x29, 0x2e,
	0x65, 0x78, 0x69, 0x73, 0x74, 0x73, 0x5f, 0x6f, 0x6e, 0x65, 0x28, 0x78, 0x2c, 0x20, 0x78, 0x20,
	0x3d, 0x3d, 0x20, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2e, 0x6d, 0x6f, 0x64,
	0x75, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x29, 0x29, 0x29, 0x20, 0x3d, 0x3d, 0x20, 0x30,
	0x20, 0x3f, 0x20, 0x27, 0x27, 0x20, 0x3a, 0x20, 0x28, 0x74, 0x68, 0x69, 0x73, 0x2e, 0x6d, 0x6f,
	0x64, 0x75, 0x6c, 0x65, 0x73, 0x2e, 0x6d, 0x61, 0x70, 0x28, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65,
	0x6e, 0x63, 0x65, 0x2c, 0x20, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2e, 0x6d,
	0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x29, 0x2e, 0x66, 0x69, 0x6c, 0x74,
	0x65, 0x72, 0x28, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x2c, 0x20,
	0x21, 0x74, 0x68, 0x69, 0x73, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x73, 0x2e, 0x6d, 0x61,
	0x70, 0x28, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2c, 0x20, 0x72, 0x65, 0x66,
	0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x29, 0x2e, 0x65, 0x78, 0x69, 0x73, 0x74, 0x73, 0x5f, 0x6f, 0x6e, 0x65, 0x28, 0x78,
	0x2c, 0x20, 0x78, 0x20, 0x3d, 0x3d, 0x20, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x29, 0x29, 0x29, 0x5b, 0x30, 0x5d, 0x20, 0x2b, 0x20, 0x27, 0x20, 0x68, 0x61, 0x73,
	0x20, 0x61, 0x70, 0x70, 0x65, 0x61, 0x72, 0x65, 0x64, 0x20, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x70,
	0x6c, 0x65, 0x20, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x27, 0x22, 0x72, 0x0a, 0x14, 0x47, 0x6c, 0x6f,
	0x62, 0x61, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63,
	0x65, 0x12, 0x27, 0x0a, 0x0b, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x06, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x0a,
	0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x31, 0x0a, 0x10, 0x6c, 0x61,
	0x74, 0x65, 0x73, 0x74, 0x5f, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x42, 0x06, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x0f, 0x6c, 0x61,
	0x74, 0x65, 0x73, 0x74, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x22, 0x9e, 0x03,
	0x0a, 0x0b, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x3f, 0x0a,
	0x0a, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x1f, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x2e, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e,
	0x63, 0x65, 0x52, 0x0a, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x73, 0x3a, 0xcd,
	0x02, 0xba, 0x48, 0xc9, 0x02, 0x1a, 0xc6, 0x02, 0x0a, 0x1e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65,
	0x5f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x5f, 0x72, 0x65,
	0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x73, 0x1a, 0xa3, 0x02, 0x73, 0x69, 0x7a, 0x65, 0x28,
	0x74, 0x68, 0x69, 0x73, 0x2e, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x73, 0x2e,
	0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x28, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65,
	0x2c, 0x20, 0x21, 0x74, 0x68, 0x69, 0x73, 0x2e, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63,
	0x65, 0x73, 0x2e, 0x6d, 0x61, 0x70, 0x28, 0x69, 0x2c, 0x20, 0x69, 0x2e, 0x6e, 0x61, 0x6d, 0x65,
	0x29, 0x2e, 0x65, 0x78, 0x69, 0x73, 0x74, 0x73, 0x5f, 0x6f, 0x6e, 0x65, 0x28, 0x78, 0x2c, 0x20,
	0x78, 0x20, 0x3d, 0x3d, 0x20, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2e, 0x6e,
	0x61, 0x6d, 0x65, 0x29, 0x29, 0x29, 0x20, 0x3d, 0x3d, 0x20, 0x30, 0x20, 0x3f, 0x20, 0x27, 0x27,
	0x20, 0x3a, 0x20, 0x28, 0x74, 0x68, 0x69, 0x73, 0x2e, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e,
	0x63, 0x65, 0x73, 0x2e, 0x6d, 0x61, 0x70, 0x28, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63,
	0x65, 0x2c, 0x20, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2e, 0x6e, 0x61, 0x6d,
	0x65, 0x29, 0x2e, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x28, 0x6e, 0x61, 0x6d, 0x65, 0x2c, 0x20,
	0x21, 0x74, 0x68, 0x69, 0x73, 0x2e, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x73,
	0x2e, 0x6d, 0x61, 0x70, 0x28, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2c, 0x20,
	0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2e, 0x6e, 0x61, 0x6d, 0x65, 0x29, 0x2e,
	0x65, 0x78, 0x69, 0x73, 0x74, 0x73, 0x5f, 0x6f, 0x6e, 0x65, 0x28, 0x78, 0x2c, 0x20, 0x78, 0x20,
	0x3d, 0x3d, 0x20, 0x6e, 0x61, 0x6d, 0x65, 0x29, 0x29, 0x29, 0x5b, 0x30, 0x5d, 0x20, 0x2b, 0x20,
	0x27, 0x20, 0x68, 0x61, 0x73, 0x20, 0x61, 0x70, 0x70, 0x65, 0x61, 0x72, 0x65, 0x64, 0x20, 0x6d,
	0x75, 0x6c, 0x74, 0x69, 0x70, 0x6c, 0x65, 0x20, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x27, 0x22, 0x4d,
	0x0a, 0x0f, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63,
	0x65, 0x12, 0x1a, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42,
	0x06, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1e, 0x0a,
	0x06, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x06, 0xba,
	0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x06, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x42, 0x4b, 0x5a,
	0x49, 0x62, 0x75, 0x66, 0x2e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x67,
	0x6f, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x6d, 0x6f, 0x64, 0x75, 0x6c,
	0x65, 0x73, 0x2d, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f,
	0x6c, 0x62, 0x75, 0x66, 0x66, 0x65, 0x72, 0x73, 0x2f, 0x67, 0x6f, 0x2f, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_state_v1alpha1_state_proto_rawDescOnce sync.Once
	file_state_v1alpha1_state_proto_rawDescData = file_state_v1alpha1_state_proto_rawDesc
)

func file_state_v1alpha1_state_proto_rawDescGZIP() []byte {
	file_state_v1alpha1_state_proto_rawDescOnce.Do(func() {
		file_state_v1alpha1_state_proto_rawDescData = protoimpl.X.CompressGZIP(file_state_v1alpha1_state_proto_rawDescData)
	})
	return file_state_v1alpha1_state_proto_rawDescData
}

var file_state_v1alpha1_state_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_state_v1alpha1_state_proto_goTypes = []any{
	(*GlobalState)(nil),          // 0: state.v1alpha1.GlobalState
	(*GlobalStateReference)(nil), // 1: state.v1alpha1.GlobalStateReference
	(*ModuleState)(nil),          // 2: state.v1alpha1.ModuleState
	(*ModuleReference)(nil),      // 3: state.v1alpha1.ModuleReference
}
var file_state_v1alpha1_state_proto_depIdxs = []int32{
	1, // 0: state.v1alpha1.GlobalState.modules:type_name -> state.v1alpha1.GlobalStateReference
	3, // 1: state.v1alpha1.ModuleState.references:type_name -> state.v1alpha1.ModuleReference
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_state_v1alpha1_state_proto_init() }
func file_state_v1alpha1_state_proto_init() {
	if File_state_v1alpha1_state_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_state_v1alpha1_state_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*GlobalState); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_state_v1alpha1_state_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*GlobalStateReference); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_state_v1alpha1_state_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*ModuleState); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_state_v1alpha1_state_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*ModuleReference); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_state_v1alpha1_state_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_state_v1alpha1_state_proto_goTypes,
		DependencyIndexes: file_state_v1alpha1_state_proto_depIdxs,
		MessageInfos:      file_state_v1alpha1_state_proto_msgTypes,
	}.Build()
	File_state_v1alpha1_state_proto = out.File
	file_state_v1alpha1_state_proto_rawDesc = nil
	file_state_v1alpha1_state_proto_goTypes = nil
	file_state_v1alpha1_state_proto_depIdxs = nil
}
