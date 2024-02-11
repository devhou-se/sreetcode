package service

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	disallowedUserAgents = []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	}
)

// sreekiMapper is a URL mapper that maps sreekipedia.org URLs to wikipedia.org URLs.
func sreekiMapper(h string) (*url.URL, error) {
	slog.Info(fmt.Sprintf("sreekiMapper: %s", h))

	if h == "" {
		return nil, fmt.Errorf("empty host")
	}

	if strings.HasPrefix(h, "localhost") {
		h = "en.sreekipedia.org"
	}

	d := fmt.Sprintf("https://%s", h)
	parts := strings.Split(d, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid host")
	}

	np := len(parts)
	if !(parts[np-2] == "sreekipedia" && parts[np-1] == "org") {
		return nil, fmt.Errorf("invalid host")
	}

	parts[np-2] = "wikipedia"

	up := strings.Join(parts, ".")
	u2, err := url.Parse(up)
	if err != nil {
		return nil, err
	}

	return u2, nil
}

// blockAgents is a middleware function that blocks requests from disallowed user agents.
func blockAgents(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, ua := range disallowedUserAgents {
			if r.UserAgent() == ua {
				slog.Info(fmt.Sprintf("Disallowed user agent with request: %s %s", r.Method, r.URL))
				slog.Warn(fmt.Sprintf("Blocked request from disallowed user agent: %s", r.UserAgent()))
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// middlewareFunc is a middleware function that logs the request.
func middlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(fmt.Sprintf("%s %s", r.Method, r.URL))
		next.ServeHTTP(w, r)
	})
}

// timerFunc is a middleware function that times processing of the request.
func timerFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info(fmt.Sprintf("Request took %s", time.Now().Sub(start)))
	})
}
