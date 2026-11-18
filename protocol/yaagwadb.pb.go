package protocol

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Empty struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Empty) Reset() {
	*x = Empty{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{0}
}

type Attributes struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Inode         uint64                 `protobuf:"varint,1,opt,name=inode,proto3" json:"inode,omitempty"`
	ParentInode   uint64                 `protobuf:"varint,2,opt,name=parent_inode,json=parentInode,proto3" json:"parent_inode,omitempty"`
	Name          string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Mode          uint32                 `protobuf:"varint,4,opt,name=mode,proto3" json:"mode,omitempty"`
	Uid           uint32                 `protobuf:"varint,5,opt,name=uid,proto3" json:"uid,omitempty"`
	Gid           uint32                 `protobuf:"varint,6,opt,name=gid,proto3" json:"gid,omitempty"`
	Size          uint64                 `protobuf:"varint,7,opt,name=size,proto3" json:"size,omitempty"`
	AccessTime    int64                  `protobuf:"varint,8,opt,name=access_time,json=accessTime,proto3" json:"access_time,omitempty"` // UNIX nanoseconds
	ModifyTime    int64                  `protobuf:"varint,9,opt,name=modify_time,json=modifyTime,proto3" json:"modify_time,omitempty"`
	ChangeTime    int64                  `protobuf:"varint,10,opt,name=change_time,json=changeTime,proto3" json:"change_time,omitempty"`
	Nlink         uint32                 `protobuf:"varint,11,opt,name=nlink,proto3" json:"nlink,omitempty"` // always 1 (2 for directories)
	Rdev          uint32                 `protobuf:"varint,12,opt,name=rdev,proto3" json:"rdev,omitempty"`   // always 0
	Blocks        uint64                 `protobuf:"varint,13,opt,name=blocks,proto3" json:"blocks,omitempty"`
	BlockSize     uint32                 `protobuf:"varint,14,opt,name=block_size,json=blockSize,proto3" json:"block_size,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Attributes) Reset() {
	*x = Attributes{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Attributes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Attributes) ProtoMessage() {}

func (x *Attributes) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Attributes.ProtoReflect.Descriptor instead.
func (*Attributes) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{1}
}

func (x *Attributes) GetInode() uint64 {
	if x != nil {
		return x.Inode
	}
	return 0
}

func (x *Attributes) GetParentInode() uint64 {
	if x != nil {
		return x.ParentInode
	}
	return 0
}

func (x *Attributes) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Attributes) GetMode() uint32 {
	if x != nil {
		return x.Mode
	}
	return 0
}

func (x *Attributes) GetUid() uint32 {
	if x != nil {
		return x.Uid
	}
	return 0
}

func (x *Attributes) GetGid() uint32 {
	if x != nil {
		return x.Gid
	}
	return 0
}

func (x *Attributes) GetSize() uint64 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *Attributes) GetAccessTime() int64 {
	if x != nil {
		return x.AccessTime
	}
	return 0
}

func (x *Attributes) GetModifyTime() int64 {
	if x != nil {
		return x.ModifyTime
	}
	return 0
}

func (x *Attributes) GetChangeTime() int64 {
	if x != nil {
		return x.ChangeTime
	}
	return 0
}

func (x *Attributes) GetNlink() uint32 {
	if x != nil {
		return x.Nlink
	}
	return 0
}

func (x *Attributes) GetRdev() uint32 {
	if x != nil {
		return x.Rdev
	}
	return 0
}

func (x *Attributes) GetBlocks() uint64 {
	if x != nil {
		return x.Blocks
	}
	return 0
}

func (x *Attributes) GetBlockSize() uint32 {
	if x != nil {
		return x.BlockSize
	}
	return 0
}

type LookupRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ParentInode   uint64                 `protobuf:"varint,1,opt,name=parent_inode,json=parentInode,proto3" json:"parent_inode,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LookupRequest) Reset() {
	*x = LookupRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LookupRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LookupRequest) ProtoMessage() {}

func (x *LookupRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LookupRequest.ProtoReflect.Descriptor instead.
func (*LookupRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{2}
}

func (x *LookupRequest) GetParentInode() uint64 {
	if x != nil {
		return x.ParentInode
	}
	return 0
}

func (x *LookupRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type GetattrRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Inode         uint64                 `protobuf:"varint,1,opt,name=inode,proto3" json:"inode,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetattrRequest) Reset() {
	*x = GetattrRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetattrRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetattrRequest) ProtoMessage() {}

func (x *GetattrRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetattrRequest.ProtoReflect.Descriptor instead.
func (*GetattrRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{3}
}

func (x *GetattrRequest) GetInode() uint64 {
	if x != nil {
		return x.Inode
	}
	return 0
}

type SetattrRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Inode         uint64                 `protobuf:"varint,1,opt,name=inode,proto3" json:"inode,omitempty"`
	Mode          *uint32                `protobuf:"varint,2,opt,name=mode,proto3,oneof" json:"mode,omitempty"`
	Uid           *uint32                `protobuf:"varint,3,opt,name=uid,proto3,oneof" json:"uid,omitempty"`
	Gid           *uint32                `protobuf:"varint,4,opt,name=gid,proto3,oneof" json:"gid,omitempty"`
	Size          *uint64                `protobuf:"varint,5,opt,name=size,proto3,oneof" json:"size,omitempty"` // only truncation to 0 is accepted
	AccessTime    *int64                 `protobuf:"varint,6,opt,name=access_time,json=accessTime,proto3,oneof" json:"access_time,omitempty"`
	ModifyTime    *int64                 `protobuf:"varint,7,opt,name=modify_time,json=modifyTime,proto3,oneof" json:"modify_time,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SetattrRequest) Reset() {
	*x = SetattrRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SetattrRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetattrRequest) ProtoMessage() {}

func (x *SetattrRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetattrRequest.ProtoReflect.Descriptor instead.
func (*SetattrRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{4}
}

func (x *SetattrRequest) GetInode() uint64 {
	if x != nil {
		return x.Inode
	}
	return 0
}

func (x *SetattrRequest) GetMode() uint32 {
	if x != nil && x.Mode != nil {
		return *x.Mode
	}
	return 0
}

func (x *SetattrRequest) GetUid() uint32 {
	if x != nil && x.Uid != nil {
		return *x.Uid
	}
	return 0
}

func (x *SetattrRequest) GetGid() uint32 {
	if x != nil && x.Gid != nil {
		return *x.Gid
	}
	return 0
}

func (x *SetattrRequest) GetSize() uint64 {
	if x != nil && x.Size != nil {
		return *x.Size
	}
	return 0
}

func (x *SetattrRequest) GetAccessTime() int64 {
	if x != nil && x.AccessTime != nil {
		return *x.AccessTime
	}
	return 0
}

func (x *SetattrRequest) GetModifyTime() int64 {
	if x != nil && x.ModifyTime != nil {
		return *x.ModifyTime
	}
	return 0
}

type ReaddirRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Inode         uint64                 `protobuf:"varint,1,opt,name=inode,proto3" json:"inode,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ReaddirRequest) Reset() {
	*x = ReaddirRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ReaddirRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReaddirRequest) ProtoMessage() {}

func (x *ReaddirRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReaddirRequest.ProtoReflect.Descriptor instead.
func (*ReaddirRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{5}
}

func (x *ReaddirRequest) GetInode() uint64 {
	if x != nil {
		return x.Inode
	}
	return 0
}

type DirectoryEntry struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Attributes    *Attributes            `protobuf:"bytes,2,opt,name=attributes,proto3" json:"attributes,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DirectoryEntry) Reset() {
	*x = DirectoryEntry{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DirectoryEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DirectoryEntry) ProtoMessage() {}

func (x *DirectoryEntry) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DirectoryEntry.ProtoReflect.Descriptor instead.
func (*DirectoryEntry) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{6}
}

func (x *DirectoryEntry) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *DirectoryEntry) GetAttributes() *Attributes {
	if x != nil {
		return x.Attributes
	}
	return nil
}

type ReadRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Inode         uint64                 `protobuf:"varint,1,opt,name=inode,proto3" json:"inode,omitempty"`
	Offset        int64                  `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	Size          int64                  `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"` // 0 means to the end
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ReadRequest) Reset() {
	*x = ReadRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ReadRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadRequest) ProtoMessage() {}

func (x *ReadRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReadRequest.ProtoReflect.Descriptor instead.
func (*ReadRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{7}
}

func (x *ReadRequest) GetInode() uint64 {
	if x != nil {
		return x.Inode
	}
	return 0
}

func (x *ReadRequest) GetOffset() int64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *ReadRequest) GetSize() int64 {
	if x != nil {
		return x.Size
	}
	return 0
}

type ReadResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Data          []byte                 `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ReadResponse) Reset() {
	*x = ReadResponse{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ReadResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadResponse) ProtoMessage() {}

func (x *ReadResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReadResponse.ProtoReflect.Descriptor instead.
func (*ReadResponse) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{8}
}

func (x *ReadResponse) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type WriteTarget struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ParentInode   uint64                 `protobuf:"varint,1,opt,name=parent_inode,json=parentInode,proto3" json:"parent_inode,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Mode          uint32                 `protobuf:"varint,3,opt,name=mode,proto3" json:"mode,omitempty"`
	Uid           uint32                 `protobuf:"varint,4,opt,name=uid,proto3" json:"uid,omitempty"`
	Gid           uint32                 `protobuf:"varint,5,opt,name=gid,proto3" json:"gid,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *WriteTarget) Reset() {
	*x = WriteTarget{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *WriteTarget) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WriteTarget) ProtoMessage() {}

func (x *WriteTarget) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WriteTarget.ProtoReflect.Descriptor instead.
func (*WriteTarget) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{9}
}

func (x *WriteTarget) GetParentInode() uint64 {
	if x != nil {
		return x.ParentInode
	}
	return 0
}

func (x *WriteTarget) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *WriteTarget) GetMode() uint32 {
	if x != nil {
		return x.Mode
	}
	return 0
}

func (x *WriteTarget) GetUid() uint32 {
	if x != nil {
		return x.Uid
	}
	return 0
}

func (x *WriteTarget) GetGid() uint32 {
	if x != nil {
		return x.Gid
	}
	return 0
}

type WriteRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to Content:
	//
	//	*WriteRequest_Target
	//	*WriteRequest_Data
	Content       isWriteRequest_Content `protobuf_oneof:"content"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *WriteRequest) Reset() {
	*x = WriteRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *WriteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WriteRequest) ProtoMessage() {}

func (x *WriteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WriteRequest.ProtoReflect.Descriptor instead.
func (*WriteRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{10}
}

func (x *WriteRequest) GetContent() isWriteRequest_Content {
	if x != nil {
		return x.Content
	}
	return nil
}

func (x *WriteRequest) GetTarget() *WriteTarget {
	if x != nil {
		if x, ok := x.Content.(*WriteRequest_Target); ok {
			return x.Target
		}
	}
	return nil
}

func (x *WriteRequest) GetData() []byte {
	if x != nil {
		if x, ok := x.Content.(*WriteRequest_Data); ok {
			return x.Data
		}
	}
	return nil
}

type isWriteRequest_Content interface {
	isWriteRequest_Content()
}

type WriteRequest_Target struct {
	Target *WriteTarget `protobuf:"bytes,1,opt,name=target,proto3,oneof"` // first message
}

type WriteRequest_Data struct {
	Data []byte `protobuf:"bytes,2,opt,name=data,proto3,oneof"` // second and later messages
}

func (*WriteRequest_Target) isWriteRequest_Content() {}

func (*WriteRequest_Data) isWriteRequest_Content() {}

type WriteResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FileDataId    string                 `protobuf:"bytes,1,opt,name=file_data_id,json=fileDataId,proto3" json:"file_data_id,omitempty"`
	Size          uint64                 `protobuf:"varint,2,opt,name=size,proto3" json:"size,omitempty"`
	Attributes    *Attributes            `protobuf:"bytes,3,opt,name=attributes,proto3" json:"attributes,omitempty"` // the written file as recorded in the index
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *WriteResponse) Reset() {
	*x = WriteResponse{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *WriteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WriteResponse) ProtoMessage() {}

func (x *WriteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WriteResponse.ProtoReflect.Descriptor instead.
func (*WriteResponse) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{11}
}

func (x *WriteResponse) GetFileDataId() string {
	if x != nil {
		return x.FileDataId
	}
	return ""
}

func (x *WriteResponse) GetSize() uint64 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *WriteResponse) GetAttributes() *Attributes {
	if x != nil {
		return x.Attributes
	}
	return nil
}

type UnlinkRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ParentInode   uint64                 `protobuf:"varint,1,opt,name=parent_inode,json=parentInode,proto3" json:"parent_inode,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UnlinkRequest) Reset() {
	*x = UnlinkRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UnlinkRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UnlinkRequest) ProtoMessage() {}

func (x *UnlinkRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UnlinkRequest.ProtoReflect.Descriptor instead.
func (*UnlinkRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{12}
}

func (x *UnlinkRequest) GetParentInode() uint64 {
	if x != nil {
		return x.ParentInode
	}
	return 0
}

func (x *UnlinkRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type MkdirRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ParentInode   uint64                 `protobuf:"varint,1,opt,name=parent_inode,json=parentInode,proto3" json:"parent_inode,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Mode          uint32                 `protobuf:"varint,3,opt,name=mode,proto3" json:"mode,omitempty"`
	Uid           uint32                 `protobuf:"varint,4,opt,name=uid,proto3" json:"uid,omitempty"`
	Gid           uint32                 `protobuf:"varint,5,opt,name=gid,proto3" json:"gid,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *MkdirRequest) Reset() {
	*x = MkdirRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MkdirRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MkdirRequest) ProtoMessage() {}

func (x *MkdirRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MkdirRequest.ProtoReflect.Descriptor instead.
func (*MkdirRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{13}
}

func (x *MkdirRequest) GetParentInode() uint64 {
	if x != nil {
		return x.ParentInode
	}
	return 0
}

func (x *MkdirRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *MkdirRequest) GetMode() uint32 {
	if x != nil {
		return x.Mode
	}
	return 0
}

func (x *MkdirRequest) GetUid() uint32 {
	if x != nil {
		return x.Uid
	}
	return 0
}

func (x *MkdirRequest) GetGid() uint32 {
	if x != nil {
		return x.Gid
	}
	return 0
}

type RmdirRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ParentInode   uint64                 `protobuf:"varint,1,opt,name=parent_inode,json=parentInode,proto3" json:"parent_inode,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RmdirRequest) Reset() {
	*x = RmdirRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RmdirRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RmdirRequest) ProtoMessage() {}

func (x *RmdirRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RmdirRequest.ProtoReflect.Descriptor instead.
func (*RmdirRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{14}
}

func (x *RmdirRequest) GetParentInode() uint64 {
	if x != nil {
		return x.ParentInode
	}
	return 0
}

func (x *RmdirRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type RenameRequest struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	OldParentInode uint64                 `protobuf:"varint,1,opt,name=old_parent_inode,json=oldParentInode,proto3" json:"old_parent_inode,omitempty"`
	OldName        string                 `protobuf:"bytes,2,opt,name=old_name,json=oldName,proto3" json:"old_name,omitempty"`
	NewParentInode uint64                 `protobuf:"varint,3,opt,name=new_parent_inode,json=newParentInode,proto3" json:"new_parent_inode,omitempty"`
	NewName        string                 `protobuf:"bytes,4,opt,name=new_name,json=newName,proto3" json:"new_name,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *RenameRequest) Reset() {
	*x = RenameRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[15]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RenameRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RenameRequest) ProtoMessage() {}

func (x *RenameRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[15]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RenameRequest.ProtoReflect.Descriptor instead.
func (*RenameRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{15}
}

func (x *RenameRequest) GetOldParentInode() uint64 {
	if x != nil {
		return x.OldParentInode
	}
	return 0
}

func (x *RenameRequest) GetOldName() string {
	if x != nil {
		return x.OldName
	}
	return ""
}

func (x *RenameRequest) GetNewParentInode() uint64 {
	if x != nil {
		return x.NewParentInode
	}
	return 0
}

func (x *RenameRequest) GetNewName() string {
	if x != nil {
		return x.NewName
	}
	return ""
}

type GetxattrRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Inode         uint64                 `protobuf:"varint,1,opt,name=inode,proto3" json:"inode,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetxattrRequest) Reset() {
	*x = GetxattrRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[16]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetxattrRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetxattrRequest) ProtoMessage() {}

func (x *GetxattrRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[16]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetxattrRequest.ProtoReflect.Descriptor instead.
func (*GetxattrRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{16}
}

func (x *GetxattrRequest) GetInode() uint64 {
	if x != nil {
		return x.Inode
	}
	return 0
}

func (x *GetxattrRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type GetxattrResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         []byte                 `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetxattrResponse) Reset() {
	*x = GetxattrResponse{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[17]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetxattrResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetxattrResponse) ProtoMessage() {}

func (x *GetxattrResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[17]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetxattrResponse.ProtoReflect.Descriptor instead.
func (*GetxattrResponse) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{17}
}

func (x *GetxattrResponse) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

type SetxattrRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Inode         uint64                 `protobuf:"varint,1,opt,name=inode,proto3" json:"inode,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Value         []byte                 `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SetxattrRequest) Reset() {
	*x = SetxattrRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[18]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SetxattrRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetxattrRequest) ProtoMessage() {}

func (x *SetxattrRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[18]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetxattrRequest.ProtoReflect.Descriptor instead.
func (*SetxattrRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{18}
}

func (x *SetxattrRequest) GetInode() uint64 {
	if x != nil {
		return x.Inode
	}
	return 0
}

func (x *SetxattrRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *SetxattrRequest) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

type ListxattrRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Inode         uint64                 `protobuf:"varint,1,opt,name=inode,proto3" json:"inode,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListxattrRequest) Reset() {
	*x = ListxattrRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[19]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListxattrRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListxattrRequest) ProtoMessage() {}

func (x *ListxattrRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[19]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListxattrRequest.ProtoReflect.Descriptor instead.
func (*ListxattrRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{19}
}

func (x *ListxattrRequest) GetInode() uint64 {
	if x != nil {
		return x.Inode
	}
	return 0
}

type ListxattrResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Names         []string               `protobuf:"bytes,1,rep,name=names,proto3" json:"names,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListxattrResponse) Reset() {
	*x = ListxattrResponse{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[20]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListxattrResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListxattrResponse) ProtoMessage() {}

func (x *ListxattrResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[20]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListxattrResponse.ProtoReflect.Descriptor instead.
func (*ListxattrResponse) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{20}
}

func (x *ListxattrResponse) GetNames() []string {
	if x != nil {
		return x.Names
	}
	return nil
}

type RemovexattrRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Inode         uint64                 `protobuf:"varint,1,opt,name=inode,proto3" json:"inode,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RemovexattrRequest) Reset() {
	*x = RemovexattrRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[21]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RemovexattrRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemovexattrRequest) ProtoMessage() {}

func (x *RemovexattrRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[21]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemovexattrRequest.ProtoReflect.Descriptor instead.
func (*RemovexattrRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{21}
}

func (x *RemovexattrRequest) GetInode() uint64 {
	if x != nil {
		return x.Inode
	}
	return 0
}

func (x *RemovexattrRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type StatfsResponse struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	BlockSize       uint64                 `protobuf:"varint,1,opt,name=block_size,json=blockSize,proto3" json:"block_size,omitempty"`
	Blocks          uint64                 `protobuf:"varint,2,opt,name=blocks,proto3" json:"blocks,omitempty"`
	BlocksFree      uint64                 `protobuf:"varint,3,opt,name=blocks_free,json=blocksFree,proto3" json:"blocks_free,omitempty"`
	BlocksAvailable uint64                 `protobuf:"varint,4,opt,name=blocks_available,json=blocksAvailable,proto3" json:"blocks_available,omitempty"`
	Files           uint64                 `protobuf:"varint,5,opt,name=files,proto3" json:"files,omitempty"`
	FilesFree       uint64                 `protobuf:"varint,6,opt,name=files_free,json=filesFree,proto3" json:"files_free,omitempty"`
	NameLength      uint32                 `protobuf:"varint,7,opt,name=name_length,json=nameLength,proto3" json:"name_length,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *StatfsResponse) Reset() {
	*x = StatfsResponse{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[22]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StatfsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatfsResponse) ProtoMessage() {}

func (x *StatfsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[22]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatfsResponse.ProtoReflect.Descriptor instead.
func (*StatfsResponse) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{22}
}

func (x *StatfsResponse) GetBlockSize() uint64 {
	if x != nil {
		return x.BlockSize
	}
	return 0
}

func (x *StatfsResponse) GetBlocks() uint64 {
	if x != nil {
		return x.Blocks
	}
	return 0
}

func (x *StatfsResponse) GetBlocksFree() uint64 {
	if x != nil {
		return x.BlocksFree
	}
	return 0
}

func (x *StatfsResponse) GetBlocksAvailable() uint64 {
	if x != nil {
		return x.BlocksAvailable
	}
	return 0
}

func (x *StatfsResponse) GetFiles() uint64 {
	if x != nil {
		return x.Files
	}
	return 0
}

func (x *StatfsResponse) GetFilesFree() uint64 {
	if x != nil {
		return x.FilesFree
	}
	return 0
}

func (x *StatfsResponse) GetNameLength() uint32 {
	if x != nil {
		return x.NameLength
	}
	return 0
}

type QueueStatus struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Queue         string                 `protobuf:"bytes,1,opt,name=queue,proto3" json:"queue,omitempty"`
	Pending       uint64                 `protobuf:"varint,2,opt,name=pending,proto3" json:"pending,omitempty"`
	Running       uint64                 `protobuf:"varint,3,opt,name=running,proto3" json:"running,omitempty"`
	Failed        uint64                 `protobuf:"varint,4,opt,name=failed,proto3" json:"failed,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *QueueStatus) Reset() {
	*x = QueueStatus{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[23]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *QueueStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueueStatus) ProtoMessage() {}

func (x *QueueStatus) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[23]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueueStatus.ProtoReflect.Descriptor instead.
func (*QueueStatus) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{23}
}

func (x *QueueStatus) GetQueue() string {
	if x != nil {
		return x.Queue
	}
	return ""
}

func (x *QueueStatus) GetPending() uint64 {
	if x != nil {
		return x.Pending
	}
	return 0
}

func (x *QueueStatus) GetRunning() uint64 {
	if x != nil {
		return x.Running
	}
	return 0
}

func (x *QueueStatus) GetFailed() uint64 {
	if x != nil {
		return x.Failed
	}
	return 0
}

type StatusResponse struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	Queues           []*QueueStatus         `protobuf:"bytes,1,rep,name=queues,proto3" json:"queues,omitempty"`
	FileData         uint64                 `protobuf:"varint,2,opt,name=file_data,json=fileData,proto3" json:"file_data,omitempty"`
	LogicallyDeleted uint64                 `protobuf:"varint,4,opt,name=logically_deleted,json=logicallyDeleted,proto3" json:"logically_deleted,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *StatusResponse) Reset() {
	*x = StatusResponse{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[24]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StatusResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusResponse) ProtoMessage() {}

func (x *StatusResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[24]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusResponse.ProtoReflect.Descriptor instead.
func (*StatusResponse) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{24}
}

func (x *StatusResponse) GetQueues() []*QueueStatus {
	if x != nil {
		return x.Queues
	}
	return nil
}

func (x *StatusResponse) GetFileData() uint64 {
	if x != nil {
		return x.FileData
	}
	return 0
}

func (x *StatusResponse) GetLogicallyDeleted() uint64 {
	if x != nil {
		return x.LogicallyDeleted
	}
	return 0
}

type ListTasksRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Queue         string                 `protobuf:"bytes,1,opt,name=queue,proto3" json:"queue,omitempty"` // empty means all queues
	ActiveOnly    bool                   `protobuf:"varint,2,opt,name=active_only,json=activeOnly,proto3" json:"active_only,omitempty"`
	Limit         uint32                 `protobuf:"varint,3,opt,name=limit,proto3" json:"limit,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListTasksRequest) Reset() {
	*x = ListTasksRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[25]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListTasksRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListTasksRequest) ProtoMessage() {}

func (x *ListTasksRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[25]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListTasksRequest.ProtoReflect.Descriptor instead.
func (*ListTasksRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{25}
}

func (x *ListTasksRequest) GetQueue() string {
	if x != nil {
		return x.Queue
	}
	return ""
}

func (x *ListTasksRequest) GetActiveOnly() bool {
	if x != nil {
		return x.ActiveOnly
	}
	return false
}

func (x *ListTasksRequest) GetLimit() uint32 {
	if x != nil {
		return x.Limit
	}
	return 0
}

type Task struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Queue         string                 `protobuf:"bytes,1,opt,name=queue,proto3" json:"queue,omitempty"`
	Id            uint64                 `protobuf:"varint,2,opt,name=id,proto3" json:"id,omitempty"`
	Kind          string                 `protobuf:"bytes,3,opt,name=kind,proto3" json:"kind,omitempty"`
	State         string                 `protobuf:"bytes,4,opt,name=state,proto3" json:"state,omitempty"`
	Attempts      uint32                 `protobuf:"varint,5,opt,name=attempts,proto3" json:"attempts,omitempty"`
	NextRunAt     int64                  `protobuf:"varint,6,opt,name=next_run_at,json=nextRunAt,proto3" json:"next_run_at,omitempty"`
	LastError     string                 `protobuf:"bytes,7,opt,name=last_error,json=lastError,proto3" json:"last_error,omitempty"`
	CreatedAt     int64                  `protobuf:"varint,8,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	FinishedAt    int64                  `protobuf:"varint,9,opt,name=finished_at,json=finishedAt,proto3" json:"finished_at,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Task) Reset() {
	*x = Task{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[26]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Task) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Task) ProtoMessage() {}

func (x *Task) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[26]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Task.ProtoReflect.Descriptor instead.
func (*Task) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{26}
}

func (x *Task) GetQueue() string {
	if x != nil {
		return x.Queue
	}
	return ""
}

func (x *Task) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Task) GetKind() string {
	if x != nil {
		return x.Kind
	}
	return ""
}

func (x *Task) GetState() string {
	if x != nil {
		return x.State
	}
	return ""
}

func (x *Task) GetAttempts() uint32 {
	if x != nil {
		return x.Attempts
	}
	return 0
}

func (x *Task) GetNextRunAt() int64 {
	if x != nil {
		return x.NextRunAt
	}
	return 0
}

func (x *Task) GetLastError() string {
	if x != nil {
		return x.LastError
	}
	return ""
}

func (x *Task) GetCreatedAt() int64 {
	if x != nil {
		return x.CreatedAt
	}
	return 0
}

func (x *Task) GetFinishedAt() int64 {
	if x != nil {
		return x.FinishedAt
	}
	return 0
}

type ListTasksResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Tasks         []*Task                `protobuf:"bytes,1,rep,name=tasks,proto3" json:"tasks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListTasksResponse) Reset() {
	*x = ListTasksResponse{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[27]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListTasksResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListTasksResponse) ProtoMessage() {}

func (x *ListTasksResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[27]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListTasksResponse.ProtoReflect.Descriptor instead.
func (*ListTasksResponse) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{27}
}

func (x *ListTasksResponse) GetTasks() []*Task {
	if x != nil {
		return x.Tasks
	}
	return nil
}

type QueryValue struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to Value:
	//
	//	*QueryValue_Integer
	//	*QueryValue_Real
	//	*QueryValue_Text
	Value         isQueryValue_Value `protobuf_oneof:"value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *QueryValue) Reset() {
	*x = QueryValue{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[28]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *QueryValue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryValue) ProtoMessage() {}

func (x *QueryValue) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[28]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryValue.ProtoReflect.Descriptor instead.
func (*QueryValue) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{28}
}

func (x *QueryValue) GetValue() isQueryValue_Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *QueryValue) GetInteger() int64 {
	if x != nil {
		if x, ok := x.Value.(*QueryValue_Integer); ok {
			return x.Integer
		}
	}
	return 0
}

func (x *QueryValue) GetReal() float64 {
	if x != nil {
		if x, ok := x.Value.(*QueryValue_Real); ok {
			return x.Real
		}
	}
	return 0
}

func (x *QueryValue) GetText() string {
	if x != nil {
		if x, ok := x.Value.(*QueryValue_Text); ok {
			return x.Text
		}
	}
	return ""
}

type isQueryValue_Value interface {
	isQueryValue_Value()
}

type QueryValue_Integer struct {
	Integer int64 `protobuf:"varint,1,opt,name=integer,proto3,oneof"`
}

type QueryValue_Real struct {
	Real float64 `protobuf:"fixed64,2,opt,name=real,proto3,oneof"`
}

type QueryValue_Text struct {
	Text string `protobuf:"bytes,3,opt,name=text,proto3,oneof"`
}

func (*QueryValue_Integer) isQueryValue_Value() {}

func (*QueryValue_Real) isQueryValue_Value() {}

func (*QueryValue_Text) isQueryValue_Value() {}

type NamedQueryValue struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"` // with its prefix, such as ":name"
	Value         *QueryValue            `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NamedQueryValue) Reset() {
	*x = NamedQueryValue{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[29]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NamedQueryValue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NamedQueryValue) ProtoMessage() {}

func (x *NamedQueryValue) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[29]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NamedQueryValue.ProtoReflect.Descriptor instead.
func (*NamedQueryValue) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{29}
}

func (x *NamedQueryValue) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *NamedQueryValue) GetValue() *QueryValue {
	if x != nil {
		return x.Value
	}
	return nil
}

type QueryRequest struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	Sql             string                 `protobuf:"bytes,1,opt,name=sql,proto3" json:"sql,omitempty"`
	Parameters      []*QueryValue          `protobuf:"bytes,2,rep,name=parameters,proto3" json:"parameters,omitempty"`
	NamedParameters []*NamedQueryValue     `protobuf:"bytes,3,rep,name=named_parameters,json=namedParameters,proto3" json:"named_parameters,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *QueryRequest) Reset() {
	*x = QueryRequest{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[30]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *QueryRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryRequest) ProtoMessage() {}

func (x *QueryRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[30]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryRequest.ProtoReflect.Descriptor instead.
func (*QueryRequest) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{30}
}

func (x *QueryRequest) GetSql() string {
	if x != nil {
		return x.Sql
	}
	return ""
}

func (x *QueryRequest) GetParameters() []*QueryValue {
	if x != nil {
		return x.Parameters
	}
	return nil
}

func (x *QueryRequest) GetNamedParameters() []*NamedQueryValue {
	if x != nil {
		return x.NamedParameters
	}
	return nil
}

type QueryRow struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        []*QueryValue          `protobuf:"bytes,1,rep,name=values,proto3" json:"values,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *QueryRow) Reset() {
	*x = QueryRow{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[31]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *QueryRow) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryRow) ProtoMessage() {}

func (x *QueryRow) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[31]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryRow.ProtoReflect.Descriptor instead.
func (*QueryRow) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{31}
}

func (x *QueryRow) GetValues() []*QueryValue {
	if x != nil {
		return x.Values
	}
	return nil
}

type QueryResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Columns       []string               `protobuf:"bytes,1,rep,name=columns,proto3" json:"columns,omitempty"`
	Rows          []*QueryRow            `protobuf:"bytes,2,rep,name=rows,proto3" json:"rows,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *QueryResponse) Reset() {
	*x = QueryResponse{}
	mi := &file_protocol_yaagwadb_proto_msgTypes[32]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *QueryResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryResponse) ProtoMessage() {}

func (x *QueryResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_yaagwadb_proto_msgTypes[32]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryResponse.ProtoReflect.Descriptor instead.
func (*QueryResponse) Descriptor() ([]byte, []int) {
	return file_protocol_yaagwadb_proto_rawDescGZIP(), []int{32}
}

func (x *QueryResponse) GetColumns() []string {
	if x != nil {
		return x.Columns
	}
	return nil
}

func (x *QueryResponse) GetRows() []*QueryRow {
	if x != nil {
		return x.Rows
	}
	return nil
}

var File_protocol_yaagwadb_proto protoreflect.FileDescriptor

const file_protocol_yaagwadb_proto_rawDesc = "" +
	"\n" +
	"\x17protocol/yaagwadb.proto\x12\byaagwadb\"\a\n" +
	"\x05Empty\"\xe9\x02\n" +
	"\n" +
	"Attributes\x12\x14\n" +
	"\x05inode\x18\x01 \x01(\x04R\x05inode\x12!\n" +
	"\fparent_inode\x18\x02 \x01(\x04R\vparentInode\x12\x12\n" +
	"\x04name\x18\x03 \x01(\tR\x04name\x12\x12\n" +
	"\x04mode\x18\x04 \x01(\rR\x04mode\x12\x10\n" +
	"\x03uid\x18\x05 \x01(\rR\x03uid\x12\x10\n" +
	"\x03gid\x18\x06 \x01(\rR\x03gid\x12\x12\n" +
	"\x04size\x18\a \x01(\x04R\x04size\x12\x1f\n" +
	"\vaccess_time\x18\b \x01(\x03R\n" +
	"accessTime\x12\x1f\n" +
	"\vmodify_time\x18\t \x01(\x03R\n" +
	"modifyTime\x12\x1f\n" +
	"\vchange_time\x18\n" +
	" \x01(\x03R\n" +
	"changeTime\x12\x14\n" +
	"\x05nlink\x18\v \x01(\rR\x05nlink\x12\x12\n" +
	"\x04rdev\x18\f \x01(\rR\x04rdev\x12\x16\n" +
	"\x06blocks\x18\r \x01(\x04R\x06blocks\x12\x1d\n" +
	"\n" +
	"block_size\x18\x0e \x01(\rR\tblockSize\"F\n" +
	"\rLookupRequest\x12!\n" +
	"\fparent_inode\x18\x01 \x01(\x04R\vparentInode\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\"&\n" +
	"\x0eGetattrRequest\x12\x14\n" +
	"\x05inode\x18\x01 \x01(\x04R\x05inode\"\x94\x02\n" +
	"\x0eSetattrRequest\x12\x14\n" +
	"\x05inode\x18\x01 \x01(\x04R\x05inode\x12\x17\n" +
	"\x04mode\x18\x02 \x01(\rH\x00R\x04mode\x88\x01\x01\x12\x15\n" +
	"\x03uid\x18\x03 \x01(\rH\x01R\x03uid\x88\x01\x01\x12\x15\n" +
	"\x03gid\x18\x04 \x01(\rH\x02R\x03gid\x88\x01\x01\x12\x17\n" +
	"\x04size\x18\x05 \x01(\x04H\x03R\x04size\x88\x01\x01\x12$\n" +
	"\vaccess_time\x18\x06 \x01(\x03H\x04R\n" +
	"accessTime\x88\x01\x01\x12$\n" +
	"\vmodify_time\x18\a \x01(\x03H\x05R\n" +
	"modifyTime\x88\x01\x01B\a\n" +
	"\x05_modeB\x06\n" +
	"\x04_uidB\x06\n" +
	"\x04_gidB\a\n" +
	"\x05_sizeB\x0e\n" +
	"\f_access_timeB\x0e\n" +
	"\f_modify_time\"&\n" +
	"\x0eReaddirRequest\x12\x14\n" +
	"\x05inode\x18\x01 \x01(\x04R\x05inode\"Z\n" +
	"\x0eDirectoryEntry\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x124\n" +
	"\n" +
	"attributes\x18\x02 \x01(\v2\x14.yaagwadb.AttributesR\n" +
	"attributes\"O\n" +
	"\vReadRequest\x12\x14\n" +
	"\x05inode\x18\x01 \x01(\x04R\x05inode\x12\x16\n" +
	"\x06offset\x18\x02 \x01(\x03R\x06offset\x12\x12\n" +
	"\x04size\x18\x03 \x01(\x03R\x04size\"\"\n" +
	"\fReadResponse\x12\x12\n" +
	"\x04data\x18\x01 \x01(\fR\x04data\"|\n" +
	"\vWriteTarget\x12!\n" +
	"\fparent_inode\x18\x01 \x01(\x04R\vparentInode\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\x12\x12\n" +
	"\x04mode\x18\x03 \x01(\rR\x04mode\x12\x10\n" +
	"\x03uid\x18\x04 \x01(\rR\x03uid\x12\x10\n" +
	"\x03gid\x18\x05 \x01(\rR\x03gid\"`\n" +
	"\fWriteRequest\x12/\n" +
	"\x06target\x18\x01 \x01(\v2\x15.yaagwadb.WriteTargetH\x00R\x06target\x12\x14\n" +
	"\x04data\x18\x02 \x01(\fH\x00R\x04dataB\t\n" +
	"\acontent\"{\n" +
	"\rWriteResponse\x12 \n" +
	"\ffile_data_id\x18\x01 \x01(\tR\n" +
	"fileDataId\x12\x12\n" +
	"\x04size\x18\x02 \x01(\x04R\x04size\x124\n" +
	"\n" +
	"attributes\x18\x03 \x01(\v2\x14.yaagwadb.AttributesR\n" +
	"attributes\"F\n" +
	"\rUnlinkRequest\x12!\n" +
	"\fparent_inode\x18\x01 \x01(\x04R\vparentInode\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\"}\n" +
	"\fMkdirRequest\x12!\n" +
	"\fparent_inode\x18\x01 \x01(\x04R\vparentInode\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\x12\x12\n" +
	"\x04mode\x18\x03 \x01(\rR\x04mode\x12\x10\n" +
	"\x03uid\x18\x04 \x01(\rR\x03uid\x12\x10\n" +
	"\x03gid\x18\x05 \x01(\rR\x03gid\"E\n" +
	"\fRmdirRequest\x12!\n" +
	"\fparent_inode\x18\x01 \x01(\x04R\vparentInode\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\"\x99\x01\n" +
	"\rRenameRequest\x12(\n" +
	"\x10old_parent_inode\x18\x01 \x01(\x04R\x0eoldParentInode\x12\x19\n" +
	"\bold_name\x18\x02 \x01(\tR\aoldName\x12(\n" +
	"\x10new_parent_inode\x18\x03 \x01(\x04R\x0enewParentInode\x12\x19\n" +
	"\bnew_name\x18\x04 \x01(\tR\anewName\";\n" +
	"\x0fGetxattrRequest\x12\x14\n" +
	"\x05inode\x18\x01 \x01(\x04R\x05inode\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\"(\n" +
	"\x10GetxattrResponse\x12\x14\n" +
	"\x05value\x18\x01 \x01(\fR\x05value\"Q\n" +
	"\x0fSetxattrRequest\x12\x14\n" +
	"\x05inode\x18\x01 \x01(\x04R\x05inode\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\x12\x14\n" +
	"\x05value\x18\x03 \x01(\fR\x05value\"(\n" +
	"\x10ListxattrRequest\x12\x14\n" +
	"\x05inode\x18\x01 \x01(\x04R\x05inode\")\n" +
	"\x11ListxattrResponse\x12\x14\n" +
	"\x05names\x18\x01 \x03(\tR\x05names\">\n" +
	"\x12RemovexattrRequest\x12\x14\n" +
	"\x05inode\x18\x01 \x01(\x04R\x05inode\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\"\xe9\x01\n" +
	"\x0eStatfsResponse\x12\x1d\n" +
	"\n" +
	"block_size\x18\x01 \x01(\x04R\tblockSize\x12\x16\n" +
	"\x06blocks\x18\x02 \x01(\x04R\x06blocks\x12\x1f\n" +
	"\vblocks_free\x18\x03 \x01(\x04R\n" +
	"blocksFree\x12)\n" +
	"\x10blocks_available\x18\x04 \x01(\x04R\x0fblocksAvailable\x12\x14\n" +
	"\x05files\x18\x05 \x01(\x04R\x05files\x12\x1d\n" +
	"\n" +
	"files_free\x18\x06 \x01(\x04R\tfilesFree\x12\x1f\n" +
	"\vname_length\x18\a \x01(\rR\n" +
	"nameLength\"o\n" +
	"\vQueueStatus\x12\x14\n" +
	"\x05queue\x18\x01 \x01(\tR\x05queue\x12\x18\n" +
	"\apending\x18\x02 \x01(\x04R\apending\x12\x18\n" +
	"\arunning\x18\x03 \x01(\x04R\arunning\x12\x16\n" +
	"\x06failed\x18\x04 \x01(\x04R\x06failed\"\x8f\x01\n" +
	"\x0eStatusResponse\x12-\n" +
	"\x06queues\x18\x01 \x03(\v2\x15.yaagwadb.QueueStatusR\x06queues\x12\x1b\n" +
	"\tfile_data\x18\x02 \x01(\x04R\bfileData\x12+\n" +
	"\x11logically_deleted\x18\x04 \x01(\x04R\x10logicallyDeletedJ\x04\b\x03\x10\x04\"_\n" +
	"\x10ListTasksRequest\x12\x14\n" +
	"\x05queue\x18\x01 \x01(\tR\x05queue\x12\x1f\n" +
	"\vactive_only\x18\x02 \x01(\bR\n" +
	"activeOnly\x12\x14\n" +
	"\x05limit\x18\x03 \x01(\rR\x05limit\"\xf1\x01\n" +
	"\x04Task\x12\x14\n" +
	"\x05queue\x18\x01 \x01(\tR\x05queue\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\x04R\x02id\x12\x12\n" +
	"\x04kind\x18\x03 \x01(\tR\x04kind\x12\x14\n" +
	"\x05state\x18\x04 \x01(\tR\x05state\x12\x1a\n" +
	"\battempts\x18\x05 \x01(\rR\battempts\x12\x1e\n" +
	"\vnext_run_at\x18\x06 \x01(\x03R\tnextRunAt\x12\x1d\n" +
	"\n" +
	"last_error\x18\a \x01(\tR\tlastError\x12\x1d\n" +
	"\n" +
	"created_at\x18\b \x01(\x03R\tcreatedAt\x12\x1f\n" +
	"\vfinished_at\x18\t \x01(\x03R\n" +
	"finishedAt\"9\n" +
	"\x11ListTasksResponse\x12$\n" +
	"\x05tasks\x18\x01 \x03(\v2\x0e.yaagwadb.TaskR\x05tasks\"]\n" +
	"\n" +
	"QueryValue\x12\x1a\n" +
	"\ainteger\x18\x01 \x01(\x03H\x00R\ainteger\x12\x14\n" +
	"\x04real\x18\x02 \x01(\x01H\x00R\x04real\x12\x14\n" +
	"\x04text\x18\x03 \x01(\tH\x00R\x04textB\a\n" +
	"\x05value\"Q\n" +
	"\x0fNamedQueryValue\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12*\n" +
	"\x05value\x18\x02 \x01(\v2\x14.yaagwadb.QueryValueR\x05value\"\x9c\x01\n" +
	"\fQueryRequest\x12\x10\n" +
	"\x03sql\x18\x01 \x01(\tR\x03sql\x124\n" +
	"\n" +
	"parameters\x18\x02 \x03(\v2\x14.yaagwadb.QueryValueR\n" +
	"parameters\x12D\n" +
	"\x10named_parameters\x18\x03 \x03(\v2\x19.yaagwadb.NamedQueryValueR\x0fnamedParameters\"8\n" +
	"\bQueryRow\x12,\n" +
	"\x06values\x18\x01 \x03(\v2\x14.yaagwadb.QueryValueR\x06values\"Q\n" +
	"\rQueryResponse\x12\x18\n" +
	"\acolumns\x18\x01 \x03(\tR\acolumns\x12&\n" +
	"\x04rows\x18\x02 \x03(\v2\x12.yaagwadb.QueryRowR\x04rows2\xa9\b\n" +
	"\bYaagwaDB\x127\n" +
	"\x06Lookup\x12\x17.yaagwadb.LookupRequest\x1a\x14.yaagwadb.Attributes\x129\n" +
	"\aGetattr\x12\x18.yaagwadb.GetattrRequest\x1a\x14.yaagwadb.Attributes\x129\n" +
	"\aSetattr\x12\x18.yaagwadb.SetattrRequest\x1a\x14.yaagwadb.Attributes\x12?\n" +
	"\aReaddir\x12\x18.yaagwadb.ReaddirRequest\x1a\x18.yaagwadb.DirectoryEntry0\x01\x127\n" +
	"\x04Read\x12\x15.yaagwadb.ReadRequest\x1a\x16.yaagwadb.ReadResponse0\x01\x12:\n" +
	"\x05Write\x12\x16.yaagwadb.WriteRequest\x1a\x17.yaagwadb.WriteResponse(\x01\x122\n" +
	"\x06Unlink\x12\x17.yaagwadb.UnlinkRequest\x1a\x0f.yaagwadb.Empty\x125\n" +
	"\x05Mkdir\x12\x16.yaagwadb.MkdirRequest\x1a\x14.yaagwadb.Attributes\x120\n" +
	"\x05Rmdir\x12\x16.yaagwadb.RmdirRequest\x1a\x0f.yaagwadb.Empty\x122\n" +
	"\x06Rename\x12\x17.yaagwadb.RenameRequest\x1a\x0f.yaagwadb.Empty\x12A\n" +
	"\bGetxattr\x12\x19.yaagwadb.GetxattrRequest\x1a\x1a.yaagwadb.GetxattrResponse\x126\n" +
	"\bSetxattr\x12\x19.yaagwadb.SetxattrRequest\x1a\x0f.yaagwadb.Empty\x12D\n" +
	"\tListxattr\x12\x1a.yaagwadb.ListxattrRequest\x1a\x1b.yaagwadb.ListxattrResponse\x12<\n" +
	"\vRemovexattr\x12\x1c.yaagwadb.RemovexattrRequest\x1a\x0f.yaagwadb.Empty\x123\n" +
	"\x06Statfs\x12\x0f.yaagwadb.Empty\x1a\x18.yaagwadb.StatfsResponse\x128\n" +
	"\x05Query\x12\x16.yaagwadb.QueryRequest\x1a\x17.yaagwadb.QueryResponse\x123\n" +
	"\x06Status\x12\x0f.yaagwadb.Empty\x1a\x18.yaagwadb.StatusResponse\x12D\n" +
	"\tListTasks\x12\x1a.yaagwadb.ListTasksRequest\x1a\x1b.yaagwadb.ListTasksResponseB\x13Z\x11yaagwadb/protocolb\x06proto3"

var (
	file_protocol_yaagwadb_proto_rawDescOnce sync.Once
	file_protocol_yaagwadb_proto_rawDescData []byte
)

func file_protocol_yaagwadb_proto_rawDescGZIP() []byte {
	file_protocol_yaagwadb_proto_rawDescOnce.Do(func() {
		file_protocol_yaagwadb_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_protocol_yaagwadb_proto_rawDesc), len(file_protocol_yaagwadb_proto_rawDesc)))
	})
	return file_protocol_yaagwadb_proto_rawDescData
}

var file_protocol_yaagwadb_proto_msgTypes = make([]protoimpl.MessageInfo, 33)
var file_protocol_yaagwadb_proto_goTypes = []any{
	(*Empty)(nil),              // 0: yaagwadb.Empty
	(*Attributes)(nil),         // 1: yaagwadb.Attributes
	(*LookupRequest)(nil),      // 2: yaagwadb.LookupRequest
	(*GetattrRequest)(nil),     // 3: yaagwadb.GetattrRequest
	(*SetattrRequest)(nil),     // 4: yaagwadb.SetattrRequest
	(*ReaddirRequest)(nil),     // 5: yaagwadb.ReaddirRequest
	(*DirectoryEntry)(nil),     // 6: yaagwadb.DirectoryEntry
	(*ReadRequest)(nil),        // 7: yaagwadb.ReadRequest
	(*ReadResponse)(nil),       // 8: yaagwadb.ReadResponse
	(*WriteTarget)(nil),        // 9: yaagwadb.WriteTarget
	(*WriteRequest)(nil),       // 10: yaagwadb.WriteRequest
	(*WriteResponse)(nil),      // 11: yaagwadb.WriteResponse
	(*UnlinkRequest)(nil),      // 12: yaagwadb.UnlinkRequest
	(*MkdirRequest)(nil),       // 13: yaagwadb.MkdirRequest
	(*RmdirRequest)(nil),       // 14: yaagwadb.RmdirRequest
	(*RenameRequest)(nil),      // 15: yaagwadb.RenameRequest
	(*GetxattrRequest)(nil),    // 16: yaagwadb.GetxattrRequest
	(*GetxattrResponse)(nil),   // 17: yaagwadb.GetxattrResponse
	(*SetxattrRequest)(nil),    // 18: yaagwadb.SetxattrRequest
	(*ListxattrRequest)(nil),   // 19: yaagwadb.ListxattrRequest
	(*ListxattrResponse)(nil),  // 20: yaagwadb.ListxattrResponse
	(*RemovexattrRequest)(nil), // 21: yaagwadb.RemovexattrRequest
	(*StatfsResponse)(nil),     // 22: yaagwadb.StatfsResponse
	(*QueueStatus)(nil),        // 23: yaagwadb.QueueStatus
	(*StatusResponse)(nil),     // 24: yaagwadb.StatusResponse
	(*ListTasksRequest)(nil),   // 25: yaagwadb.ListTasksRequest
	(*Task)(nil),               // 26: yaagwadb.Task
	(*ListTasksResponse)(nil),  // 27: yaagwadb.ListTasksResponse
	(*QueryValue)(nil),         // 28: yaagwadb.QueryValue
	(*NamedQueryValue)(nil),    // 29: yaagwadb.NamedQueryValue
	(*QueryRequest)(nil),       // 30: yaagwadb.QueryRequest
	(*QueryRow)(nil),           // 31: yaagwadb.QueryRow
	(*QueryResponse)(nil),      // 32: yaagwadb.QueryResponse
}
var file_protocol_yaagwadb_proto_depIdxs = []int32{
	1,  // 0: yaagwadb.DirectoryEntry.attributes:type_name -> yaagwadb.Attributes
	9,  // 1: yaagwadb.WriteRequest.target:type_name -> yaagwadb.WriteTarget
	1,  // 2: yaagwadb.WriteResponse.attributes:type_name -> yaagwadb.Attributes
	23, // 3: yaagwadb.StatusResponse.queues:type_name -> yaagwadb.QueueStatus
	26, // 4: yaagwadb.ListTasksResponse.tasks:type_name -> yaagwadb.Task
	28, // 5: yaagwadb.NamedQueryValue.value:type_name -> yaagwadb.QueryValue
	28, // 6: yaagwadb.QueryRequest.parameters:type_name -> yaagwadb.QueryValue
	29, // 7: yaagwadb.QueryRequest.named_parameters:type_name -> yaagwadb.NamedQueryValue
	28, // 8: yaagwadb.QueryRow.values:type_name -> yaagwadb.QueryValue
	31, // 9: yaagwadb.QueryResponse.rows:type_name -> yaagwadb.QueryRow
	2,  // 10: yaagwadb.YaagwaDB.Lookup:input_type -> yaagwadb.LookupRequest
	3,  // 11: yaagwadb.YaagwaDB.Getattr:input_type -> yaagwadb.GetattrRequest
	4,  // 12: yaagwadb.YaagwaDB.Setattr:input_type -> yaagwadb.SetattrRequest
	5,  // 13: yaagwadb.YaagwaDB.Readdir:input_type -> yaagwadb.ReaddirRequest
	7,  // 14: yaagwadb.YaagwaDB.Read:input_type -> yaagwadb.ReadRequest
	10, // 15: yaagwadb.YaagwaDB.Write:input_type -> yaagwadb.WriteRequest
	12, // 16: yaagwadb.YaagwaDB.Unlink:input_type -> yaagwadb.UnlinkRequest
	13, // 17: yaagwadb.YaagwaDB.Mkdir:input_type -> yaagwadb.MkdirRequest
	14, // 18: yaagwadb.YaagwaDB.Rmdir:input_type -> yaagwadb.RmdirRequest
	15, // 19: yaagwadb.YaagwaDB.Rename:input_type -> yaagwadb.RenameRequest
	16, // 20: yaagwadb.YaagwaDB.Getxattr:input_type -> yaagwadb.GetxattrRequest
	18, // 21: yaagwadb.YaagwaDB.Setxattr:input_type -> yaagwadb.SetxattrRequest
	19, // 22: yaagwadb.YaagwaDB.Listxattr:input_type -> yaagwadb.ListxattrRequest
	21, // 23: yaagwadb.YaagwaDB.Removexattr:input_type -> yaagwadb.RemovexattrRequest
	0,  // 24: yaagwadb.YaagwaDB.Statfs:input_type -> yaagwadb.Empty
	30, // 25: yaagwadb.YaagwaDB.Query:input_type -> yaagwadb.QueryRequest
	0,  // 26: yaagwadb.YaagwaDB.Status:input_type -> yaagwadb.Empty
	25, // 27: yaagwadb.YaagwaDB.ListTasks:input_type -> yaagwadb.ListTasksRequest
	1,  // 28: yaagwadb.YaagwaDB.Lookup:output_type -> yaagwadb.Attributes
	1,  // 29: yaagwadb.YaagwaDB.Getattr:output_type -> yaagwadb.Attributes
	1,  // 30: yaagwadb.YaagwaDB.Setattr:output_type -> yaagwadb.Attributes
	6,  // 31: yaagwadb.YaagwaDB.Readdir:output_type -> yaagwadb.DirectoryEntry
	8,  // 32: yaagwadb.YaagwaDB.Read:output_type -> yaagwadb.ReadResponse
	11, // 33: yaagwadb.YaagwaDB.Write:output_type -> yaagwadb.WriteResponse
	0,  // 34: yaagwadb.YaagwaDB.Unlink:output_type -> yaagwadb.Empty
	1,  // 35: yaagwadb.YaagwaDB.Mkdir:output_type -> yaagwadb.Attributes
	0,  // 36: yaagwadb.YaagwaDB.Rmdir:output_type -> yaagwadb.Empty
	0,  // 37: yaagwadb.YaagwaDB.Rename:output_type -> yaagwadb.Empty
	17, // 38: yaagwadb.YaagwaDB.Getxattr:output_type -> yaagwadb.GetxattrResponse
	0,  // 39: yaagwadb.YaagwaDB.Setxattr:output_type -> yaagwadb.Empty
	20, // 40: yaagwadb.YaagwaDB.Listxattr:output_type -> yaagwadb.ListxattrResponse
	0,  // 41: yaagwadb.YaagwaDB.Removexattr:output_type -> yaagwadb.Empty
	22, // 42: yaagwadb.YaagwaDB.Statfs:output_type -> yaagwadb.StatfsResponse
	32, // 43: yaagwadb.YaagwaDB.Query:output_type -> yaagwadb.QueryResponse
	24, // 44: yaagwadb.YaagwaDB.Status:output_type -> yaagwadb.StatusResponse
	27, // 45: yaagwadb.YaagwaDB.ListTasks:output_type -> yaagwadb.ListTasksResponse
	28, // [28:46] is the sub-list for method output_type
	10, // [10:28] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_protocol_yaagwadb_proto_init() }
func file_protocol_yaagwadb_proto_init() {
	if File_protocol_yaagwadb_proto != nil {
		return
	}
	file_protocol_yaagwadb_proto_msgTypes[4].OneofWrappers = []any{}
	file_protocol_yaagwadb_proto_msgTypes[10].OneofWrappers = []any{
		(*WriteRequest_Target)(nil),
		(*WriteRequest_Data)(nil),
	}
	file_protocol_yaagwadb_proto_msgTypes[28].OneofWrappers = []any{
		(*QueryValue_Integer)(nil),
		(*QueryValue_Real)(nil),
		(*QueryValue_Text)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_protocol_yaagwadb_proto_rawDesc), len(file_protocol_yaagwadb_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   33,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_protocol_yaagwadb_proto_goTypes,
		DependencyIndexes: file_protocol_yaagwadb_proto_depIdxs,
		MessageInfos:      file_protocol_yaagwadb_proto_msgTypes,
	}.Build()
	File_protocol_yaagwadb_proto = out.File
	file_protocol_yaagwadb_proto_goTypes = nil
	file_protocol_yaagwadb_proto_depIdxs = nil
}
