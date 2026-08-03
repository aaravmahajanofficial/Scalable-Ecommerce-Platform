package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/stretchr/testify/assert"
)

// errorResponseWriter mocks a response writer that fails on Write.
type errorResponseWriter struct {
	http.ResponseWriter
}

func (e *errorResponseWriter) Write(b []byte) (int, error) {
	return 0, errors.New("mock write error")
}

func TestSuccess(t *testing.T) {
	t.Run("Success response", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"key": "value"}

		Success(w, http.StatusOK, data)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp APIResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Nil(t, resp.Error)
		assert.Equal(t, map[string]interface{}{"key": "value"}, resp.Data)
	})

	t.Run("WriteJSON fails (unmarshalable data)", func(t *testing.T) {
		w := httptest.NewRecorder()
		// channel cannot be JSON marshaled, causing Encode to fail
		unmarshalableData := make(chan int)

		// This should log an error but not panic
		Success(w, http.StatusOK, unmarshalableData)
	})
}

func TestError(t *testing.T) {
	t.Run("AppError without details", func(t *testing.T) {
		w := httptest.NewRecorder()
		appErr := appErrors.NotFoundError("resource not found")

		Error(w, appErr)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp APIResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)

		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, appErrors.ErrCodeNotFound, resp.Error.Code)
		assert.Equal(t, "resource not found", resp.Error.Message)
		assert.Empty(t, resp.Error.Details)
	})

	t.Run("AppError with details", func(t *testing.T) {
		w := httptest.NewRecorder()
		appErr := appErrors.ValidationError("invalid input").WithDetail("field 'id' is required")

		Error(w, appErr)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp APIResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)

		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, appErrors.ErrCodeValidation, resp.Error.Code)
		assert.Equal(t, "invalid input", resp.Error.Message)
		assert.Len(t, resp.Error.Details, 1)
		assert.Equal(t, "field 'id' is required", resp.Error.Details[0])
	})

	t.Run("Generic standard error", func(t *testing.T) {
		w := httptest.NewRecorder()
		genericErr := errors.New("something went wrong")

		Error(w, genericErr)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp APIResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)

		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, appErrors.ErrCodeInternal, resp.Error.Code)
		assert.Equal(t, "An unexpected error occurred", resp.Error.Message)
		assert.Empty(t, resp.Error.Details)
	})

	t.Run("WriteJSON fails for Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		errWriter := &errorResponseWriter{w}
		genericErr := errors.New("something went wrong")

		// This should log an error but not panic
		Error(errWriter, genericErr)
	})
}
