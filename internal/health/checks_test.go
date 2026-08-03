package health_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestNewLivenessHandler(t *testing.T) {
	handler := health.NewLivenessHandler()

	req := httptest.NewRequest(http.MethodGet, "/liveness", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
	assert.True(t, strings.HasPrefix(rr.Body.String(), "Service is alive. Time: "))
}
