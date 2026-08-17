package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	apperrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
)

func TestWriteJSON(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		data := map[string]string{"message": "hello world"}

		err := WriteJSON(recorder, http.StatusOK, data)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, recorder.Code)
		}

		if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected content type application/json, got %s", contentType)
		}

		var resp map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if !reflect.DeepEqual(data, resp) {
			t.Errorf("expected body %v, got %v", data, resp)
		}
	})

	t.Run("encoding error", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		// Use a channel, which cannot be marshaled to JSON
		data := make(chan int)

		err := WriteJSON(recorder, http.StatusOK, data)
		if err == nil {
			t.Error("expected error due to unmarshalable type, got nil")
		}
	})
}

func TestSuccess(t *testing.T) {
	recorder := httptest.NewRecorder()
	data := map[string]string{"user": "test"}

	Success(recorder, http.StatusCreated, data)

	if recorder.Code != http.StatusCreated {
		t.Errorf("expected status code %d, got %d", http.StatusCreated, recorder.Code)
	}

	var resp APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}

	// Convert map to interface{} for comparison
	respData, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", resp.Data)
	}

	if respData["user"] != "test" {
		t.Errorf("expected data user to be test, got %v", respData["user"])
	}
}

func TestError(t *testing.T) {
	t.Run("standard error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		stdErr := errors.New("something went wrong")

		Error(recorder, stdErr)

		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, recorder.Code)
		}

		var resp APIResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if resp.Success {
			t.Error("expected success to be false")
		}

		if resp.Error == nil {
			t.Fatal("expected error to be present")
		}

		if resp.Error.Code != apperrors.ErrCodeInternal {
			t.Errorf("expected error code %s, got %s", apperrors.ErrCodeInternal, resp.Error.Code)
		}

		if resp.Error.Message != "An unexpected error occurred" {
			t.Errorf("expected error message 'An unexpected error occurred', got '%s'", resp.Error.Message)
		}
	})

	t.Run("app error without details", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		appErr := apperrors.NotFoundError("user not found")

		Error(recorder, appErr)

		if recorder.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, recorder.Code)
		}

		var resp APIResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if resp.Success {
			t.Error("expected success to be false")
		}

		if resp.Error == nil {
			t.Fatal("expected error to be present")
		}

		if resp.Error.Code != apperrors.ErrCodeNotFound {
			t.Errorf("expected error code %s, got %s", apperrors.ErrCodeNotFound, resp.Error.Code)
		}

		if resp.Error.Message != "user not found" {
			t.Errorf("expected error message 'user not found', got '%s'", resp.Error.Message)
		}

		if len(resp.Error.Details) != 0 {
			t.Errorf("expected 0 details, got %d", len(resp.Error.Details))
		}
	})

	t.Run("app error with details", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		appErr := apperrors.ValidationError("invalid input").WithDetail("email is required")

		Error(recorder, appErr)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d", http.StatusBadRequest, recorder.Code)
		}

		var resp APIResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if resp.Success {
			t.Error("expected success to be false")
		}

		if resp.Error == nil {
			t.Fatal("expected error to be present")
		}

		if resp.Error.Code != apperrors.ErrCodeValidation {
			t.Errorf("expected error code %s, got %s", apperrors.ErrCodeValidation, resp.Error.Code)
		}

		if len(resp.Error.Details) != 1 {
			t.Fatalf("expected 1 detail, got %d", len(resp.Error.Details))
		}

		if resp.Error.Details[0] != "email is required" {
			t.Errorf("expected detail 'email is required', got '%s'", resp.Error.Details[0])
		}
	})
}
