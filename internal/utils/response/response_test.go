package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/stretchr/testify/assert"
)

func TestSuccess(t *testing.T) {
	t.Run("success response with data", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		data := map[string]string{"key": "value"}
		expectedStatusCode := http.StatusOK

		response.Success(recorder, expectedStatusCode, data)

		assert.Equal(t, expectedStatusCode, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var resp response.APIResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)

		// Type assertion for the data field, which is generic (any/interface{})
		dataMap, ok := resp.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value", dataMap["key"])
	})

	t.Run("success response with nil data", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		expectedStatusCode := http.StatusCreated

		response.Success(recorder, expectedStatusCode, nil)

		assert.Equal(t, expectedStatusCode, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		var resp response.APIResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Nil(t, resp.Data)
	})

	t.Run("success response with unmarshalable data triggers log error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		expectedStatusCode := http.StatusOK

		// channels cannot be json marshaled, so it should trigger WriteJSON error
		unmarshalableData := make(chan int)

		// Capture the standard log output or test it indirectly (the response status code will still be written, but the json body might fail)
		response.Success(recorder, expectedStatusCode, unmarshalableData)

		assert.Equal(t, expectedStatusCode, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	})
}
