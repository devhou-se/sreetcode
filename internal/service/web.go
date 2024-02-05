package service

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

func (s *Server) router(cfg config.Config) (*chi.Mux, error) {
	r := chi.NewRouter()
	r.Use(middlewareFunc, timerFunc)

	// Set up routes for specific assets to be replaced.
	for requestedAsset, replacementAsset := range util.StaticFileOverrides {
		r.HandleFunc(requestedAsset, assetOverrideHandler(replacementAsset))
	}

	handlers := map[string]http.HandlerFunc{}
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		u, err := sreekiMapper(r.Host)
		if err != nil {
			slog.Error("Error mapping URL")
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		handler, ok := handlers[u.String()]
		if !ok {
			handler = s.proxyHandler(u)
			handlers[u.String()] = handler
		}

		handler.ServeHTTP(w, r)
	})

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

func (s *Server) proxyHandler(u *url.URL) http.HandlerFunc {
	client := &http.Client{}

	handle := func(w http.ResponseWriter, r *http.Request) {
		u2 := r.URL
		u2.Scheme = u.Scheme
		u2.Host = u.Host

		req, err := http.NewRequest(r.Method, u2.String(), r.Body)
		if err != nil {
			http.Error(w, "Error creating request", http.StatusInternalServerError)
			slog.Error("Error creating request: %s", err)
			return
		}

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
	return handle
}
