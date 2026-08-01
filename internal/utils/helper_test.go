package utils

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestUser struct {
	Name  string `json:"name" validate:"required"`
	Age   int    `json:"age" validate:"gt=18"`
	Email string `json:"email" validate:"required,email"`
}

type FormatTestStruct struct {
	MinField string `validate:"min=5"`
	MaxField string `validate:"max=5"`
	LtField  int    `validate:"lt=5"`
	DefField string `validate:"len=5"`
}

func TestDecodeJSONBody(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		body := []byte(`{"name":"John","age":30,"email":"john@example.com"}`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

		var user TestUser
		err := DecodeJSONBody(req, &user)
		assert.NoError(t, err)
		assert.Equal(t, "John", user.Name)
		assert.Equal(t, 30, user.Age)
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte{}))

		var user TestUser
		err := DecodeJSONBody(req, &user)
		assert.Error(t, err)
		appErr, ok := errors.IsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "Request body cannot be empty", appErr.Message)
	})

	t.Run("invalid json", func(t *testing.T) {
		body := []byte(`{"name":"John",`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

		var user TestUser
		err := DecodeJSONBody(req, &user)
		assert.Error(t, err)
		appErr, ok := errors.IsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "Invalid JSON format", appErr.Message)
	})
}

func TestValidateStruct(t *testing.T) {
	validate := validator.New()

	t.Run("success", func(t *testing.T) {
		user := TestUser{
			Name:  "John",
			Age:   20,
			Email: "john@example.com",
		}
		err := ValidateStruct(context.Background(), validate, user)
		assert.NoError(t, err)
	})

	t.Run("validation failed", func(t *testing.T) {
		user := TestUser{
			Name:  "",
			Age:   10,
			Email: "invalid",
		}
		err := ValidateStruct(context.Background(), validate, user)
		assert.Error(t, err)
		appErr, ok := errors.IsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "Validation Failed", appErr.Message)
		assert.Contains(t, appErr.Detail, "Field Name is required")
		assert.Contains(t, appErr.Detail, "Field Age must be greater than 18")
		assert.Contains(t, appErr.Detail, "Field Email must be a valid email address")
	})
}

func TestParseAndValidate(t *testing.T) {
	validate := validator.New()

	t.Run("success", func(t *testing.T) {
		body := []byte(`{"name":"John","age":30,"email":"john@example.com"}`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		var user TestUser
		ok := ParseAndValidate(req, rec, &user, validate)
		assert.True(t, ok)
		assert.Equal(t, "John", user.Name)
	})

	t.Run("decode error", func(t *testing.T) {
		body := []byte(`{"name":"John",`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		var user TestUser
		ok := ParseAndValidate(req, rec, &user, validate)
		assert.False(t, ok)
		assert.NotEqual(t, http.StatusOK, rec.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		body := []byte(`{"name":"","age":10,"email":"invalid"}`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		var user TestUser
		ok := ParseAndValidate(req, rec, &user, validate)
		assert.False(t, ok)
		assert.NotEqual(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Validation Failed")
	})
}

func TestFormatValidationError(t *testing.T) {
	validate := validator.New()
	s := FormatTestStruct{
		MinField: "a",
		MaxField: "abcdef",
		LtField:  10,
		DefField: "a",
	}

	err := validate.Struct(s)
	require.Error(t, err)

	verrs := err.(validator.ValidationErrors)
	for _, verr := range verrs {
		msg := formatValidationError(verr)
		switch verr.Field() {
		case "MinField":
			assert.Equal(t, "Field MinField must be at least 5 characters", msg)
		case "MaxField":
			assert.Equal(t, "Field MaxField must be at most 5 characters", msg)
		case "LtField":
			assert.Equal(t, "Field LtField must be less than 5", msg)
		case "DefField":
			assert.Equal(t, "Field DefField is invalid: len=5", msg)
		}
	}
}

func TestParseInt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("id", "123")

		val, err := ParseInt(req, "id")
		assert.NoError(t, err)
		assert.Equal(t, int64(123), val)
	})

	t.Run("invalid int", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("id", "abc")

		_, err := ParseInt(req, "id")
		assert.Error(t, err)
		appErr, ok := errors.IsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "Invalid id ID", appErr.Message)
	})
}

func TestParseID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		uid := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("id", uid.String())

		val, err := ParseID(req, "id")
		assert.NoError(t, err)
		assert.Equal(t, uid, val)
	})

	t.Run("missing param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		_, err := ParseID(req, "id")
		assert.Error(t, err)
		appErr, ok := errors.IsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "Missing path parameter: id", appErr.Message)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("id", "123")

		_, err := ParseID(req, "id")
		assert.Error(t, err)
		appErr, ok := errors.IsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "Invalid id ID format: must be a UUID", appErr.Message)
	})
}
