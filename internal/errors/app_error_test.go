package errors_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewAppError(t *testing.T) {
	code := "TEST_CODE"
	msg := "Test message"
	statusCode := http.StatusBadRequest

	err := appErrors.NewAppError(code, msg, statusCode)

	assert.NotNil(t, err)
	assert.Equal(t, code, err.Code)
	assert.Equal(t, msg, err.Message)
	assert.Equal(t, statusCode, err.StatusCode)
	assert.Empty(t, err.Detail)
	assert.Nil(t, err.Err)
}

func TestAppError_Error(t *testing.T) {
	msg := "Test message"
	err := appErrors.NewAppError("TEST_CODE", msg, http.StatusBadRequest)

	assert.Equal(t, msg, err.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest).WithError(innerErr)

	assert.Equal(t, innerErr, err.Unwrap())
}

func TestAppError_WithDetail(t *testing.T) {
	detail := "Some details"
	err := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest).WithDetail(detail)

	assert.Equal(t, detail, err.Detail)
}

func TestAppError_WithError(t *testing.T) {
	innerErr := errors.New("inner error")
	err := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest).WithError(innerErr)

	assert.Equal(t, innerErr, err.Err)
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name       string
		constructor func(string) *appErrors.AppError
		expectedCode string
		expectedStatus int
	}{
		{
			name: "ValidationError",
			constructor: appErrors.ValidationError,
			expectedCode: appErrors.ErrCodeValidation,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "BadRequestError",
			constructor: appErrors.BadRequestError,
			expectedCode: appErrors.ErrCodeBadRequest,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "NotFoundError",
			constructor: appErrors.NotFoundError,
			expectedCode: appErrors.ErrCodeNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "UnauthorizedError",
			constructor: appErrors.UnauthorizedError,
			expectedCode: appErrors.ErrCodeUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "ForbiddenError",
			constructor: appErrors.ForbiddenError,
			expectedCode: appErrors.ErrCodeForbidden,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "InternalError",
			constructor: appErrors.InternalError,
			expectedCode: appErrors.ErrCodeInternal,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "DatabaseError",
			constructor: appErrors.DatabaseError,
			expectedCode: appErrors.ErrCodeDatabaseError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "DuplicateEntryError",
			constructor: appErrors.DuplicateEntryError,
			expectedCode: appErrors.ErrCodeDuplicateEntry,
			expectedStatus: http.StatusConflict,
		},
		{
			name: "ThirdPartyError",
			constructor: appErrors.ThirdPartyError,
			expectedCode: appErrors.ErrCodeThirdPartyError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "TooManyRequestsError",
			constructor: appErrors.TooManyRequestsError,
			expectedCode: appErrors.ErrCodeTooManyRequests,
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name: "ResourceExhaustedError",
			constructor: appErrors.ResourceExhaustedError,
			expectedCode: appErrors.ErrCodeResourceExhausted,
			expectedStatus: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := "Test message"
			err := tt.constructor(msg)

			assert.NotNil(t, err)
			assert.Equal(t, tt.expectedCode, err.Code)
			assert.Equal(t, msg, err.Message)
			assert.Equal(t, tt.expectedStatus, err.StatusCode)
		})
	}
}

func TestIsAppError(t *testing.T) {
	appErr := appErrors.NewAppError("TEST_CODE", "Test message", http.StatusBadRequest)

	// Test directly
	err1, ok1 := appErrors.IsAppError(appErr)
	assert.True(t, ok1)
	assert.Equal(t, appErr, err1)

	// Test with fmt.Errorf wrapping
	wrappedErr := fmt.Errorf("wrapped: %w", appErr)
	err2, ok2 := appErrors.IsAppError(wrappedErr)
	assert.True(t, ok2)
	assert.Equal(t, appErr.Code, err2.Code)

	// Test with non-AppError
	stdErr := errors.New("standard error")
	err3, ok3 := appErrors.IsAppError(stdErr)
	assert.False(t, ok3)
	assert.Nil(t, err3)
}

func TestAddValidationError(t *testing.T) {
	field := "email"
	reason := "invalid format"

	err := appErrors.AddValidationError(field, reason)

	assert.NotNil(t, err)
	assert.Equal(t, appErrors.ErrCodeValidation, err.Code)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, "Invalid field 'email': invalid format", err.Message)
}
