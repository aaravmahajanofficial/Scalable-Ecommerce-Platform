package utils

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/stretchr/testify/assert"
)

type TestData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func (e *errorReader) Close() error {
	return nil
}

func TestDecodeJSONBody(t *testing.T) {
	t.Run("successful decode", func(t *testing.T) {
		body := []byte(`{"name":"John Doe","age":30}`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		var dest TestData

		err := DecodeJSONBody(req, &dest)

		assert.NoError(t, err)
		assert.Equal(t, "John Doe", dest.Name)
		assert.Equal(t, 30, dest.Age)
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte{}))
		var dest TestData

		err := DecodeJSONBody(req, &dest)

		assert.Error(t, err)
		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeBadRequest, appErr.Code)
		assert.Equal(t, "Request body cannot be empty", appErr.Message)
	})

	t.Run("invalid json", func(t *testing.T) {
		body := []byte(`{"name":"John Doe",age":30}`) // malformed json
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		var dest TestData

		err := DecodeJSONBody(req, &dest)

		assert.Error(t, err)
		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeBadRequest, appErr.Code)
		assert.Equal(t, "Invalid JSON format", appErr.Message)
	})

	t.Run("reader error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Body = &errorReader{} // replace body with error reader
		var dest TestData

		err := DecodeJSONBody(req, &dest)

		assert.Error(t, err)
		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeBadRequest, appErr.Code)
		assert.Equal(t, "Failed to read request body", appErr.Message)
	})
}
