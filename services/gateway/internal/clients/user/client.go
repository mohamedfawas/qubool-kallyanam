package user

// import (
// 	"context"
// 	"fmt"

// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/credentials/insecure"

// 	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
// )

// // Client wraps the user service client
// type Client struct {
// 	conn   *grpc.ClientConn
// 	client userpb.UserServiceClient
// }

// // NewClient creates a new user service client
// func NewClient(address string) (*Client, error) {
// 	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to connect to user service: %w", err)
// 	}

// 	return &Client{
// 		conn:   conn,
// 		client: userpb.NewUserServiceClient(conn),
// 	}, nil
// }

// // CreateProfile sends a request to create a user profile
// func (c *Client) CreateProfile(ctx context.Context, name string, age int32, gender string, religion string,
// 	caste string, motherTongue string, maritalStatus string, heightCm int32, education string,
// 	occupation string, annualIncome int64, city string, state string, country string, about string) (bool, string, string, error) {

// 	location := &userpb.LocationInfo{
// 		City:    city,
// 		State:   state,
// 		Country: country,
// 	}

// 	resp, err := c.client.CreateProfile(ctx, &userpb.CreateProfileRequest{
// 		Name:          name,
// 		Age:           age,
// 		Gender:        gender,
// 		Religion:      religion,
// 		Caste:         caste,
// 		MotherTongue:  motherTongue,
// 		MaritalStatus: maritalStatus,
// 		HeightCm:      heightCm,
// 		Education:     education,
// 		Occupation:    occupation,
// 		AnnualIncome:  annualIncome,
// 		Location:      location,
// 		About:         about,
// 	})
// 	if err != nil {
// 		return false, "", "", err
// 	}

// 	return resp.Success, resp.Id, resp.Message, nil
// }

// // Close closes the client connection
// func (c *Client) Close() error {
// 	return c.conn.Close()
// }
