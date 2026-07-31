// Package middleware provides HTTP middleware for the API gateway.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Errors returned by the auth validator.
var (
	// ErrInvalidToken is returned when the auth service rejects a token.
	ErrInvalidToken = errors.New("invalid token")
	// ErrAuthServiceUnavailable is returned when the auth service cannot be reached.
	ErrAuthServiceUnavailable = errors.New("auth service unavailable")
)

// AuthValidator validates bearer tokens against the auth service.
type AuthValidator struct {
	baseURL string
	http    *http.Client
}

// NewAuthValidator creates an AuthValidator for the given auth service URL.
func NewAuthValidator(baseURL string) *AuthValidator {
	return &AuthValidator{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Validate forwards a token to the auth service and returns the user identity.
func (v *AuthValidator) Validate(ctx context.Context, token string) (userID, email string, err error) {
	url := v.baseURL + "/internal/auth/validate"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := v.http.Do(req)
	if err != nil {
		return "", "", ErrAuthServiceUnavailable
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("warning: failed to close auth validate response body: %v", cerr)
		}
	}()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", "", ErrInvalidToken
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", ErrAuthServiceUnavailable
	}

	var result struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", ErrAuthServiceUnavailable
	}
	return result.UserID, result.Email, nil
}

// Middleware returns a gin handler that authenticates requests with a bearer token.
// It aborts with 401 when the token is missing or invalid, and 503 when the auth
// service is unreachable. On success it forwards X-User-Id and X-User-Email.
func (v *AuthValidator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "TOKEN_MISSING", "message": "missing authorization token"},
			})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		userID, email, err := v.Validate(c.Request.Context(), token)
		if errors.Is(err, ErrInvalidToken) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "invalid or expired token"},
			})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   gin.H{"code": "AUTH_SERVICE_UNAVAILABLE", "message": "authentication service unavailable"},
			})
			return
		}

		c.Request.Header.Set("X-User-Id", userID)
		c.Request.Header.Set("X-User-Email", email)
		c.Next()
	}
}
