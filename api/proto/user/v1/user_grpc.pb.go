// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: user/v1/user.proto

package userpb

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
	UserService_UpdateProfile_FullMethodName            = "/user.v1.UserService/UpdateProfile"
	UserService_PatchProfile_FullMethodName             = "/user.v1.UserService/PatchProfile"
	UserService_UploadProfilePhoto_FullMethodName       = "/user.v1.UserService/UploadProfilePhoto"
	UserService_DeleteProfilePhoto_FullMethodName       = "/user.v1.UserService/DeleteProfilePhoto"
	UserService_GetProfile_FullMethodName               = "/user.v1.UserService/GetProfile"
	UserService_UpdatePartnerPreferences_FullMethodName = "/user.v1.UserService/UpdatePartnerPreferences"
	UserService_PatchPartnerPreferences_FullMethodName  = "/user.v1.UserService/PatchPartnerPreferences"
	UserService_GetPartnerPreferences_FullMethodName    = "/user.v1.UserService/GetPartnerPreferences"
)

// UserServiceClient is the client API for UserService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type UserServiceClient interface {
	// UpdateProfile updates user profile information
	UpdateProfile(ctx context.Context, in *UpdateProfileRequest, opts ...grpc.CallOption) (*UpdateProfileResponse, error)
	// PatchProfile partially updates user profile information
	PatchProfile(ctx context.Context, in *PatchProfileRequest, opts ...grpc.CallOption) (*UpdateProfileResponse, error)
	// UploadProfilePhoto uploads a profile photo for a user
	UploadProfilePhoto(ctx context.Context, in *UploadProfilePhotoRequest, opts ...grpc.CallOption) (*UploadProfilePhotoResponse, error)
	// DeleteProfilePhoto deletes the profile photo of a user
	DeleteProfilePhoto(ctx context.Context, in *DeleteProfilePhotoRequest, opts ...grpc.CallOption) (*DeleteProfilePhotoResponse, error)
	// GetProfile retrieves user profile information
	GetProfile(ctx context.Context, in *GetProfileRequest, opts ...grpc.CallOption) (*GetProfileResponse, error)
	// UpdatePartnerPreferences updates a user's partner preferences
	UpdatePartnerPreferences(ctx context.Context, in *UpdatePartnerPreferencesRequest, opts ...grpc.CallOption) (*UpdatePartnerPreferencesResponse, error)
	PatchPartnerPreferences(ctx context.Context, in *PatchPartnerPreferencesRequest, opts ...grpc.CallOption) (*UpdatePartnerPreferencesResponse, error)
	GetPartnerPreferences(ctx context.Context, in *GetPartnerPreferencesRequest, opts ...grpc.CallOption) (*GetPartnerPreferencesResponse, error)
}

type userServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewUserServiceClient(cc grpc.ClientConnInterface) UserServiceClient {
	return &userServiceClient{cc}
}

func (c *userServiceClient) UpdateProfile(ctx context.Context, in *UpdateProfileRequest, opts ...grpc.CallOption) (*UpdateProfileResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdateProfileResponse)
	err := c.cc.Invoke(ctx, UserService_UpdateProfile_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) PatchProfile(ctx context.Context, in *PatchProfileRequest, opts ...grpc.CallOption) (*UpdateProfileResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdateProfileResponse)
	err := c.cc.Invoke(ctx, UserService_PatchProfile_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) UploadProfilePhoto(ctx context.Context, in *UploadProfilePhotoRequest, opts ...grpc.CallOption) (*UploadProfilePhotoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UploadProfilePhotoResponse)
	err := c.cc.Invoke(ctx, UserService_UploadProfilePhoto_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) DeleteProfilePhoto(ctx context.Context, in *DeleteProfilePhotoRequest, opts ...grpc.CallOption) (*DeleteProfilePhotoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteProfilePhotoResponse)
	err := c.cc.Invoke(ctx, UserService_DeleteProfilePhoto_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) GetProfile(ctx context.Context, in *GetProfileRequest, opts ...grpc.CallOption) (*GetProfileResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetProfileResponse)
	err := c.cc.Invoke(ctx, UserService_GetProfile_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) UpdatePartnerPreferences(ctx context.Context, in *UpdatePartnerPreferencesRequest, opts ...grpc.CallOption) (*UpdatePartnerPreferencesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdatePartnerPreferencesResponse)
	err := c.cc.Invoke(ctx, UserService_UpdatePartnerPreferences_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) PatchPartnerPreferences(ctx context.Context, in *PatchPartnerPreferencesRequest, opts ...grpc.CallOption) (*UpdatePartnerPreferencesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdatePartnerPreferencesResponse)
	err := c.cc.Invoke(ctx, UserService_PatchPartnerPreferences_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) GetPartnerPreferences(ctx context.Context, in *GetPartnerPreferencesRequest, opts ...grpc.CallOption) (*GetPartnerPreferencesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetPartnerPreferencesResponse)
	err := c.cc.Invoke(ctx, UserService_GetPartnerPreferences_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UserServiceServer is the server API for UserService service.
// All implementations must embed UnimplementedUserServiceServer
// for forward compatibility.
type UserServiceServer interface {
	// UpdateProfile updates user profile information
	UpdateProfile(context.Context, *UpdateProfileRequest) (*UpdateProfileResponse, error)
	// PatchProfile partially updates user profile information
	PatchProfile(context.Context, *PatchProfileRequest) (*UpdateProfileResponse, error)
	// UploadProfilePhoto uploads a profile photo for a user
	UploadProfilePhoto(context.Context, *UploadProfilePhotoRequest) (*UploadProfilePhotoResponse, error)
	// DeleteProfilePhoto deletes the profile photo of a user
	DeleteProfilePhoto(context.Context, *DeleteProfilePhotoRequest) (*DeleteProfilePhotoResponse, error)
	// GetProfile retrieves user profile information
	GetProfile(context.Context, *GetProfileRequest) (*GetProfileResponse, error)
	// UpdatePartnerPreferences updates a user's partner preferences
	UpdatePartnerPreferences(context.Context, *UpdatePartnerPreferencesRequest) (*UpdatePartnerPreferencesResponse, error)
	PatchPartnerPreferences(context.Context, *PatchPartnerPreferencesRequest) (*UpdatePartnerPreferencesResponse, error)
	GetPartnerPreferences(context.Context, *GetPartnerPreferencesRequest) (*GetPartnerPreferencesResponse, error)
	mustEmbedUnimplementedUserServiceServer()
}

// UnimplementedUserServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedUserServiceServer struct{}

func (UnimplementedUserServiceServer) UpdateProfile(context.Context, *UpdateProfileRequest) (*UpdateProfileResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateProfile not implemented")
}
func (UnimplementedUserServiceServer) PatchProfile(context.Context, *PatchProfileRequest) (*UpdateProfileResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PatchProfile not implemented")
}
func (UnimplementedUserServiceServer) UploadProfilePhoto(context.Context, *UploadProfilePhotoRequest) (*UploadProfilePhotoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UploadProfilePhoto not implemented")
}
func (UnimplementedUserServiceServer) DeleteProfilePhoto(context.Context, *DeleteProfilePhotoRequest) (*DeleteProfilePhotoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteProfilePhoto not implemented")
}
func (UnimplementedUserServiceServer) GetProfile(context.Context, *GetProfileRequest) (*GetProfileResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetProfile not implemented")
}
func (UnimplementedUserServiceServer) UpdatePartnerPreferences(context.Context, *UpdatePartnerPreferencesRequest) (*UpdatePartnerPreferencesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdatePartnerPreferences not implemented")
}
func (UnimplementedUserServiceServer) PatchPartnerPreferences(context.Context, *PatchPartnerPreferencesRequest) (*UpdatePartnerPreferencesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PatchPartnerPreferences not implemented")
}
func (UnimplementedUserServiceServer) GetPartnerPreferences(context.Context, *GetPartnerPreferencesRequest) (*GetPartnerPreferencesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPartnerPreferences not implemented")
}
func (UnimplementedUserServiceServer) mustEmbedUnimplementedUserServiceServer() {}
func (UnimplementedUserServiceServer) testEmbeddedByValue()                     {}

// UnsafeUserServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to UserServiceServer will
// result in compilation errors.
type UnsafeUserServiceServer interface {
	mustEmbedUnimplementedUserServiceServer()
}

func RegisterUserServiceServer(s grpc.ServiceRegistrar, srv UserServiceServer) {
	// If the following call pancis, it indicates UnimplementedUserServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&UserService_ServiceDesc, srv)
}

func _UserService_UpdateProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).UpdateProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserService_UpdateProfile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).UpdateProfile(ctx, req.(*UpdateProfileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_PatchProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PatchProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).PatchProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserService_PatchProfile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).PatchProfile(ctx, req.(*PatchProfileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_UploadProfilePhoto_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UploadProfilePhotoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).UploadProfilePhoto(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserService_UploadProfilePhoto_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).UploadProfilePhoto(ctx, req.(*UploadProfilePhotoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_DeleteProfilePhoto_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteProfilePhotoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).DeleteProfilePhoto(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserService_DeleteProfilePhoto_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).DeleteProfilePhoto(ctx, req.(*DeleteProfilePhotoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_GetProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).GetProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserService_GetProfile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).GetProfile(ctx, req.(*GetProfileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_UpdatePartnerPreferences_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdatePartnerPreferencesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).UpdatePartnerPreferences(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserService_UpdatePartnerPreferences_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).UpdatePartnerPreferences(ctx, req.(*UpdatePartnerPreferencesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_PatchPartnerPreferences_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PatchPartnerPreferencesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).PatchPartnerPreferences(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserService_PatchPartnerPreferences_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).PatchPartnerPreferences(ctx, req.(*PatchPartnerPreferencesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_GetPartnerPreferences_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPartnerPreferencesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).GetPartnerPreferences(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserService_GetPartnerPreferences_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).GetPartnerPreferences(ctx, req.(*GetPartnerPreferencesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// UserService_ServiceDesc is the grpc.ServiceDesc for UserService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var UserService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "user.v1.UserService",
	HandlerType: (*UserServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateProfile",
			Handler:    _UserService_UpdateProfile_Handler,
		},
		{
			MethodName: "PatchProfile",
			Handler:    _UserService_PatchProfile_Handler,
		},
		{
			MethodName: "UploadProfilePhoto",
			Handler:    _UserService_UploadProfilePhoto_Handler,
		},
		{
			MethodName: "DeleteProfilePhoto",
			Handler:    _UserService_DeleteProfilePhoto_Handler,
		},
		{
			MethodName: "GetProfile",
			Handler:    _UserService_GetProfile_Handler,
		},
		{
			MethodName: "UpdatePartnerPreferences",
			Handler:    _UserService_UpdatePartnerPreferences_Handler,
		},
		{
			MethodName: "PatchPartnerPreferences",
			Handler:    _UserService_PatchPartnerPreferences_Handler,
		},
		{
			MethodName: "GetPartnerPreferences",
			Handler:    _UserService_GetPartnerPreferences_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "user/v1/user.proto",
}
