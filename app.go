package main

import (
	"github.com/devhou-se/sreetcode/internal/service"
)

func main() {
	s, err := service.NewWebServer()
	if err != nil {
		panic(err)
	}

	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
