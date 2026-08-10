package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	customerrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestWriteJSON(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"key": "value"}
		err := WriteJSON(w, http.StatusOK, data)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, data, resp)
	})

	t.Run("failure - unencodable data", func(t *testing.T) {
		w := httptest.NewRecorder()
		// Channels cannot be marshaled into JSON
		err := WriteJSON(w, http.StatusOK, make(chan int))

		assert.Error(t, err)
	})
}

func TestSuccess(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"msg": "all good"}

		Success(w, http.StatusOK, data)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Nil(t, resp.Error)

		// Convert interface{} back to map for comparison
		dataMap := resp.Data.(map[string]interface{})
		assert.Equal(t, "all good", dataMap["msg"])
	})
}

func TestError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
		expectedMsg    string
		expectedDetail []string
	}{
		{
			name:           "generic error",
			err:            errors.New("some unexpected error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   customerrors.ErrCodeInternal,
			expectedMsg:    "An unexpected error occurred",
			expectedDetail: nil,
		},
		{
			name:           "app error without detail",
			err:            customerrors.NewAppError("CUSTOM_CODE", "custom message", http.StatusBadRequest),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "CUSTOM_CODE",
			expectedMsg:    "custom message",
			expectedDetail: nil,
		},
		{
			name:           "app error with detail",
			err:            customerrors.NewAppError("CUSTOM_CODE", "custom message", http.StatusBadRequest).WithDetail("some detail"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "CUSTOM_CODE",
			expectedMsg:    "custom message",
			expectedDetail: []string{"some detail"},
		},
		{
			name:           "validation error helper",
			err:            customerrors.ValidationError("invalid input"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   customerrors.ErrCodeValidation,
			expectedMsg:    "invalid input",
			expectedDetail: nil,
		},
		{
			name:           "not found error helper",
			err:            customerrors.NotFoundError("user not found"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   customerrors.ErrCodeNotFound,
			expectedMsg:    "user not found",
			expectedDetail: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			Error(w, tt.err)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var resp APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)

			assert.False(t, resp.Success)
			assert.Nil(t, resp.Data)
			assert.NotNil(t, resp.Error)

			assert.Equal(t, tt.expectedCode, resp.Error.Code)
			assert.Equal(t, tt.expectedMsg, resp.Error.Message)
			assert.Equal(t, tt.expectedDetail, resp.Error.Details)
		})
	}
}
