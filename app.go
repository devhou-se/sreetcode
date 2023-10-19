package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
)

func unsreefy(in string) string {
	replace := map[string]string{
		"Sremium": "Premium",
		"sremium": "premium",
		"Sreet":   "Leet",
		"sreet":   "leet",
	}

	for k, v := range replace {
		in = strings.ReplaceAll(in, k, v)
	}

	return in
}

func sreefy(in string) string {
	replace := map[string]string{
		"Leet":    "Sreet",
		" leet":   " sreet",
		"Premium": "Sreemium",
		"premium": "sreemium",
	}

	for k, v := range replace {
		in = strings.ReplaceAll(in, k, v)
	}

	return in
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := chi.NewRouter()

	u, err := url.Parse("https://leetcode.com")
	if err != nil {
		panic(err)
	}

	c := http.Client{}

	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		u2 := r.URL
		u2.Scheme = u.Scheme
		u2.Host = u.Host

		u2, _ = url.Parse(unsreefy(u2.String()))

		req, err := http.NewRequest(r.Method, u2.String(), r.Body)
		if err != nil {
			fmt.Println("err:", err)
		}
		resp, err := c.Do(req)
		if err != nil {
			fmt.Println("err:", err)
		}
		b, _ := io.ReadAll(resp.Body)

		for h, v := range resp.Header {
			for _, vv := range v {
				w.Header().Add(h, vv)
			}
		}

		s := string(b)
		s = sreefy(s)
		b = []byte(s)

		w.WriteHeader(resp.StatusCode)
		w.Write(b)
	})

	server := http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
