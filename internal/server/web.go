package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/go-chi/chi/v5"

	"github.com/devhou-se/sreetcode/internal/util"
)

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
func proxyRequest(eligible func(*url.URL) bool, client *http.Client, targetURL *url.URL, w http.ResponseWriter, r *http.Request) {
	// Modify request URL.
	proxiedURL, err := url.Parse(util.Unsreefy(r.URL.String()))
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	if !eligible(proxiedURL) {
		http.Redirect(w, r, proxiedURL.String(), http.StatusPermanentRedirect)
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
	resp, err := client.Do(req)
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

	// Modify the response body text and write it to the original response writer.
	modifiedBody := util.Sreefy(string(body))
	modifiedBody = util.UpdateURLs(modifiedBody)
	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(modifiedBody))
}

// ProxyHandler returns an HTTP handler function that proxies requests.
func ProxyHandler(eligible func(*url.URL) bool, client *http.Client, u *url.URL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(eligible, client, u, w, r)
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

func newRouter(f func(string) (*url.URL, bool)) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middlewareFunc)

	httpClient := &http.Client{}
	for original, replaced := range util.URLMappings {
		originalUrl, _ := url.Parse(original)
		router.Handle(fmt.Sprintf("%s*", replaced), http.StripPrefix(replaced, ProxyHandler(httpClient, originalUrl)))
	}

	router.HandleFunc("/news/*", func(w http.ResponseWriter, r *http.Request) {
		newsUrl, _ := url.Parse("https://en.wikinews.org/")
		http.StripPrefix("/news/", ProxyHandler(httpClient, newsUrl)).ServeHTTP(w, r)
	})

	// Set up routes for specific assets to be replaced.
	mappings := map[string]string{
		"/static/images/mobile/copyright/sreekipedia-wordmark-en.svg": "sreekipedia.org/sreekipedia-wordmark-en.svg",
		"/static/images/mobile/copyright/sreekipedia-tagline-en.svg":  "sreekipedia.org/tagling.svg",
		"/static/favicon/sreekipedia.ico":                             "sreekipedia.org/sreeki.ico",
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
			handler = ProxyHandler(httpClient, u)
			handlers[u.String()] = handler
		}

		handler.ServeHTTP(w, r)
	})

	return router
}

// sreekiMapper is a URL mapper that maps sreekipedia.org URLs to wikipedia.org URLs.
func sreekiMapper(h string) (*url.URL, bool) {
	if h == "" {
		return nil, false
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

// NewWebServer initializes a new web server with predefined routes and handlers.
func NewWebServer(u string) (*http.Server, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := chi.NewRouter()

	router.Use(middlewareFunc)

	httpClient := &http.Client{}

	eligibleFunc := func(u *url.URL) bool {
		if strings.
	}

	for original, replaced := range util.URLMappings {
		originalUrl, _ := url.Parse(original)
		router.Handle(fmt.Sprintf("%s*", replaced), http.StripPrefix(replaced, ProxyHandler(eligibleFunc, httpClient, originalUrl)))
	}

	router.HandleFunc("/news/*", func(w http.ResponseWriter, r *http.Request) {
		newsUrl, _ := url.Parse("https://en.wikinews.org/")
		http.StripPrefix("/news/", ProxyHandler(eligibleFunc, httpClient, newsUrl)).ServeHTTP(w, r)
	})

	// Set up routes for specific assets to be replaced.
	mappings := map[string]string{
		"/static/images/mobile/copyright/sreekipedia-wordmark-en.svg": "sreekipedia.org/sreekipedia-wordmark-en.svg",
		"/static/images/mobile/copyright/sreekipedia-tagline-en.svg":  "sreekipedia.org/tagling.svg",
		"/static/favicon/sreekipedia.ico":                             "sreekipedia.org/sreeki.ico",
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
			handler = ProxyHandler(eligibleFunc, httpClient, u)
			handlers[u.String()] = handler
		}

		handler.ServeHTTP(w, r)
	})

	return router
}

// sreekiMapper is a URL mapper that maps sreekipedia.org URLs to wikipedia.org URLs.
func sreekiMapper(h string) (*url.URL, bool) {
	if h == "" {
		return nil, false
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

// NewWebServer initializes a new web server with predefined routes and handlers.
func NewWebServer() (*http.Server, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	m := urlMapper(sreekiMapper)
	router := newRouter(m)

	// Set up and return the HTTP server.
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	return server, nil
}
