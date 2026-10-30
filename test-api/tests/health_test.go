package tests

import (
	transport "github.com/test/test-api/internal/transport/http"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newRouter() http.Handler {
	return transport.NewRouter()
}

func TestHealthEndpoint(t *testing.T) {
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}
