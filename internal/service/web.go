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
	r.Use(blockAgents, middlewareFunc, timerFunc)

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
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		slog.Error("Error creating request: %s", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error making request", http.StatusInternalServerError)
		slog.Error("Error making request: %s", err)
		return
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

// ProxyHandler returns an HTTP handler function that proxies requests.
func (s *Server) ProxyHandler(client *http.Client, u *url.URL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.proxyRequest(client, u, w, r)
	}
}

// ReplacedAssetHandler handles serving replaced assets from a specified location.
func ReplacedAssetHandler(assetLocation string) http.HandlerFunc {
	ctx := context.Background()

	// Create a new Cloud Storage client.
	sc, err := storage.NewClient(ctx)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Get the object from the bucket.
		o := sc.Bucket("static.xbd.au").Object(assetLocation)

		// Read the object's content.
		rc, err := o.NewReader(ctx)
		if err != nil {
			http.Error(w, "Error reading object", http.StatusInternalServerError)
			return
		}
		defer rc.Close()

		// Get the object's attributes and set the appropriate headers.
		a, err := o.Attrs(ctx)
		if err != nil {
			http.Error(w, "Error getting object attributes", http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", a.ContentType)
		w.Header().Add("Cache-Control", "public, max-age=86400")

		// Copy the object's content to the response writer.
		if _, err := io.Copy(w, rc); err != nil {
			http.Error(w, "Error writing response", http.StatusInternalServerError)
		}
	}
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

	if strings.HasPrefix(h, "localhost") {
		h = "en.sreekipedia.org"
	}

	d := fmt.Sprintf("https://%s", h)
	parts := strings.Split(d, ".")
	if len(parts) < 2 {
		return nil, false
	}

	np := len(parts)
	if !(parts[np-2] == "sreekipedia" && parts[np-1] == "org") {
		return nil, false
	}

	parts[np-2] = "wikipedia"

	up := strings.Join(parts, ".")
	u2, err := url.Parse(up)
	if err != nil {
		return nil, false
	}

	return u2, true
}

func (s *Server) newRouter(f func(string) (*url.URL, bool)) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middlewareFunc, timerFunc)

	httpClient := &http.Client{}

	for original, replaced := range util.URLMappings {
		originalUrl, _ := url.Parse(original)
		router.Handle(fmt.Sprintf("%s*", replaced), http.StripPrefix(replaced, s.ProxyHandler(httpClient, originalUrl)))
	}

	router.HandleFunc("/news/*", func(w http.ResponseWriter, r *http.Request) {
		newsUrl, _ := url.Parse("https://en.wikinews.org/")
		http.StripPrefix("/news/", s.ProxyHandler(httpClient, newsUrl)).ServeHTTP(w, r)
	})

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

	handlers := map[string]http.HandlerFunc{}

	// Handle all other requests with the proxy.
	router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		u, ok := f(r.Host)
		if !ok {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		handler, ok := handlers[u.String()]
		if !ok {
			handler = s.ProxyHandler(httpClient, u)
			handlers[u.String()] = handler
		}

		handler.ServeHTTP(w, r)
	})

	return router
}

// NewWebServer initializes a new web server with predefined routes and handlers.
func NewWebServer(cfg config.Config) (*Server, error) {
	s := &Server{
		Server: &http.Server{},
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	m := urlMapper(sreekiMapper)
	router := s.newRouter(m)

	s.Server.Addr = ":" + port
	s.Server.Handler = router

	sreeify, err := sreeify.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	s.sreeify = sreeify

	return s, nil
}
