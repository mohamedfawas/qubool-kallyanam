package user

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

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
		"user-id": userID,
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
		"user-id": userID,
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

func (c *Client) DeleteProfilePhoto(ctx context.Context, userID string) (bool, string, error) {
	// Create metadata with user ID
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.DeleteProfilePhotoRequest{}
	resp, err := c.client.DeleteProfilePhoto(ctx, req)
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

func (c *Client) PatchProfile(ctx context.Context, userID string,
	isBride *bool, fullName *string,
	dateOfBirth *time.Time, heightCM *int, physicallyChallenged *bool,
	community *string, maritalStatus *string, profession *string,
	professionType *string, highestEducationLevel *string, homeDistrict *string,
	clearDateOfBirth bool, clearHeightCM bool) (bool, string, error) {

	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.PatchProfileRequest{
		ClearDateOfBirth: clearDateOfBirth,
		ClearHeightCm:    clearHeightCM,
	}

	if isBride != nil {
		req.IsBride = &wrapperspb.BoolValue{Value: *isBride}
	}

	if fullName != nil {
		req.FullName = &wrapperspb.StringValue{Value: *fullName}
	}

	if dateOfBirth != nil && !clearDateOfBirth {
		req.DateOfBirth = timestamppb.New(*dateOfBirth)
	}

	if heightCM != nil && !clearHeightCM {
		req.HeightCm = &wrapperspb.Int32Value{Value: int32(*heightCM)}
	}

	if physicallyChallenged != nil {
		req.PhysicallyChallenged = &wrapperspb.BoolValue{Value: *physicallyChallenged}
	}

	if community != nil {
		req.Community = &wrapperspb.StringValue{Value: *community}
	}

	if maritalStatus != nil {
		req.MaritalStatus = &wrapperspb.StringValue{Value: *maritalStatus}
	}

	if profession != nil {
		req.Profession = &wrapperspb.StringValue{Value: *profession}
	}

	if professionType != nil {
		req.ProfessionType = &wrapperspb.StringValue{Value: *professionType}
	}

	if highestEducationLevel != nil {
		req.HighestEducationLevel = &wrapperspb.StringValue{Value: *highestEducationLevel}
	}

	if homeDistrict != nil {
		req.HomeDistrict = &wrapperspb.StringValue{Value: *homeDistrict}
	}

	resp, err := c.client.PatchProfile(ctx, req)
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

func (c *Client) GetProfile(ctx context.Context, userID string) (bool, string, *userpb.ProfileData, error) {
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.GetProfileRequest{}
	resp, err := c.client.GetProfile(ctx, req)
	if err != nil {
		return false, "", nil, err
	}

	return resp.Success, resp.Message, resp.Profile, nil
}

// UpdatePartnerPreferences calls the UpdatePartnerPreferences RPC
func (c *Client) UpdatePartnerPreferences(
	ctx context.Context,
	userID string,
	minAgeYears *int,
	maxAgeYears *int,
	minHeightCM *int,
	maxHeightCM *int,
	acceptPhysicallyChallenged bool,
	preferredCommunities []string,
	preferredMaritalStatus []string,
	preferredProfessions []string,
	preferredProfessionTypes []string,
	preferredEducationLevels []string,
	preferredHomeDistricts []string,
) (bool, string, error) {
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.UpdatePartnerPreferencesRequest{
		AcceptPhysicallyChallenged: acceptPhysicallyChallenged,
		PreferredCommunities:       preferredCommunities,
		PreferredMaritalStatus:     preferredMaritalStatus,
		PreferredProfessions:       preferredProfessions,
		PreferredProfessionTypes:   preferredProfessionTypes,
		PreferredEducationLevels:   preferredEducationLevels,
		PreferredHomeDistricts:     preferredHomeDistricts,
	}

	if minAgeYears != nil {
		req.MinAgeYears = int32(*minAgeYears)
	}
	if maxAgeYears != nil {
		req.MaxAgeYears = int32(*maxAgeYears)
	}
	if minHeightCM != nil {
		req.MinHeightCm = int32(*minHeightCM)
	}
	if maxHeightCM != nil {
		req.MaxHeightCm = int32(*maxHeightCM)
	}

	resp, err := c.client.UpdatePartnerPreferences(ctx, req)
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

func (c *Client) PatchPartnerPreferences(
	ctx context.Context,
	userID string,
	minAgeYears *int,
	maxAgeYears *int,
	minHeightCM *int,
	maxHeightCM *int,
	acceptPhysicallyChallenged *bool,
	preferredCommunities []string,
	preferredMaritalStatus []string,
	preferredProfessions []string,
	preferredProfessionTypes []string,
	preferredEducationLevels []string,
	preferredHomeDistricts []string,
	clearPreferredCommunities bool,
	clearPreferredMaritalStatus bool,
	clearPreferredProfessions bool,
	clearPreferredProfessionTypes bool,
	clearPreferredEducationLevels bool,
	clearPreferredHomeDistricts bool,
) (bool, string, error) {
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.PatchPartnerPreferencesRequest{
		ClearPreferredCommunities:     clearPreferredCommunities,
		ClearPreferredMaritalStatus:   clearPreferredMaritalStatus,
		ClearPreferredProfessions:     clearPreferredProfessions,
		ClearPreferredProfessionTypes: clearPreferredProfessionTypes,
		ClearPreferredEducationLevels: clearPreferredEducationLevels,
		ClearPreferredHomeDistricts:   clearPreferredHomeDistricts,
	}

	if minAgeYears != nil {
		req.MinAgeYears = &wrapperspb.Int32Value{Value: int32(*minAgeYears)}
	}
	if maxAgeYears != nil {
		req.MaxAgeYears = &wrapperspb.Int32Value{Value: int32(*maxAgeYears)}
	}
	if minHeightCM != nil {
		req.MinHeightCm = &wrapperspb.Int32Value{Value: int32(*minHeightCM)}
	}
	if maxHeightCM != nil {
		req.MaxHeightCm = &wrapperspb.Int32Value{Value: int32(*maxHeightCM)}
	}
	if acceptPhysicallyChallenged != nil {
		req.AcceptPhysicallyChallenged = &wrapperspb.BoolValue{Value: *acceptPhysicallyChallenged}
	}

	if len(preferredCommunities) > 0 {
		req.PreferredCommunities = preferredCommunities
	}
	if len(preferredMaritalStatus) > 0 {
		req.PreferredMaritalStatus = preferredMaritalStatus
	}
	if len(preferredProfessions) > 0 {
		req.PreferredProfessions = preferredProfessions
	}
	if len(preferredProfessionTypes) > 0 {
		req.PreferredProfessionTypes = preferredProfessionTypes
	}
	if len(preferredEducationLevels) > 0 {
		req.PreferredEducationLevels = preferredEducationLevels
	}
	if len(preferredHomeDistricts) > 0 {
		req.PreferredHomeDistricts = preferredHomeDistricts
	}

	resp, err := c.client.PatchPartnerPreferences(ctx, req)
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

func (c *Client) GetPartnerPreferences(ctx context.Context, userID string) (bool, string, *userpb.PartnerPreferencesData, error) {
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.GetPartnerPreferencesRequest{}
	resp, err := c.client.GetPartnerPreferences(ctx, req)
	if err != nil {
		return false, "", nil, err
	}

	return resp.Success, resp.Message, resp.Preferences, nil
}

// GetRecommendedMatches gets recommended profiles for the user
func (c *Client) GetRecommendedMatches(ctx context.Context, userID string, limit, offset int) (bool, string, []*userpb.RecommendedProfileData, *userpb.PaginationData, error) {
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.GetRecommendedMatchesRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	resp, err := c.client.GetRecommendedMatches(ctx, req)
	if err != nil {
		return false, "", nil, nil, err
	}

	return resp.Success, resp.Message, resp.Profiles, resp.Pagination, nil
}

// RecordMatchAction records a user's action on a potential match
// RecordMatchAction records a user's action on a potential match
func (c *Client) RecordMatchAction(ctx context.Context, userID string, profileID uint64, action string) (bool, string, bool, error) {
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.RecordMatchActionRequest{
		ProfileId: profileID, // Changed from TargetUserId
		Action:    action,
	}

	resp, err := c.client.RecordMatchAction(ctx, req)
	if err != nil {
		return false, "", false, err
	}

	return resp.Success, resp.Message, resp.IsMutualMatch, nil
}

// GetMatchHistory gets user's match history
func (c *Client) GetMatchHistory(ctx context.Context, userID string, status string, limit, offset int) (bool, string, []*userpb.MatchHistoryItem, *userpb.PaginationData, error) {
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.GetMatchHistoryRequest{
		Status: status,
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	resp, err := c.client.GetMatchHistory(ctx, req)
	if err != nil {
		return false, "", nil, nil, err
	}

	return resp.Success, resp.Message, resp.Matches, resp.Pagination, nil
}

// UpdateMatchAction updates a user's previous action on a potential match
func (c *Client) UpdateMatchAction(ctx context.Context, userID string, profileID uint64, action string) (bool, string, bool, bool, error) {
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.UpdateMatchActionRequest{
		ProfileId: profileID,
		Action:    action,
	}

	resp, err := c.client.UpdateMatchAction(ctx, req)
	if err != nil {
		return false, "", false, false, err
	}

	return resp.Success, resp.Message, resp.IsMutualMatch, resp.WasMutualMatchBroken, nil
}

func (c *Client) GetMutualMatches(ctx context.Context, userID string, limit, offset int) (bool, string, []*userpb.MutualMatchData, *userpb.PaginationData, error) {
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.GetMutualMatchesRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	resp, err := c.client.GetMutualMatches(ctx, req)
	if err != nil {
		return false, "", nil, nil, err
	}

	return resp.Success, resp.Message, resp.Matches, resp.Pagination, nil
}

// ResolveUserID resolves public ID to UUID
func (c *Client) ResolveUserID(ctx context.Context, publicID uint64) (string, error) {
	req := &userpb.GetProfileByIDRequest{
		ProfileId: publicID,
	}

	resp, err := c.client.GetProfileByID(ctx, req)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("user not found: %s", resp.Message)
	}

	return resp.UserUuid, nil
}

// GetBasicProfile gets basic user profile information
func (c *Client) GetBasicProfile(ctx context.Context, userUUID string) (*userpb.BasicProfileData, error) {
	req := &userpb.GetBasicProfileRequest{
		UserUuid: userUUID,
	}

	resp, err := c.client.GetBasicProfile(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("failed to get profile: %s", resp.Message)
	}

	return resp.Profile, nil
}

// UploadUserPhoto uploads an additional photo for a user
func (c *Client) UploadUserPhoto(ctx context.Context, userID string, photoData []byte, fileName string, contentType string, displayOrder int) (bool, string, string, error) {
	// Create metadata with user ID
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.UploadUserPhotoRequest{
		PhotoData:    photoData,
		FileName:     fileName,
		ContentType:  contentType,
		DisplayOrder: int32(displayOrder),
	}

	resp, err := c.client.UploadUserPhoto(ctx, req)
	if err != nil {
		return false, "", "", err
	}

	return resp.Success, resp.Message, resp.PhotoUrl, nil
}

// GetUserPhotos retrieves all photos for a user
func (c *Client) GetUserPhotos(ctx context.Context, userID string) (bool, string, []*userpb.UserPhotoData, error) {
	// Create metadata with user ID
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.GetUserPhotosRequest{}
	resp, err := c.client.GetUserPhotos(ctx, req)
	if err != nil {
		return false, "", nil, err
	}

	return resp.Success, resp.Message, resp.Photos, nil
}

// DeleteUserPhoto deletes a specific user photo
func (c *Client) DeleteUserPhoto(ctx context.Context, userID string, displayOrder int) (bool, string, error) {
	// Create metadata with user ID
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.DeleteUserPhotoRequest{
		DisplayOrder: int32(displayOrder),
	}

	resp, err := c.client.DeleteUserPhoto(ctx, req)
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

// UploadUserVideo uploads an introduction video for a user
func (c *Client) UploadUserVideo(ctx context.Context, userID string, videoData []byte, fileName string, contentType string) (bool, string, string, error) {
	// Create metadata with user ID
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.UploadUserVideoRequest{
		VideoData:   videoData,
		FileName:    fileName,
		ContentType: contentType,
	}

	resp, err := c.client.UploadUserVideo(ctx, req)
	if err != nil {
		return false, "", "", err
	}

	return resp.Success, resp.Message, resp.VideoUrl, nil
}

// GetUserVideo retrieves the video for a user
func (c *Client) GetUserVideo(ctx context.Context, userID string) (bool, string, *userpb.UserVideoData, error) {
	// Create metadata with user ID
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.GetUserVideoRequest{}
	resp, err := c.client.GetUserVideo(ctx, req)
	if err != nil {
		return false, "", nil, err
	}

	return resp.Success, resp.Message, resp.Video, nil
}

// DeleteUserVideo deletes the user's introduction video
func (c *Client) DeleteUserVideo(ctx context.Context, userID string) (bool, string, error) {
	// Create metadata with user ID
	md := metadata.New(map[string]string{
		"user-id": userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &userpb.DeleteUserVideoRequest{}
	resp, err := c.client.DeleteUserVideo(ctx, req)
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
