package service

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
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

// middlewareFunc is a middleware function that logs the request.
func middlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(fmt.Sprintf("%s %s\n", r.Method, r.URL))
		next.ServeHTTP(w, r)
	})
}

// timerFunc is a middleware function that times processing of the request.
func timerFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info(fmt.Sprintf("Request took %s\n", time.Now().Sub(start)))
	})
}
