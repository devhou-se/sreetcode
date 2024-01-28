package config

import (
	"os"
)

type Config struct {
	// SreeifierServer is the address of the Sreeification gRPC server.
	SreeifierServer string
}

func envOrDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		v = def
	}
	return v
}

func Load() Config {
	return Config{
		SreeifierServer: envOrDefault("SREEIFIER_SERVER", "host.docker.internal:50051"),
	}
}
