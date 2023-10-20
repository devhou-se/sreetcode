package server

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"cloud.google.com/go/storage"
	"github.com/go-chi/chi/v5"

	"github.com/devhou-se/sreetcode/internal/util"
)

// proxyRequest forwards the request to the target URL and writes back the response.
func proxyRequest(client *http.Client, targetURL *url.URL, w http.ResponseWriter, r *http.Request) {
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
		return
	}

	// Send the request using the provided HTTP client and get the response.
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error making request", http.StatusInternalServerError)
		return
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
	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(modifiedBody))
}

// ProxyHandler returns an HTTP handler function that proxies requests.
func ProxyHandler(client *http.Client, u *url.URL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(client, u, w, r)
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

// NewWebServer initializes a new web server with predefined routes and handlers.
func NewWebServer(u string) (*http.Server, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := chi.NewRouter()

	router.Use(middlewareFunc)

	// Define the target URL for the proxy.
	targetURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{}

	// Set up routes for specific assets to be replaced.
	mappings := map[string]string{
		// ... existing mappings ...
	}
	for requestedAsset, replacementAsset := range mappings {
		router.HandleFunc(requestedAsset, ReplacedAssetHandler(replacementAsset))
	}

	// Handle all other requests with the proxy.
	router.HandleFunc("/*", ProxyHandler(httpClient, targetURL))

	// Set up and return the HTTP server.
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	return server, nil
}

// StartServer starts the server and log any error if occurs.
func StartServer(server *http.Server) {
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}
}
