// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package generated

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// ArticlesServiceClient is the client API for ArticlesService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ArticlesServiceClient interface {
	List(ctx context.Context, in *ListArticlesRequest, opts ...grpc.CallOption) (*ArticlesResponse, error)
}

type articlesServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewArticlesServiceClient(cc grpc.ClientConnInterface) ArticlesServiceClient {
	return &articlesServiceClient{cc}
}

func (c *articlesServiceClient) List(ctx context.Context, in *ListArticlesRequest, opts ...grpc.CallOption) (*ArticlesResponse, error) {
	out := new(ArticlesResponse)
	err := c.cc.Invoke(ctx, "/generated.ArticlesService/List", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ArticlesServiceServer is the server API for ArticlesService service.
// All implementations must embed UnimplementedArticlesServiceServer
// for forward compatibility
type ArticlesServiceServer interface {
	List(context.Context, *ListArticlesRequest) (*ArticlesResponse, error)
	mustEmbedUnimplementedArticlesServiceServer()
}

// UnimplementedArticlesServiceServer must be embedded to have forward compatible implementations.
type UnimplementedArticlesServiceServer struct {
}

func (UnimplementedArticlesServiceServer) List(context.Context, *ListArticlesRequest) (*ArticlesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedArticlesServiceServer) mustEmbedUnimplementedArticlesServiceServer() {}

// UnsafeArticlesServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ArticlesServiceServer will
// result in compilation errors.
type UnsafeArticlesServiceServer interface {
	mustEmbedUnimplementedArticlesServiceServer()
}

func RegisterArticlesServiceServer(s grpc.ServiceRegistrar, srv ArticlesServiceServer) {
	s.RegisterService(&_ArticlesService_serviceDesc, srv)
}

func _ArticlesService_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListArticlesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArticlesServiceServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/generated.ArticlesService/List",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArticlesServiceServer).List(ctx, req.(*ListArticlesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _ArticlesService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "generated.ArticlesService",
	HandlerType: (*ArticlesServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "List",
			Handler:    _ArticlesService_List_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "articles.proto",
}
