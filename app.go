package main

import (
	"github.com/devhou-se/sreetcode/internal/config"
	"github.com/devhou-se/sreetcode/internal/service"
)

func main() {
	cfg := config.Load()

	s, err := service.NewWebServer(cfg)
	if err != nil {
		panic(err)
	}

	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
