package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/stretchr/testify/assert"
)

func TestWriteJSON(t *testing.T) {
	t.Run("writes JSON correctly", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		data := map[string]string{"key": "value"}

		err := response.WriteJSON(recorder, http.StatusOK, data)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var responseBody map[string]string
		err = json.Unmarshal(recorder.Body.Bytes(), &responseBody)
		assert.NoError(t, err)
		assert.Equal(t, data, responseBody)
	})
}

func TestSuccess(t *testing.T) {
	t.Run("writes success response correctly", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		data := map[string]string{"message": "success data"}

		response.Success(recorder, http.StatusCreated, data)

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var responseBody response.APIResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
		assert.NoError(t, err)
		assert.True(t, responseBody.Success)

		// data is an any type which gets unmarshaled to map[string]interface{}
		expectedData := map[string]interface{}{"message": "success data"}
		assert.Equal(t, expectedData, responseBody.Data)
	})

	t.Run("writes success response with nil data", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		response.Success(recorder, http.StatusOK, nil)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var responseBody response.APIResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
		assert.NoError(t, err)
		assert.True(t, responseBody.Success)
		assert.Nil(t, responseBody.Data)
	})
}
