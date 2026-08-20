package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		data := map[string]string{"message": "hello world"}

		err := response.WriteJSON(recorder, http.StatusOK, data)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var resp map[string]string
		err = json.Unmarshal(recorder.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, data, resp)
	})

	t.Run("encoding error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		data := make(chan int)

		err := response.WriteJSON(recorder, http.StatusOK, data)
		assert.Error(t, err)
	})
}

func TestSuccess(t *testing.T) {
	recorder := httptest.NewRecorder()
	data := map[string]string{"user": "test"}

	response.Success(recorder, http.StatusCreated, data)
	assert.Equal(t, http.StatusCreated, recorder.Code)

	var resp response.APIResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	respData, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test", respData["user"])
}

func parseErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.APIResponse {
	t.Helper()

	var resp response.APIResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)

	return resp
}

func TestError_StandardError(t *testing.T) {
	recorder := httptest.NewRecorder()
	response.Error(recorder, errors.New("something went wrong"))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	resp := parseErrorResponse(t, recorder)
	assert.Equal(t, apperrors.ErrCodeInternal, resp.Error.Code)
	assert.Equal(t, "An unexpected error occurred", resp.Error.Message)
}

func TestError_AppErrorWithoutDetails(t *testing.T) {
	recorder := httptest.NewRecorder()
	response.Error(recorder, apperrors.NotFoundError("user not found"))

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	resp := parseErrorResponse(t, recorder)
	assert.Equal(t, apperrors.ErrCodeNotFound, resp.Error.Code)
	assert.Equal(t, "user not found", resp.Error.Message)
	assert.Empty(t, resp.Error.Details)
}

func TestError_AppErrorWithDetails(t *testing.T) {
	recorder := httptest.NewRecorder()
	response.Error(recorder, apperrors.ValidationError("invalid input").WithDetail("email is required"))

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	resp := parseErrorResponse(t, recorder)
	assert.Equal(t, apperrors.ErrCodeValidation, resp.Error.Code)
	require.Len(t, resp.Error.Details, 1)
	assert.Equal(t, "email is required", resp.Error.Details[0])
}
