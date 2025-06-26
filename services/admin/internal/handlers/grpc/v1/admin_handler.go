package v1

import (
	"context"

	adminpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/admin/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/domain/services"
)

type AdminHandler struct {
	adminpb.UnimplementedAdminServiceServer
	adminService *services.AdminService
	logger       logging.Logger
}

func NewAdminHandler(
	adminService *services.AdminService,
	logger logging.Logger,
) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
		logger:       logger,
	}
}

func (h *AdminHandler) GetUsers(ctx context.Context, req *adminpb.GetUsersRequest) (*adminpb.GetUsersResponse, error) {
	return h.adminService.GetUsers(ctx, req)
}

func (h *AdminHandler) GetUser(ctx context.Context, req *adminpb.GetUserRequest) (*adminpb.GetUserResponse, error) {
	return h.adminService.GetUser(ctx, req)
}
