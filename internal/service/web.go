package service

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/go-chi/chi/v5"

	"github.com/devhou-se/sreetcode/internal/config"
	"github.com/devhou-se/sreetcode/internal/integration/sreeify"
	"github.com/devhou-se/sreetcode/internal/util"
)

var (
	disallowedUserAgents = []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	}
)

type Server struct {
	*http.Server
	sreeify *sreeify.Client
}

// NewWebServer creates a new web server.
func NewWebServer(cfg config.Config) (*Server, error) {
	s := &Server{}
	var err error

	s.Server, err = s.httpServer(cfg)
	if err != nil {
		return nil, err
	}

	s.sreeify, err = sreeify.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// httpServer creates a new HTTP server with router
func (s *Server) httpServer(cfg config.Config) (*http.Server, error) {
	hs := &http.Server{}
	var err error

	hs.Addr = ":" + cfg.Port
	hs.Handler, err = s.router(cfg)
	if err != nil {
		return nil, err
	}

	return hs, nil
}

// router creates a new router with middleware and routes
func (s *Server) router(cfg config.Config) (*chi.Mux, error) {
	r := chi.NewRouter()
	r.Use(middlewareFunc, timerFunc)

	// Set up routes for specific assets to be replaced.
	for requestedAsset, replacementAsset := range util.StaticFileOverrides {
		r.HandleFunc(requestedAsset, assetOverrideHandler(replacementAsset))
	}

	r.HandleFunc("/*", s.proxyHandler)

	return r, nil
}

// ReplacedAssetHandler handles serving replaced assets from a specified location.
func assetOverrideHandler(assetLocation string) http.HandlerFunc {
	ctx := context.Background()

	// Create a new Cloud Storage client.
	sc, err := storage.NewClient(ctx)
	if err != nil {
		panic(err)
	}
	bucket := sc.Bucket("static.xbd.au")

	return func(w http.ResponseWriter, r *http.Request) {
		obj := bucket.Object(assetLocation)
		reader, err := obj.NewReader(ctx)
		if err != nil {
			http.Error(w, "Error reading object", http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		w.Header().Set("Content-Type", reader.Attrs.ContentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")

		if _, err := io.Copy(w, reader); err != nil {
			http.Error(w, "Error writing response", http.StatusInternalServerError)
		}
	}
}

// proxyHandler is a handler that proxies requests to the appropriate URL.
func (s *Server) proxyHandler(w http.ResponseWriter, r *http.Request) {
	u, err := sreekiMapper(r.Host)
	if err != nil {
		slog.Error("Error mapping URL")
		http.Error(w, "Not found", http.StatusNotFound)
		return

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
		// temp
		slog.Info(fmt.Sprintf("allowed user agent: %s", r.UserAgent()))
		next.ServeHTTP(w, r)
	})
}

// middlewareFunc is a middleware function that logs the request.
func middlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s\n", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

// timerFunc is a middleware function that times processing of the request.
func timerFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("Request took %s\n", time.Now().Sub(start))
	})
}

// sreekiMapper is a URL mapper that maps sreekipedia.org URLs to wikipedia.org URLs.
func sreekiMapper(h string) (*url.URL, bool) {
	slog.Info(fmt.Sprintf("sreekiMapper: %s", h))

	if h == "" {
		return nil, false
	}

	u2 := r.URL
	u2.Scheme = u.Scheme
	u2.Host = u.Host

	if strings.HasPrefix(u2.Path, "/wiki/") {
		u2.Path = strings.Replace(u2.Path, "/wiki/", "/sreeki/", 1)
		http.Redirect(w, r, u2.Path, http.StatusTemporaryRedirect)
	}

	if strings.HasPrefix(u2.Path, "/sreeki/") {
		u2.Path = strings.Replace(u2.Path, "/sreeki/", "/wiki/", 1)
	}

	req, err := http.NewRequest(r.Method, u2.String(), r.Body)
	if err != nil {
		return nil, false
	}

	return u2, true
}

func (s *Server) newRouter(f func(string) (*url.URL, bool)) *chi.Mux {
	router := chi.NewRouter()

	router.Use(blockAgents, middlewareFunc, timerFunc)

	httpClient := &http.Client{}

	for original, replaced := range util.URLMappings {
		originalUrl, _ := url.Parse(original)
		router.Handle(fmt.Sprintf("%s*", replaced), http.StripPrefix(replaced, s.ProxyHandler(httpClient, originalUrl)))
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error making request", http.StatusInternalServerError)
		slog.Error("Error making request: %s", err)
		return
	// Set up routes for specific assets to be replaced.
	mappings := map[string]string{
		"/static/images/mobile/copyright/sreekipedia-wordmark-en.svg": "sreekipedia.org/sreekipedia-wordmark-en.svg",
		"/static/images/mobile/copyright/sreekipedia-tagline-en.svg":  "sreekipedia.org/tagling.svg",
		"/static/favicon/sreekipedia.ico":                             "sreekipedia.org/sreeki.ico",
		"/robots.txt":                                                 "sreekipedia.org/robots.txt",
	}
	for requestedAsset, replacementAsset := range mappings {
		router.HandleFunc(requestedAsset, ReplacedAssetHandler(replacementAsset))
	}
	defer resp.Body.Close()

	for h, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(h, v)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading response", http.StatusInternalServerError)
		slog.Error("Error reading response: %s", err)
		return
	}

	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	modifiedBody, err := s.sreeify.Sreeify(body)
	if err != nil {
		http.Error(w, "Error sreeifying response", http.StatusInternalServerError)
		slog.Error("Error sreeifying response: %s", err)
		return
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(modifiedBody)
}
