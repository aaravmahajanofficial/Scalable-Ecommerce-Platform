package utils_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPayloadExtended struct {
	Name   string `json:"name" validate:"required,min=2,max=10"`
	Email  string `json:"email" validate:"required,email"`
	Age    int    `json:"age" validate:"gt=17,lt=100"`
	Status string `json:"status" validate:"oneof=active inactive"`
}

type errReader struct{}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func (e *errReader) Close() error {
	return nil
}

func TestDecodeJSONBody(t *testing.T) {
	t.Run("Valid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"John","email":"john@example.com","age":25,"status":"active"}`))
		var dest testPayloadExtended
		err := utils.DecodeJSONBody(req, &dest)
		assert.NoError(t, err)
		assert.Equal(t, "John", dest.Name)
	})

	t.Run("Empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		var dest testPayloadExtended
		err := utils.DecodeJSONBody(req, &dest)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "Request body cannot be empty")
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"John",`))
		var dest testPayloadExtended
		err := utils.DecodeJSONBody(req, &dest)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "Invalid JSON format")
	})

	t.Run("Read error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", &errReader{})
		var dest testPayloadExtended
		err := utils.DecodeJSONBody(req, &dest)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "Failed to read request body")
	})
}

func TestValidateStruct(t *testing.T) {
	val := validator.New()
	ctx := context.Background()

	t.Run("Valid struct", func(t *testing.T) {
		data := testPayloadExtended{Name: "John", Email: "john@example.com", Age: 25, Status: "active"}
		err := utils.ValidateStruct(ctx, val, data)
		assert.NoError(t, err)
	})

	t.Run("Invalid struct", func(t *testing.T) {
		data := testPayloadExtended{Name: "J", Email: "invalid", Age: 10, Status: "pending"}
		err := utils.ValidateStruct(ctx, val, data)
		assert.Error(t, err)

		appErr, ok := apperrors.IsAppError(err)
		require.True(t, ok)
		assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)
		assert.Contains(t, appErr.Detail, "Field Name must be at least 2 characters")
		assert.Contains(t, appErr.Detail, "Field Email must be a valid email address")
		assert.Contains(t, appErr.Detail, "Field Age must be greater than 17")
		assert.Contains(t, appErr.Detail, "Field Status is invalid: oneof=active inactive")
	})

	t.Run("Invalid struct - Max and LT", func(t *testing.T) {
		data := testPayloadExtended{Name: "John Doe Extra Long", Email: "john@example.com", Age: 150, Status: "active"}
		err := utils.ValidateStruct(ctx, val, data)
		assert.Error(t, err)

		appErr, ok := apperrors.IsAppError(err)
		require.True(t, ok)
		assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)
		assert.Contains(t, appErr.Detail, "Field Name must be at most 10 characters")
		assert.Contains(t, appErr.Detail, "Field Age must be less than 100")
	})

	t.Run("Unexpected validation error", func(t *testing.T) {
	    // validator.Struct returns InvalidValidationError if the type is incorrect.
	    err := utils.ValidateStruct(ctx, val, nil)
	    assert.Error(t, err)

	    appErr, ok := apperrors.IsAppError(err)
		require.True(t, ok)
		assert.Equal(t, apperrors.ErrCodeInternal, appErr.Code)
		assert.Contains(t, appErr.Message, "Unexpected validation error")
	})
}

func TestParseAndValidate(t *testing.T) {
	val := validator.New()

	t.Run("Valid payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"John","email":"john@example.com","age":25,"status":"active"}`))
		w := httptest.NewRecorder()
		var dest testPayloadExtended

		ok := utils.ParseAndValidate(req, w, &dest, val)
		assert.True(t, ok)
		assert.Equal(t, "John", dest.Name)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"John",`))
		w := httptest.NewRecorder()
		var dest testPayloadExtended

		ok := utils.ParseAndValidate(req, w, &dest, val)
		assert.False(t, ok)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"J","email":"invalid","age":10,"status":"pending"}`))
		w := httptest.NewRecorder()
		var dest testPayloadExtended

		ok := utils.ParseAndValidate(req, w, &dest, val)
		assert.False(t, ok)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
		assert.Contains(t, w.Body.String(), "Field Name must be at least 2 characters")
	})
}

func TestParseInt(t *testing.T) {
	t.Run("Valid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("id", "123")

		id, err := utils.ParseInt(req, "id")
		assert.NoError(t, err)
		assert.Equal(t, int64(123), id)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("id", "abc")

		id, err := utils.ParseInt(req, "id")
		assert.Error(t, err)
		assert.Equal(t, int64(0), id)
		assert.ErrorContains(t, err, "Invalid id ID")
	})
}

func TestParseID(t *testing.T) {
	t.Run("Valid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		validUUID := uuid.New()
		req.SetPathValue("user_id", validUUID.String())

		id, err := utils.ParseID(req, "user_id")
		assert.NoError(t, err)
		assert.Equal(t, validUUID, id)
	})

	t.Run("Missing UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("user_id", "")

		id, err := utils.ParseID(req, "user_id")
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, id)
		assert.ErrorContains(t, err, "Missing path parameter: user_id")
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("user_id", "invalid-uuid")

		id, err := utils.ParseID(req, "user_id")
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, id)
		assert.ErrorContains(t, err, "Invalid user_id ID format: must be a UUID")
	})
}
