// pkg/validation/request.go
package validation

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/errors"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/http/response"
)

var validate = validator.New()

// BindAndValidate binds and validates a request
func BindAndValidate(c *gin.Context, req interface{}) error {
	if err := c.ShouldBind(req); err != nil {
		return errors.Wrap(err, errors.CodeInvalidArgument, "Invalid request format")
	}

	if err := validate.Struct(req); err != nil {
		return errors.Wrap(err, errors.CodeInvalidArgument, "Request validation failed")
	}

	return nil
}

// ValidateRequest handles binding, validation and standard error responses
func ValidateRequest(c *gin.Context, req interface{}) bool {
	if err := BindAndValidate(c, req); err != nil {
		response.SendError(c, err, "Request validation failed")
		return false
	}
	return true
}
