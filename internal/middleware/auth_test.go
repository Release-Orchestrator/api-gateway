package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupAuthRouter(v *AuthValidator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	protected := r.Group("/api/v1")
	protected.Use(v.Middleware())
	protected.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetHeader("X-User-Id"), "email": c.GetHeader("X-User-Email")})
	})
	return r
}

func authServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/auth/validate" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func request(r *gin.Engine, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeEnvelope(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	return result
}

func TestMiddleware_ValidToken(t *testing.T) {
	server := authServer(t, http.StatusOK, `{"user_id":"u-123","email":"ada@x.io"}`)
	validator := NewAuthValidator(server.URL)
	r := setupAuthRouter(validator)

	w := request(r, "valid-token")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.UserID != "u-123" || result.Email != "ada@x.io" {
		t.Fatalf("expected forwarded identity, got %s", w.Body.String())
	}
}

func TestMiddleware_MissingToken(t *testing.T) {
	validator := NewAuthValidator("http://localhost:19999")
	r := setupAuthRouter(validator)

	w := request(r, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	result := decodeEnvelope(t, w.Body.Bytes())
	errorObj := result["error"].(map[string]interface{})
	if errorObj["code"] != "TOKEN_MISSING" {
		t.Fatalf("expected TOKEN_MISSING, got %v", errorObj["code"])
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	server := authServer(t, http.StatusUnauthorized, `{"error":"invalid or expired token"}`)
	validator := NewAuthValidator(server.URL)
	r := setupAuthRouter(validator)

	w := request(r, "bad-token")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	result := decodeEnvelope(t, w.Body.Bytes())
	errorObj := result["error"].(map[string]interface{})
	if errorObj["code"] != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %v", errorObj["code"])
	}
}

func TestMiddleware_AuthServiceDown(t *testing.T) {
	validator := NewAuthValidator("http://localhost:19999")
	r := setupAuthRouter(validator)

	w := request(r, "some-token")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	result := decodeEnvelope(t, w.Body.Bytes())
	errorObj := result["error"].(map[string]interface{})
	if errorObj["code"] != "AUTH_SERVICE_UNAVAILABLE" {
		t.Fatalf("expected AUTH_SERVICE_UNAVAILABLE, got %v", errorObj["code"])
	}
}

func TestMiddleware_AuthServiceError(t *testing.T) {
	server := authServer(t, http.StatusInternalServerError, `{"error":"boom"}`)
	validator := NewAuthValidator(server.URL)
	r := setupAuthRouter(validator)

	w := request(r, "some-token")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
