package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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

// urlMapper is a function that maps a URL to another URL.
func urlMapper(f func(string) (*url.URL, bool)) func(string) (*url.URL, bool) {
	cache := map[string]*url.URL{}
	invalid := map[string]bool{}
	return func(host string) (*url.URL, bool) {
		if host == "" {
			return nil, false
		}
		_, ok := invalid[host]
		if ok {
			return nil, false
		}
		_, ok = cache[host]
		if !ok {
			u2, ok2 := f(host)
			if !ok2 {
				invalid[host] = true
				return nil, false
			}
			cache[host] = u2
		}
		u2 := cache[host]
		return u2, true
	}
}

// proxyRequest forwards the request to the target URL and writes back the response.
func (s *Server) proxyRequest(client *http.Client, targetURL *url.URL, w http.ResponseWriter, r *http.Request) {
	// Modify request URL.
	proxiedURL, err := url.Parse(util.Unsreefy(r.URL.String()))
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	proxiedURL.Scheme = targetURL.Scheme
	proxiedURL.Host = targetURL.Host

	// Create a new request to the target URL.
	req, err := http.NewRequest(r.Method, proxiedURL.String(), r.Body)
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		log.Printf("Error creating request: %s\n", err)
	}

	// Send the request using the provided HTTP client and get the response.
	start := time.Now()
	resp, err := client.Do(req)
	slog.Info(fmt.Sprintf("Proxy request took %s\n", time.Now().Sub(start)))
	if err != nil {
		http.Error(w, "Error making request", http.StatusInternalServerError)
		log.Printf("Error making request: %s\n", err)
	}
	defer resp.Body.Close()

	// Copy headers from the proxied response to the original response writer.
	for h, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(h, v)
		}
	}

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading response", http.StatusInternalServerError)
		return
	}

	// if mimetype isnt text/html, just return the body
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	modifiedBody, err := s.sreeify.Sreeify(body)
	if err != nil {
		slog.Error(fmt.Sprintf("Error sreeifying response: %s", err))
		http.Error(w, "Error sreeifying response", http.StatusInternalServerError)
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

	router.Use(blockAgents, middlewareFunc, timerFunc)

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
