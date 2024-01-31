package main

import (
	"log/slog"

	"github.com/devhou-se/sreetcode/internal/config"
	"github.com/devhou-se/sreetcode/internal/service"
)

func main() {
	cfg := config.Load()

	s, err := service.NewWebServer(cfg)
	if err != nil {
		panic(err)
	}

	slog.Info("Starting server")
	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
