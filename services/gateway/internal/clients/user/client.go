package user

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
)

type Client struct {
	conn   *grpc.ClientConn
	client userpb.UserServiceClient
}

func NewClient(address string) (*Client, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	return &Client{
		conn:   conn,
		client: userpb.NewUserServiceClient(conn),
	}, nil
}

func (c *Client) UpdateProfile(ctx context.Context, userID string,
	isBride bool, fullName string,
	dateOfBirth *time.Time, heightCM int, physicallyChallenged bool,
	community string, maritalStatus string, profession string,
	professionType string, highestEducationLevel string, homeDistrict string) (bool, string, error) {

	// Create metadata with user ID
	md := metadata.New(map[string]string{
		"user_id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.UpdateProfileRequest{
		IsBride:               isBride,
		FullName:              fullName,
		PhysicallyChallenged:  physicallyChallenged,
		Community:             community,
		MaritalStatus:         maritalStatus,
		Profession:            profession,
		ProfessionType:        professionType,
		HighestEducationLevel: highestEducationLevel,
		HomeDistrict:          homeDistrict,
		HeightCm:              int32(heightCM),
	}

	if dateOfBirth != nil {
		req.DateOfBirth = timestamppb.New(*dateOfBirth)
	}

	resp, err := c.client.UpdateProfile(ctx, req)
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

func (c *Client) UploadProfilePhoto(ctx context.Context, userID string, photoData []byte, fileName string, contentType string) (bool, string, string, error) {
	// Create metadata with user ID
	md := metadata.New(map[string]string{
		"user_id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.UploadProfilePhotoRequest{
		PhotoData:   photoData,
		FileName:    fileName,
		ContentType: contentType,
	}

	resp, err := c.client.UploadProfilePhoto(ctx, req)
	if err != nil {
		return false, "", "", err
	}

	return resp.Success, resp.Message, resp.PhotoUrl, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
