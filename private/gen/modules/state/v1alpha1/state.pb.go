// Copyright 2021-2025 Buf Technologies, Inc.
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
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: state/v1alpha1/state.proto

package v1alpha1

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	unsafe "unsafe"
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
	state              protoimpl.MessageState   `protogen:"opaque.v1"`
	xxx_hidden_Modules *[]*GlobalStateReference `protobuf:"bytes,1,rep,name=modules,proto3"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *GlobalState) Reset() {
	*x = GlobalState{}
	mi := &file_state_v1alpha1_state_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GlobalState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GlobalState) ProtoMessage() {}

func (x *GlobalState) ProtoReflect() protoreflect.Message {
	mi := &file_state_v1alpha1_state_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GlobalState) GetModules() []*GlobalStateReference {
	if x != nil {
		if x.xxx_hidden_Modules != nil {
			return *x.xxx_hidden_Modules
		}
	}
	return nil
}

func (x *GlobalState) SetModules(v []*GlobalStateReference) {
	x.xxx_hidden_Modules = &v
}

type GlobalState_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Modules []*GlobalStateReference
}

func (b0 GlobalState_builder) Build() *GlobalState {
	m0 := &GlobalState{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Modules = &b.Modules
	return m0
}

// GlobalReference is a single managed module reference with the latest
// reference from its local array of references.
type GlobalStateReference struct {
	state                      protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_ModuleName      string                 `protobuf:"bytes,1,opt,name=module_name,json=moduleName,proto3"`
	xxx_hidden_LatestReference string                 `protobuf:"bytes,2,opt,name=latest_reference,json=latestReference,proto3"`
	unknownFields              protoimpl.UnknownFields
	sizeCache                  protoimpl.SizeCache
}

func (x *GlobalStateReference) Reset() {
	*x = GlobalStateReference{}
	mi := &file_state_v1alpha1_state_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GlobalStateReference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GlobalStateReference) ProtoMessage() {}

func (x *GlobalStateReference) ProtoReflect() protoreflect.Message {
	mi := &file_state_v1alpha1_state_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GlobalStateReference) GetModuleName() string {
	if x != nil {
		return x.xxx_hidden_ModuleName
	}
	return ""
}

func (x *GlobalStateReference) GetLatestReference() string {
	if x != nil {
		return x.xxx_hidden_LatestReference
	}
	return ""
}

func (x *GlobalStateReference) SetModuleName(v string) {
	x.xxx_hidden_ModuleName = v
}

func (x *GlobalStateReference) SetLatestReference(v string) {
	x.xxx_hidden_LatestReference = v
}

type GlobalStateReference_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	ModuleName      string
	LatestReference string
}

func (b0 GlobalStateReference_builder) Build() *GlobalStateReference {
	m0 := &GlobalStateReference{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_ModuleName = b.ModuleName
	x.xxx_hidden_LatestReference = b.LatestReference
	return m0
}

// ModuleState is an array of references that will be synced to a BSR cluster for a
// managed module. This is kept updated in a state file at the managed module
// directory.
type ModuleState struct {
	state                 protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_References *[]*ModuleReference    `protobuf:"bytes,1,rep,name=references,proto3"`
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *ModuleState) Reset() {
	*x = ModuleState{}
	mi := &file_state_v1alpha1_state_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ModuleState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleState) ProtoMessage() {}

func (x *ModuleState) ProtoReflect() protoreflect.Message {
	mi := &file_state_v1alpha1_state_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ModuleState) GetReferences() []*ModuleReference {
	if x != nil {
		if x.xxx_hidden_References != nil {
			return *x.xxx_hidden_References
		}
	}
	return nil
}

func (x *ModuleState) SetReferences(v []*ModuleReference) {
	x.xxx_hidden_References = &v
}

type ModuleState_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	References []*ModuleReference
}

func (b0 ModuleState_builder) Build() *ModuleState {
	m0 := &ModuleState{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_References = &b.References
	return m0
}

// ModuleReference is a single git reference of a managed module that will be
// synced to a BSR cluster.
type ModuleReference struct {
	state             protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Name   string                 `protobuf:"bytes,1,opt,name=name,proto3"`
	xxx_hidden_Digest string                 `protobuf:"bytes,2,opt,name=digest,proto3"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *ModuleReference) Reset() {
	*x = ModuleReference{}
	mi := &file_state_v1alpha1_state_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ModuleReference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleReference) ProtoMessage() {}

func (x *ModuleReference) ProtoReflect() protoreflect.Message {
	mi := &file_state_v1alpha1_state_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ModuleReference) GetName() string {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return ""
}

func (x *ModuleReference) GetDigest() string {
	if x != nil {
		return x.xxx_hidden_Digest
	}
	return ""
}

func (x *ModuleReference) SetName(v string) {
	x.xxx_hidden_Name = v
}

func (x *ModuleReference) SetDigest(v string) {
	x.xxx_hidden_Digest = v
}

type ModuleReference_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Name   string
	Digest string
}

func (b0 ModuleReference_builder) Build() *ModuleReference {
	m0 := &ModuleReference{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Name = b.Name
	x.xxx_hidden_Digest = b.Digest
	return m0
}

var File_state_v1alpha1_state_proto protoreflect.FileDescriptor

const file_state_v1alpha1_state_proto_rawDesc = "" +
	"\n" +
	"\x1astate/v1alpha1/state.proto\x12\x0estate.v1alpha1\x1a\x1bbuf/validate/validate.proto\"\x81\x03\n" +
	"\vGlobalState\x12>\n" +
	"\amodules\x18\x01 \x03(\v2$.state.v1alpha1.GlobalStateReferenceR\amodules:\xb1\x02\xbaH\xad\x02\x1a\xaa\x02\n" +
	" module_state.unique_module_names\x1a\x85\x02this.modules.map(i, i.module_name).unique() ? '' : 'module name ' + (this.modules.map(reference, reference.module_name).filter(module_name, !this.modules.map(reference, reference.module_name).exists_one(x, x == module_name)))[0] + ' has appeared multiple times'\"r\n" +
	"\x14GlobalStateReference\x12'\n" +
	"\vmodule_name\x18\x01 \x01(\tB\x06\xbaH\x03\xc8\x01\x01R\n" +
	"moduleName\x121\n" +
	"\x10latest_reference\x18\x02 \x01(\tB\x06\xbaH\x03\xc8\x01\x01R\x0flatestReference\"\xe4\x02\n" +
	"\vModuleState\x12?\n" +
	"\n" +
	"references\x18\x01 \x03(\v2\x1f.state.v1alpha1.ModuleReferenceR\n" +
	"references:\x93\x02\xbaH\x8f\x02\x1a\x8c\x02\n" +
	"\x1emodule_state.unique_references\x1a\xe9\x01this.references.map(i, i.name).unique() ? '' : 'reference ' + (this.references.map(reference, reference.name).filter(name, !this.references.map(reference, reference.name).exists_one(x, x == name)))[0] + ' has appeared multiple times'\"M\n" +
	"\x0fModuleReference\x12\x1a\n" +
	"\x04name\x18\x01 \x01(\tB\x06\xbaH\x03\xc8\x01\x01R\x04name\x12\x1e\n" +
	"\x06digest\x18\x02 \x01(\tB\x06\xbaH\x03\xc8\x01\x01R\x06digestBMZKbuf.build/gen/go/bufbuild/managed-modules/protocolbuffers/go/state/v1alpha1b\x06proto3"

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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_state_v1alpha1_state_proto_rawDesc), len(file_state_v1alpha1_state_proto_rawDesc)),
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
	file_state_v1alpha1_state_proto_goTypes = nil
	file_state_v1alpha1_state_proto_depIdxs = nil
}
