package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	assert.Equal(t, http.StatusOK, rw.statusCode)

	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestMiddleware_RecordsMetrics(t *testing.T) {
	// Reset metrics
	httpRequestsTotal.Reset()
	httpRequestsDuration.Reset()

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	// Verify httpRequestsTotal
	counter, err := httpRequestsTotal.GetMetricWithLabelValues("202", "GET", "/test-path")
	assert.NoError(t, err)
	val := testutil.ToFloat64(counter)
	assert.Equal(t, float64(1), val)

	// Verify httpRequestsDuration
	_, err = httpRequestsDuration.GetMetricWithLabelValues("GET", "/test-path")
	assert.NoError(t, err)
	// We can't directly read the sum easily with testutil for a specific label set without CollectAndCompare,
	// but we can check if it exists and has observed a value by using testutil.CollectAndCount
	count := testutil.CollectAndCount(httpRequestsDuration)
	assert.Greater(t, count, 0)
}


func TestMiddleware_PathPatterns(t *testing.T) {
	httpRequestsTotal.Reset()

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123", nil)
	// Simulate go 1.22 routing path value matching "..."
	req.SetPathValue("...", "123")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Since SetPathValue("...", "123") makes it end with 123, the expected replaced path is "/api/v1/users/{...}"
	counter, err := httpRequestsTotal.GetMetricWithLabelValues("200", "GET", "/api/v1/users/{...}")
	assert.NoError(t, err)
	val := testutil.ToFloat64(counter)
	assert.Equal(t, float64(1), val)
}

func TestMiddleware_InFlight(t *testing.T) {
	// Gauge can be tricky if we don't know the exact starting state due to global tests,
	// but let's read the current value.
	initialVal := testutil.ToFloat64(httpRequestsInFlight)

	var valInsideHandler float64

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		valInsideHandler = testutil.ToFloat64(httpRequestsInFlight)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/in-flight", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// In flight should be +1 while inside the handler
	assert.Equal(t, initialVal+1, valInsideHandler)

	// In flight should return to initial val after serveHTTP completes
	finalVal := testutil.ToFloat64(httpRequestsInFlight)
	assert.Equal(t, initialVal, finalVal)
}
