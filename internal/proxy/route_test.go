package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouter_ExactRoutesDoNotRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"path": r.URL.Path})
	}))
	defer backend.Close()

	userProxy, err := NewServiceProxy("user-service", backend.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Mirrors cmd/main.go route registration (without auth middleware).
	r := gin.New()
	api := r.Group("/api/v1")
	{
		api.Any("/users", gin.WrapH(userProxy))
		api.Any("/users/*path", gin.WrapH(userProxy))
	}

	cases := []string{"/api/v1/users", "/api/v1/users/"}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d (location %q)", path, w.Code, w.Header().Get("Location"))
		}

		var result struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("GET %s: invalid JSON: %v", path, err)
		}
		if result.Path != "/api/v1/users" {
			t.Fatalf("GET %s: expected backend path /api/v1/users, got %q", path, result.Path)
		}
	}
}
