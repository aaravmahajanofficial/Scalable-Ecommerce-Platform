package utils

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testData struct {
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
		var dest testData

		err := DecodeJSONBody(req, &dest)

		assert.NoError(t, err)
		assert.Equal(t, "John Doe", dest.Name)
		assert.Equal(t, 30, dest.Age)
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte{}))
		var dest testData

		err := DecodeJSONBody(req, &dest)

		assert.ErrorContains(t, err, "Request body cannot be empty")
	})

	t.Run("invalid json", func(t *testing.T) {
		body := []byte(`{"name":"John Doe",age":30}`) // malformed json
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		var dest testData

		err := DecodeJSONBody(req, &dest)

		assert.ErrorContains(t, err, "Invalid JSON format")
	})

	t.Run("reader error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Body = &errorReader{} // replace body with error reader
		var dest testData

		err := DecodeJSONBody(req, &dest)

		assert.ErrorContains(t, err, "Failed to read request body")
	})
}
