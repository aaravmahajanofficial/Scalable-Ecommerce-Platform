package errors_test

import (
	"errors"
	"net/http"
	"testing"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewAppError(t *testing.T) {
	err := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest)

	assert.NotNil(t, err)
	assert.Equal(t, "TEST_CODE", err.Code)
	assert.Equal(t, "Test message", err.Message)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Empty(t, err.Detail)
	assert.Nil(t, err.Err)
}

func TestAppError_Error(t *testing.T) {
	err := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest)
	assert.Equal(t, "Test message", err.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest).WithError(innerErr)

	assert.Equal(t, innerErr, err.Unwrap())
}

func TestAppError_WithDetail(t *testing.T) {
	err := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest).WithDetail("Some details")

	assert.Equal(t, "Some details", err.Detail)
}

func TestAppError_WithError(t *testing.T) {
	innerErr := errors.New("inner error")
	err := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest).WithError(innerErr)

	assert.Equal(t, innerErr, err.Err)
}

func TestIsAppError(t *testing.T) {
	t.Run("when it is an AppError", func(t *testing.T) {
		appErr := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest)
		wrappedErr := errors.Join(errors.New("wrapper"), appErr)

		extractedErr, ok := appErrors.IsAppError(wrappedErr)
		assert.True(t, ok)
		assert.Equal(t, appErr, extractedErr)
	})

	t.Run("when it is not an AppError", func(t *testing.T) {
		stdErr := errors.New("standard error")

		extractedErr, ok := appErrors.IsAppError(stdErr)
		assert.False(t, ok)
		assert.Nil(t, extractedErr)
	})
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) *appErrors.AppError
		msg      string
		expectedCode string
		expectedStatus int
	}{
		{"ValidationError", appErrors.ValidationError, "val err", appErrors.ErrCodeValidation, http.StatusBadRequest},
		{"BadRequestError", appErrors.BadRequestError, "bad req", appErrors.ErrCodeBadRequest, http.StatusBadRequest},
		{"NotFoundError", appErrors.NotFoundError, "not found", appErrors.ErrCodeNotFound, http.StatusNotFound},
		{"UnauthorizedError", appErrors.UnauthorizedError, "unauth", appErrors.ErrCodeUnauthorized, http.StatusUnauthorized},
		{"ForbiddenError", appErrors.ForbiddenError, "forbidden", appErrors.ErrCodeForbidden, http.StatusForbidden},
		{"InternalError", appErrors.InternalError, "internal", appErrors.ErrCodeInternal, http.StatusInternalServerError},
		{"DatabaseError", appErrors.DatabaseError, "db err", appErrors.ErrCodeDatabaseError, http.StatusInternalServerError},
		{"DuplicateEntryError", appErrors.DuplicateEntryError, "dup", appErrors.ErrCodeDuplicateEntry, http.StatusConflict},
		{"ThirdPartyError", appErrors.ThirdPartyError, "third party", appErrors.ErrCodeThirdPartyError, http.StatusInternalServerError},
		{"TooManyRequestsError", appErrors.TooManyRequestsError, "too many", appErrors.ErrCodeTooManyRequests, http.StatusTooManyRequests},
		{"ResourceExhaustedError", appErrors.ResourceExhaustedError, "exhausted", appErrors.ErrCodeResourceExhausted, http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.msg)
			assert.Equal(t, tt.expectedCode, err.Code)
			assert.Equal(t, tt.msg, err.Message)
			assert.Equal(t, tt.expectedStatus, err.StatusCode)
		})
	}
}

func TestAddValidationError(t *testing.T) {
	err := appErrors.AddValidationError("email", "invalid format")
	assert.Equal(t, appErrors.ErrCodeValidation, err.Code)
	assert.Equal(t, "Invalid field 'email': invalid format", err.Message)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
}
