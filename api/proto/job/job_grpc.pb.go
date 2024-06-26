// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.6
// source: api/proto/job/job.proto

package job

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// JobClient is the client API for Job service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type JobClient interface {
	// Start a new job and begin execution of the specified command immediately
	Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*Response, error)
	// Stop execution of the specified job immediately
	Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*Response, error)
	// Query details about specified job; this function can run on a job of any status
	Query(ctx context.Context, in *QueryRequest, opts ...grpc.CallOption) (*Response, error)
	// Get the full output (stdout and stderr) of any existing job
	Output(ctx context.Context, in *OutputRequest, opts ...grpc.CallOption) (Job_OutputClient, error)
}

type jobClient struct {
	cc grpc.ClientConnInterface
}

func NewJobClient(cc grpc.ClientConnInterface) JobClient {
	return &jobClient{cc}
}

func (c *jobClient) Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/job.Job/Start", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *jobClient) Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/job.Job/Stop", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *jobClient) Query(ctx context.Context, in *QueryRequest, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/job.Job/Query", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *jobClient) Output(ctx context.Context, in *OutputRequest, opts ...grpc.CallOption) (Job_OutputClient, error) {
	stream, err := c.cc.NewStream(ctx, &Job_ServiceDesc.Streams[0], "/job.Job/Output", opts...)
	if err != nil {
		return nil, err
	}
	x := &jobOutputClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Job_OutputClient interface {
	Recv() (*OutputResponse, error)
	grpc.ClientStream
}

type jobOutputClient struct {
	grpc.ClientStream
}

func (x *jobOutputClient) Recv() (*OutputResponse, error) {
	m := new(OutputResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// JobServer is the server API for Job service.
// All implementations must embed UnimplementedJobServer
// for forward compatibility
type JobServer interface {
	// Start a new job and begin execution of the specified command immediately
	Start(context.Context, *StartRequest) (*Response, error)
	// Stop execution of the specified job immediately
	Stop(context.Context, *StopRequest) (*Response, error)
	// Query details about specified job; this function can run on a job of any status
	Query(context.Context, *QueryRequest) (*Response, error)
	// Get the full output (stdout and stderr) of any existing job
	Output(*OutputRequest, Job_OutputServer) error
	mustEmbedUnimplementedJobServer()
}

// UnimplementedJobServer must be embedded to have forward compatible implementations.
type UnimplementedJobServer struct {
}

func (UnimplementedJobServer) Start(context.Context, *StartRequest) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Start not implemented")
}
func (UnimplementedJobServer) Stop(context.Context, *StopRequest) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
func (UnimplementedJobServer) Query(context.Context, *QueryRequest) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Query not implemented")
}
func (UnimplementedJobServer) Output(*OutputRequest, Job_OutputServer) error {
	return status.Errorf(codes.Unimplemented, "method Output not implemented")
}
func (UnimplementedJobServer) mustEmbedUnimplementedJobServer() {}

// UnsafeJobServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to JobServer will
// result in compilation errors.
type UnsafeJobServer interface {
	mustEmbedUnimplementedJobServer()
}

func RegisterJobServer(s grpc.ServiceRegistrar, srv JobServer) {
	s.RegisterService(&Job_ServiceDesc, srv)
}

func _Job_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JobServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/job.Job/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JobServer).Start(ctx, req.(*StartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Job_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JobServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/job.Job/Stop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JobServer).Stop(ctx, req.(*StopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Job_Query_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JobServer).Query(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/job.Job/Query",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JobServer).Query(ctx, req.(*QueryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Job_Output_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(OutputRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(JobServer).Output(m, &jobOutputServer{stream})
}

type Job_OutputServer interface {
	Send(*OutputResponse) error
	grpc.ServerStream
}

type jobOutputServer struct {
	grpc.ServerStream
}

func (x *jobOutputServer) Send(m *OutputResponse) error {
	return x.ServerStream.SendMsg(m)
}

// Job_ServiceDesc is the grpc.ServiceDesc for Job service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Job_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "job.Job",
	HandlerType: (*JobServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Start",
			Handler:    _Job_Start_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _Job_Stop_Handler,
		},
		{
			MethodName: "Query",
			Handler:    _Job_Query_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Output",
			Handler:       _Job_Output_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/proto/job/job.proto",
}
