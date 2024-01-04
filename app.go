package main

import (
	"github.com/devhou-se/sreetcode/internal/server"
)

func main() {
	s, err := server.NewWebServer()
	if err != nil {
		panic(err)
	}

	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
