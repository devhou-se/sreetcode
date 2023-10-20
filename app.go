package main

import (
	"github.com/devhou-se/sreetcode/internal/server"
)

func main() {
	u := "https://en.wikipedia.org"

	s, err := server.NewWebServer(u)
	if err != nil {
		panic(err)
	}

	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
