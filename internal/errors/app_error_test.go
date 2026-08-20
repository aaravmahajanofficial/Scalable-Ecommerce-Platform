package apperrors_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
)

func TestAppError_Error(t *testing.T) {
	err := appErrors.NewAppError(appErrors.ErrCodeValidation, "validation failed", http.StatusBadRequest)
	assert.Equal(t, "validation failed", err.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	err := appErrors.NewAppError(appErrors.ErrCodeInternal, "internal error", http.StatusInternalServerError).WithError(baseErr)

	unwrapped := err.Unwrap()
	assert.Equal(t, baseErr, unwrapped)

	// Test when Err is nil
	errNoBase := appErrors.NewAppError(appErrors.ErrCodeInternal, "internal error", http.StatusInternalServerError)
	assert.Nil(t, errNoBase.Unwrap())
}

func TestNewAppError(t *testing.T) {
	code := "TEST_CODE"
	msg := "test message"
	status := http.StatusTeapot

	err := appErrors.NewAppError(code, msg, status)

	assert.NotNil(t, err)
	assert.Equal(t, code, err.Code)
	assert.Equal(t, msg, err.Message)
	assert.Equal(t, status, err.StatusCode)
	assert.Empty(t, err.Detail)
	assert.Nil(t, err.Err)
}

func TestAppError_WithDetail(t *testing.T) {
	err := appErrors.NewAppError("CODE", "msg", 200)
	detail := "some detail"

	result := err.WithDetail(detail)

	assert.Equal(t, err, result, "WithDetail should return the same pointer")
	assert.Equal(t, detail, err.Detail)
}

func TestAppError_WithError(t *testing.T) {
	err := appErrors.NewAppError("CODE", "msg", 200)
	baseErr := errors.New("inner error")

	result := err.WithError(baseErr)

	assert.Equal(t, err, result, "WithError should return the same pointer")
	assert.Equal(t, baseErr, err.Err)
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name           string
		constructor    func(string) *appErrors.AppError
		message        string
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "ValidationError",
			constructor:    appErrors.ValidationError,
			message:        "val error",
			expectedCode:   appErrors.ErrCodeValidation,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "BadRequestError",
			constructor:    appErrors.BadRequestError,
			message:        "bad req",
			expectedCode:   appErrors.ErrCodeBadRequest,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "NotFoundError",
			constructor:    appErrors.NotFoundError,
			message:        "not found",
			expectedCode:   appErrors.ErrCodeNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "UnauthorizedError",
			constructor:    appErrors.UnauthorizedError,
			message:        "unauth",
			expectedCode:   appErrors.ErrCodeUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "ForbiddenError",
			constructor:    appErrors.ForbiddenError,
			message:        "forbidden",
			expectedCode:   appErrors.ErrCodeForbidden,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "InternalError",
			constructor:    appErrors.InternalError,
			message:        "internal",
			expectedCode:   appErrors.ErrCodeInternal,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "DatabaseError",
			constructor:    appErrors.DatabaseError,
			message:        "db error",
			expectedCode:   appErrors.ErrCodeDatabaseError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "DuplicateEntryError",
			constructor:    appErrors.DuplicateEntryError,
			message:        "dup entry",
			expectedCode:   appErrors.ErrCodeDuplicateEntry,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "ThirdPartyError",
			constructor:    appErrors.ThirdPartyError,
			message:        "3rd party",
			expectedCode:   appErrors.ErrCodeThirdPartyError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "TooManyRequestsError",
			constructor:    appErrors.TooManyRequestsError,
			message:        "rate limit",
			expectedCode:   appErrors.ErrCodeTooManyRequests,
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:           "ResourceExhaustedError",
			constructor:    appErrors.ResourceExhaustedError,
			message:        "exhausted",
			expectedCode:   appErrors.ErrCodeResourceExhausted,
			expectedStatus: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor(tt.message)
			assert.NotNil(t, err)
			assert.Equal(t, tt.expectedCode, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.expectedStatus, err.StatusCode)
		})
	}
}

func TestIsAppError(t *testing.T) {
	t.Run("it is an AppError", func(t *testing.T) {
		appErr := appErrors.NewAppError("C", "M", 200)
		result, ok := appErrors.IsAppError(appErr)
		assert.True(t, ok)
		assert.Equal(t, appErr, result)
	})

	t.Run("wrapped AppError", func(t *testing.T) {
		appErr := appErrors.NewAppError("C", "M", 200)
		wrappedErr := errors.Join(errors.New("wrapper"), appErr)

		result, ok := appErrors.IsAppError(wrappedErr)
		assert.True(t, ok)
		assert.Equal(t, appErr, result)
	})

	t.Run("not an AppError", func(t *testing.T) {
		standardErr := errors.New("standard error")
		result, ok := appErrors.IsAppError(standardErr)
		assert.False(t, ok)
		assert.Nil(t, result)
	})

	t.Run("nil error", func(t *testing.T) {
		result, ok := appErrors.IsAppError(nil)
		assert.False(t, ok)
		assert.Nil(t, result)
	})
}

func TestAddValidationError(t *testing.T) {
	err := appErrors.AddValidationError("email", "must be a valid email address")
	assert.NotNil(t, err)
	assert.Equal(t, appErrors.ErrCodeValidation, err.Code)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, "Invalid field 'email': must be a valid email address", err.Message)
}
