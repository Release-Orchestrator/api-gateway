package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServiceProxy_InvalidURL(t *testing.T) {
	_, err := NewServiceProxy("test", "://invalid-url")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestServiceProxy_ForwardsRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Host") == "" {
			t.Error("expected X-Forwarded-Host header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"message":"ok"}}`))
	}))
	defer backend.Close()

	proxy, err := NewServiceProxy("test-service", backend.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	frontend := httptest.NewServer(proxy)
	defer frontend.Close()

	resp, err := http.Get(frontend.URL + "/api/v1/test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}
	if result["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestServiceProxy_BackendDown(t *testing.T) {
	proxy, err := NewServiceProxy("test-service", "http://localhost:19999")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	frontend := httptest.NewServer(proxy)
	defer frontend.Close()

	resp, err := http.Get(frontend.URL + "/api/v1/test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}
	if result["success"] != false {
		t.Fatal("expected success false")
	}
}

func TestServiceProxy_BackendNotFound(t *testing.T) {
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"NOT_FOUND","message":"not found"}}`))
	}))
	defer frontend.Close()

	proxy, err := NewServiceProxy("test-service", frontend.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	backend := httptest.NewServer(proxy)
	defer backend.Close()

	resp, err := http.Get(backend.URL + "/api/v1/unknown")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestServiceProxy_PreservesQueryParams(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("user_id") == "" {
			t.Error("expected user_id query param")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer backend.Close()

	proxy, err := NewServiceProxy("test-service", backend.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	frontend := httptest.NewServer(proxy)
	defer frontend.Close()

	_, _ = http.Get(frontend.URL + fmt.Sprintf("/api/v1/test?user_id=%s", "123e4567-e89b-12d3-a456-426614174000"))
}

func TestSingleJoiningSlash(t *testing.T) {
	tests := []struct {
		a, b, expected string
	}{
		{"/api/v1", "/users", "/api/v1/users"},
		{"/api/v1/", "/users", "/api/v1/users"},
		{"/api/v1", "users", "/api/v1/users"},
		{"/api/v1/", "users", "/api/v1/users"},
		{"", "/users", "/users"},
		{"/api/v1", "", "/api/v1"},
	}

	for _, tt := range tests {
		result := singleJoiningSlash(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("singleJoiningSlash(%q, %q) = %q, want %q", tt.a, tt.b, result, tt.expected)
		}
	}
}
