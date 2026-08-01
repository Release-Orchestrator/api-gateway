// Package proxy provides HTTP reverse proxy utilities for forwarding requests to backend services.
package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// ServiceProxy wraps a reverse proxy for a single backend service.
type ServiceProxy struct {
	name   string
	target *url.URL
	proxy  *httputil.ReverseProxy
}

// NewServiceProxy creates a new ServiceProxy for the given service name and base URL.
func NewServiceProxy(name, baseURL string) (*ServiceProxy, error) {
	target, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL %q: %w", baseURL, err)
	}

	p := httputil.NewSingleHostReverseProxy(target)

	// Preserve original Host header
	p.Director = func(req *http.Request) {
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = normalizeTrailingSlash(singleJoiningSlash(target.Path, req.URL.Path))
		if req.URL.RawPath != "" {
			req.URL.RawPath = normalizeTrailingSlash(singleJoiningSlash(target.Path, req.URL.RawPath))
		}
	}

	// Error handler for proxy errors
	p.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		log.Printf("proxy error for %s: %v", name, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"` + strings.ToUpper(name) + `_SERVICE_UNAVAILABLE","message":"service unavailable"}}`))
	}

	// Modify response to add logging
	p.ModifyResponse = func(resp *http.Response) error {
		log.Printf("%s %s -> %s %d", resp.Request.Method, resp.Request.URL.Path, name, resp.StatusCode)
		return nil
	}

	return &ServiceProxy{
		name:   name,
		target: target,
		proxy:  p,
	}, nil
}

// ServeHTTP implements http.Handler by forwarding the request to the backend service.
func (s *ServiceProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	s.proxy.ServeHTTP(w, r.WithContext(ctx))
}

// singleJoiningSlash joins path segments with a single slash.
func singleJoiningSlash(a, b string) string {
	if b == "" {
		return a
	}
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// normalizeTrailingSlash removes a trailing slash so backends that register
// exact routes do not redirect and create an infinite redirect loop.
func normalizeTrailingSlash(p string) string {
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		return strings.TrimSuffix(p, "/")
	}
	return p
}
