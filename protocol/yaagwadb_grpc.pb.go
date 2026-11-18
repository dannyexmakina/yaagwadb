package protocol

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	YaagwaDB_Lookup_FullMethodName      = "/yaagwadb.YaagwaDB/Lookup"
	YaagwaDB_Getattr_FullMethodName     = "/yaagwadb.YaagwaDB/Getattr"
	YaagwaDB_Setattr_FullMethodName     = "/yaagwadb.YaagwaDB/Setattr"
	YaagwaDB_Readdir_FullMethodName     = "/yaagwadb.YaagwaDB/Readdir"
	YaagwaDB_Read_FullMethodName        = "/yaagwadb.YaagwaDB/Read"
	YaagwaDB_Write_FullMethodName       = "/yaagwadb.YaagwaDB/Write"
	YaagwaDB_Unlink_FullMethodName      = "/yaagwadb.YaagwaDB/Unlink"
	YaagwaDB_Mkdir_FullMethodName       = "/yaagwadb.YaagwaDB/Mkdir"
	YaagwaDB_Rmdir_FullMethodName       = "/yaagwadb.YaagwaDB/Rmdir"
	YaagwaDB_Rename_FullMethodName      = "/yaagwadb.YaagwaDB/Rename"
	YaagwaDB_Getxattr_FullMethodName    = "/yaagwadb.YaagwaDB/Getxattr"
	YaagwaDB_Setxattr_FullMethodName    = "/yaagwadb.YaagwaDB/Setxattr"
	YaagwaDB_Listxattr_FullMethodName   = "/yaagwadb.YaagwaDB/Listxattr"
	YaagwaDB_Removexattr_FullMethodName = "/yaagwadb.YaagwaDB/Removexattr"
	YaagwaDB_Statfs_FullMethodName      = "/yaagwadb.YaagwaDB/Statfs"
	YaagwaDB_Query_FullMethodName       = "/yaagwadb.YaagwaDB/Query"
	YaagwaDB_Status_FullMethodName      = "/yaagwadb.YaagwaDB/Status"
	YaagwaDB_ListTasks_FullMethodName   = "/yaagwadb.YaagwaDB/ListTasks"
)

// YaagwaDBClient is the client API for YaagwaDB service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type YaagwaDBClient interface {
	Lookup(ctx context.Context, in *LookupRequest, opts ...grpc.CallOption) (*Attributes, error)
	Getattr(ctx context.Context, in *GetattrRequest, opts ...grpc.CallOption) (*Attributes, error)
	Setattr(ctx context.Context, in *SetattrRequest, opts ...grpc.CallOption) (*Attributes, error)
	Readdir(ctx context.Context, in *ReaddirRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[DirectoryEntry], error)
	Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[ReadResponse], error)
	Write(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[WriteRequest, WriteResponse], error)
	Unlink(ctx context.Context, in *UnlinkRequest, opts ...grpc.CallOption) (*Empty, error)
	Mkdir(ctx context.Context, in *MkdirRequest, opts ...grpc.CallOption) (*Attributes, error)
	Rmdir(ctx context.Context, in *RmdirRequest, opts ...grpc.CallOption) (*Empty, error)
	Rename(ctx context.Context, in *RenameRequest, opts ...grpc.CallOption) (*Empty, error)
	Getxattr(ctx context.Context, in *GetxattrRequest, opts ...grpc.CallOption) (*GetxattrResponse, error)
	Setxattr(ctx context.Context, in *SetxattrRequest, opts ...grpc.CallOption) (*Empty, error)
	Listxattr(ctx context.Context, in *ListxattrRequest, opts ...grpc.CallOption) (*ListxattrResponse, error)
	Removexattr(ctx context.Context, in *RemovexattrRequest, opts ...grpc.CallOption) (*Empty, error)
	Statfs(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatfsResponse, error)
	Query(ctx context.Context, in *QueryRequest, opts ...grpc.CallOption) (*QueryResponse, error)
	Status(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatusResponse, error)
	ListTasks(ctx context.Context, in *ListTasksRequest, opts ...grpc.CallOption) (*ListTasksResponse, error)
}

type yaagwaDBClient struct {
	cc grpc.ClientConnInterface
}

func NewYaagwaDBClient(cc grpc.ClientConnInterface) YaagwaDBClient {
	return &yaagwaDBClient{cc}
}

func (c *yaagwaDBClient) Lookup(ctx context.Context, in *LookupRequest, opts ...grpc.CallOption) (*Attributes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Attributes)
	err := c.cc.Invoke(ctx, YaagwaDB_Lookup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Getattr(ctx context.Context, in *GetattrRequest, opts ...grpc.CallOption) (*Attributes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Attributes)
	err := c.cc.Invoke(ctx, YaagwaDB_Getattr_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Setattr(ctx context.Context, in *SetattrRequest, opts ...grpc.CallOption) (*Attributes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Attributes)
	err := c.cc.Invoke(ctx, YaagwaDB_Setattr_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Readdir(ctx context.Context, in *ReaddirRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[DirectoryEntry], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &YaagwaDB_ServiceDesc.Streams[0], YaagwaDB_Readdir_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[ReaddirRequest, DirectoryEntry]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type YaagwaDB_ReaddirClient = grpc.ServerStreamingClient[DirectoryEntry]

func (c *yaagwaDBClient) Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[ReadResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &YaagwaDB_ServiceDesc.Streams[1], YaagwaDB_Read_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[ReadRequest, ReadResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type YaagwaDB_ReadClient = grpc.ServerStreamingClient[ReadResponse]

func (c *yaagwaDBClient) Write(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[WriteRequest, WriteResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &YaagwaDB_ServiceDesc.Streams[2], YaagwaDB_Write_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[WriteRequest, WriteResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type YaagwaDB_WriteClient = grpc.ClientStreamingClient[WriteRequest, WriteResponse]

func (c *yaagwaDBClient) Unlink(ctx context.Context, in *UnlinkRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, YaagwaDB_Unlink_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Mkdir(ctx context.Context, in *MkdirRequest, opts ...grpc.CallOption) (*Attributes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Attributes)
	err := c.cc.Invoke(ctx, YaagwaDB_Mkdir_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Rmdir(ctx context.Context, in *RmdirRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, YaagwaDB_Rmdir_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Rename(ctx context.Context, in *RenameRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, YaagwaDB_Rename_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Getxattr(ctx context.Context, in *GetxattrRequest, opts ...grpc.CallOption) (*GetxattrResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetxattrResponse)
	err := c.cc.Invoke(ctx, YaagwaDB_Getxattr_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Setxattr(ctx context.Context, in *SetxattrRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, YaagwaDB_Setxattr_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Listxattr(ctx context.Context, in *ListxattrRequest, opts ...grpc.CallOption) (*ListxattrResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListxattrResponse)
	err := c.cc.Invoke(ctx, YaagwaDB_Listxattr_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Removexattr(ctx context.Context, in *RemovexattrRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, YaagwaDB_Removexattr_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Statfs(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatfsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StatfsResponse)
	err := c.cc.Invoke(ctx, YaagwaDB_Statfs_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Query(ctx context.Context, in *QueryRequest, opts ...grpc.CallOption) (*QueryResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryResponse)
	err := c.cc.Invoke(ctx, YaagwaDB_Query_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) Status(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*StatusResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, YaagwaDB_Status_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yaagwaDBClient) ListTasks(ctx context.Context, in *ListTasksRequest, opts ...grpc.CallOption) (*ListTasksResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListTasksResponse)
	err := c.cc.Invoke(ctx, YaagwaDB_ListTasks_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// YaagwaDBServer is the server API for YaagwaDB service.
// All implementations must embed UnimplementedYaagwaDBServer
// for forward compatibility.
type YaagwaDBServer interface {
	Lookup(context.Context, *LookupRequest) (*Attributes, error)
	Getattr(context.Context, *GetattrRequest) (*Attributes, error)
	Setattr(context.Context, *SetattrRequest) (*Attributes, error)
	Readdir(*ReaddirRequest, grpc.ServerStreamingServer[DirectoryEntry]) error
	Read(*ReadRequest, grpc.ServerStreamingServer[ReadResponse]) error
	Write(grpc.ClientStreamingServer[WriteRequest, WriteResponse]) error
	Unlink(context.Context, *UnlinkRequest) (*Empty, error)
	Mkdir(context.Context, *MkdirRequest) (*Attributes, error)
	Rmdir(context.Context, *RmdirRequest) (*Empty, error)
	Rename(context.Context, *RenameRequest) (*Empty, error)
	Getxattr(context.Context, *GetxattrRequest) (*GetxattrResponse, error)
	Setxattr(context.Context, *SetxattrRequest) (*Empty, error)
	Listxattr(context.Context, *ListxattrRequest) (*ListxattrResponse, error)
	Removexattr(context.Context, *RemovexattrRequest) (*Empty, error)
	Statfs(context.Context, *Empty) (*StatfsResponse, error)
	Query(context.Context, *QueryRequest) (*QueryResponse, error)
	Status(context.Context, *Empty) (*StatusResponse, error)
	ListTasks(context.Context, *ListTasksRequest) (*ListTasksResponse, error)
	mustEmbedUnimplementedYaagwaDBServer()
}

// UnimplementedYaagwaDBServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedYaagwaDBServer struct{}

func (UnimplementedYaagwaDBServer) Lookup(context.Context, *LookupRequest) (*Attributes, error) {
	return nil, status.Error(codes.Unimplemented, "method Lookup not implemented")
}
func (UnimplementedYaagwaDBServer) Getattr(context.Context, *GetattrRequest) (*Attributes, error) {
	return nil, status.Error(codes.Unimplemented, "method Getattr not implemented")
}
func (UnimplementedYaagwaDBServer) Setattr(context.Context, *SetattrRequest) (*Attributes, error) {
	return nil, status.Error(codes.Unimplemented, "method Setattr not implemented")
}
func (UnimplementedYaagwaDBServer) Readdir(*ReaddirRequest, grpc.ServerStreamingServer[DirectoryEntry]) error {
	return status.Error(codes.Unimplemented, "method Readdir not implemented")
}
func (UnimplementedYaagwaDBServer) Read(*ReadRequest, grpc.ServerStreamingServer[ReadResponse]) error {
	return status.Error(codes.Unimplemented, "method Read not implemented")
}
func (UnimplementedYaagwaDBServer) Write(grpc.ClientStreamingServer[WriteRequest, WriteResponse]) error {
	return status.Error(codes.Unimplemented, "method Write not implemented")
}
func (UnimplementedYaagwaDBServer) Unlink(context.Context, *UnlinkRequest) (*Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method Unlink not implemented")
}
func (UnimplementedYaagwaDBServer) Mkdir(context.Context, *MkdirRequest) (*Attributes, error) {
	return nil, status.Error(codes.Unimplemented, "method Mkdir not implemented")
}
func (UnimplementedYaagwaDBServer) Rmdir(context.Context, *RmdirRequest) (*Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method Rmdir not implemented")
}
func (UnimplementedYaagwaDBServer) Rename(context.Context, *RenameRequest) (*Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method Rename not implemented")
}
func (UnimplementedYaagwaDBServer) Getxattr(context.Context, *GetxattrRequest) (*GetxattrResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Getxattr not implemented")
}
func (UnimplementedYaagwaDBServer) Setxattr(context.Context, *SetxattrRequest) (*Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method Setxattr not implemented")
}
func (UnimplementedYaagwaDBServer) Listxattr(context.Context, *ListxattrRequest) (*ListxattrResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Listxattr not implemented")
}
func (UnimplementedYaagwaDBServer) Removexattr(context.Context, *RemovexattrRequest) (*Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method Removexattr not implemented")
}
func (UnimplementedYaagwaDBServer) Statfs(context.Context, *Empty) (*StatfsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Statfs not implemented")
}
func (UnimplementedYaagwaDBServer) Query(context.Context, *QueryRequest) (*QueryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Query not implemented")
}
func (UnimplementedYaagwaDBServer) Status(context.Context, *Empty) (*StatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedYaagwaDBServer) ListTasks(context.Context, *ListTasksRequest) (*ListTasksResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method ListTasks not implemented")
}
func (UnimplementedYaagwaDBServer) mustEmbedUnimplementedYaagwaDBServer() {}
func (UnimplementedYaagwaDBServer) testEmbeddedByValue()                  {}

// UnsafeYaagwaDBServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to YaagwaDBServer will
// result in compilation errors.
type UnsafeYaagwaDBServer interface {
	mustEmbedUnimplementedYaagwaDBServer()
}

func RegisterYaagwaDBServer(s grpc.ServiceRegistrar, srv YaagwaDBServer) {
	// If the following call panics, it indicates UnimplementedYaagwaDBServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&YaagwaDB_ServiceDesc, srv)
}

func _YaagwaDB_Lookup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LookupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Lookup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Lookup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Lookup(ctx, req.(*LookupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Getattr_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetattrRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Getattr(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Getattr_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Getattr(ctx, req.(*GetattrRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Setattr_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetattrRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Setattr(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Setattr_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Setattr(ctx, req.(*SetattrRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Readdir_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ReaddirRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(YaagwaDBServer).Readdir(m, &grpc.GenericServerStream[ReaddirRequest, DirectoryEntry]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type YaagwaDB_ReaddirServer = grpc.ServerStreamingServer[DirectoryEntry]

func _YaagwaDB_Read_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ReadRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(YaagwaDBServer).Read(m, &grpc.GenericServerStream[ReadRequest, ReadResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type YaagwaDB_ReadServer = grpc.ServerStreamingServer[ReadResponse]

func _YaagwaDB_Write_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(YaagwaDBServer).Write(&grpc.GenericServerStream[WriteRequest, WriteResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type YaagwaDB_WriteServer = grpc.ClientStreamingServer[WriteRequest, WriteResponse]

func _YaagwaDB_Unlink_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnlinkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Unlink(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Unlink_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Unlink(ctx, req.(*UnlinkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Mkdir_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MkdirRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Mkdir(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Mkdir_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Mkdir(ctx, req.(*MkdirRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Rmdir_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RmdirRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Rmdir(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Rmdir_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Rmdir(ctx, req.(*RmdirRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Rename_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RenameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Rename(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Rename_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Rename(ctx, req.(*RenameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Getxattr_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetxattrRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Getxattr(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Getxattr_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Getxattr(ctx, req.(*GetxattrRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Setxattr_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetxattrRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Setxattr(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Setxattr_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Setxattr(ctx, req.(*SetxattrRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Listxattr_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListxattrRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Listxattr(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Listxattr_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Listxattr(ctx, req.(*ListxattrRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Removexattr_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemovexattrRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Removexattr(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Removexattr_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Removexattr(ctx, req.(*RemovexattrRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Statfs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Statfs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Statfs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Statfs(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Query_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Query(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Query_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Query(ctx, req.(*QueryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_Status_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).Status(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _YaagwaDB_ListTasks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListTasksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YaagwaDBServer).ListTasks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YaagwaDB_ListTasks_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YaagwaDBServer).ListTasks(ctx, req.(*ListTasksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// YaagwaDB_ServiceDesc is the grpc.ServiceDesc for YaagwaDB service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var YaagwaDB_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "yaagwadb.YaagwaDB",
	HandlerType: (*YaagwaDBServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Lookup",
			Handler:    _YaagwaDB_Lookup_Handler,
		},
		{
			MethodName: "Getattr",
			Handler:    _YaagwaDB_Getattr_Handler,
		},
		{
			MethodName: "Setattr",
			Handler:    _YaagwaDB_Setattr_Handler,
		},
		{
			MethodName: "Unlink",
			Handler:    _YaagwaDB_Unlink_Handler,
		},
		{
			MethodName: "Mkdir",
			Handler:    _YaagwaDB_Mkdir_Handler,
		},
		{
			MethodName: "Rmdir",
			Handler:    _YaagwaDB_Rmdir_Handler,
		},
		{
			MethodName: "Rename",
			Handler:    _YaagwaDB_Rename_Handler,
		},
		{
			MethodName: "Getxattr",
			Handler:    _YaagwaDB_Getxattr_Handler,
		},
		{
			MethodName: "Setxattr",
			Handler:    _YaagwaDB_Setxattr_Handler,
		},
		{
			MethodName: "Listxattr",
			Handler:    _YaagwaDB_Listxattr_Handler,
		},
		{
			MethodName: "Removexattr",
			Handler:    _YaagwaDB_Removexattr_Handler,
		},
		{
			MethodName: "Statfs",
			Handler:    _YaagwaDB_Statfs_Handler,
		},
		{
			MethodName: "Query",
			Handler:    _YaagwaDB_Query_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _YaagwaDB_Status_Handler,
		},
		{
			MethodName: "ListTasks",
			Handler:    _YaagwaDB_ListTasks_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Readdir",
			Handler:       _YaagwaDB_Readdir_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "Read",
			Handler:       _YaagwaDB_Read_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "Write",
			Handler:       _YaagwaDB_Write_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "protocol/yaagwadb.proto",
}
