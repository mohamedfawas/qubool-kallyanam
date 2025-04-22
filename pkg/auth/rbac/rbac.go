package rbac

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Common errors
var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrPermissionDenied = errors.New("permission denied")
	ErrMissingClaims    = errors.New("missing auth claims in context")
)

// Permission represents a system permission
type Permission string

// Resource represents a system resource
type Resource string

// Action represents an operation on a resource
type Action string

// Commonly used resources
const (
	ResourceUser    Resource = "user"
	ResourceAdmin   Resource = "admin"
	ResourceProfile Resource = "profile"
	ResourceMatch   Resource = "match"
	ResourceChat    Resource = "chat"
	ResourceSystem  Resource = "system"
)

// Commonly used actions
const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionList   Action = "list"
	ActionManage Action = "manage" // Full control
)

// Format a permission as resource:action
func FormatPermission(resource Resource, action Action) Permission {
	return Permission(fmt.Sprintf("%s:%s", resource, action))
}

// ParsePermission parses a permission string into resource and action
func ParsePermission(permission Permission) (Resource, Action) {
	parts := strings.Split(string(permission), ":")
	if len(parts) != 2 {
		return "", ""
	}
	return Resource(parts[0]), Action(parts[1])
}

// Checker provides role-based access control
type Checker struct {
	rolePermissions map[jwt.Role]map[Permission]bool
	logger          logging.Logger
	mu              sync.RWMutex
}

// NewChecker creates a new RBAC checker with default permissions
func NewChecker(logger logging.Logger) *Checker {
	if logger == nil {
		logger = logging.Get().Named("rbac")
	}

	checker := &Checker{
		rolePermissions: make(map[jwt.Role]map[Permission]bool),
		logger:          logger,
	}

	// Initialize with default permissions
	checker.initializeDefaultPermissions()

	return checker
}

// initializeDefaultPermissions sets up default role permissions
func (c *Checker) initializeDefaultPermissions() {
	// ADMIN permissions (full access)
	adminPerms := map[Permission]bool{
		FormatPermission(ResourceSystem, ActionManage):  true,
		FormatPermission(ResourceUser, ActionManage):    true,
		FormatPermission(ResourceProfile, ActionManage): true,
		FormatPermission(ResourceMatch, ActionManage):   true,
		FormatPermission(ResourceChat, ActionManage):    true,
		FormatPermission(ResourceAdmin, ActionManage):   true,
	}

	// USER permissions (limited access)
	userPerms := map[Permission]bool{
		// Profile permissions
		FormatPermission(ResourceProfile, ActionCreate): true,
		FormatPermission(ResourceProfile, ActionRead):   true,
		FormatPermission(ResourceProfile, ActionUpdate): true,

		// Read-only access to other profiles
		FormatPermission(ResourceUser, ActionRead): true,

		// Match permissions
		FormatPermission(ResourceMatch, ActionRead):   true,
		FormatPermission(ResourceMatch, ActionCreate): true,

		// Chat permissions
		FormatPermission(ResourceChat, ActionCreate): true,
		FormatPermission(ResourceChat, ActionRead):   true,
		FormatPermission(ResourceChat, ActionUpdate): true,
	}

	c.rolePermissions[jwt.RoleAdmin] = adminPerms
	c.rolePermissions[jwt.RoleUser] = userPerms
}

// HasPermission checks if a role has a specific permission
func (c *Checker) HasPermission(role jwt.Role, permission Permission) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Admin always has all permissions
	if role == jwt.RoleAdmin {
		return true
	}

	perms, exists := c.rolePermissions[role]
	if !exists {
		return false
	}

	// Check for the specific permission
	if allowed, exists := perms[permission]; exists && allowed {
		return true
	}

	// Check for wildcard "manage" permission for the resource
	resource, _ := ParsePermission(permission)
	managePermission := FormatPermission(resource, ActionManage)

	return perms[managePermission]
}

// AddPermission adds a permission to a role
func (c *Checker) AddPermission(role jwt.Role, permission Permission) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.rolePermissions[role]; !exists {
		c.rolePermissions[role] = make(map[Permission]bool)
	}

	c.rolePermissions[role][permission] = true
}

// RemovePermission removes a permission from a role
func (c *Checker) RemovePermission(role jwt.Role, permission Permission) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if perms, exists := c.rolePermissions[role]; exists {
		delete(perms, permission)
	}
}

// CheckPermission checks if the current user has the required permission
func (c *Checker) CheckPermission(ctx context.Context, permission Permission) error {
	claims, ok := jwt.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return ErrMissingClaims
	}

	if !c.HasPermission(claims.Role, permission) {
		c.logger.Warn("Permission denied",
			logging.String("user_id", claims.UserID),
			logging.String("role", string(claims.Role)),
			logging.String("permission", string(permission)),
		)
		return ErrPermissionDenied
	}

	return nil
}

// RequirePermission is a helper to validate permissions and return standardized errors
func (c *Checker) RequirePermission(ctx context.Context, resource Resource, action Action) error {
	return c.CheckPermission(ctx, FormatPermission(resource, action))
}

// IsAdmin checks if the current user is an admin
func (c *Checker) IsAdmin(ctx context.Context) bool {
	claims, ok := jwt.ClaimsFromContext(ctx)
	return ok && claims != nil && claims.Role == jwt.RoleAdmin
}

// IsUser checks if the current user is a regular user
func (c *Checker) IsUser(ctx context.Context) bool {
	claims, ok := jwt.ClaimsFromContext(ctx)
	return ok && claims != nil && claims.Role == jwt.RoleUser
}

// IsResourceOwner checks if the current user is the owner of a resource
func (c *Checker) IsResourceOwner(ctx context.Context, ownerID string) bool {
	claims, ok := jwt.ClaimsFromContext(ctx)
	return ok && claims != nil && claims.UserID == ownerID
}

// CanAccessResource combines permission checks with ownership checks
// A user can access a resource if they have the permission or if they own the resource
func (c *Checker) CanAccessResource(ctx context.Context, resource Resource, action Action, ownerID string) error {
	claims, ok := jwt.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return ErrMissingClaims
	}

	// Admins can always access
	if claims.Role == jwt.RoleAdmin {
		return nil
	}

	// Check if user is the resource owner
	if claims.UserID == ownerID {
		return nil
	}

	// Check if user has the required permission
	return c.RequirePermission(ctx, resource, action)
}
