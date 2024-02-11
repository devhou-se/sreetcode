package config

import (
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	Port string

	// Insecure is true if the server should use an insecure connection.
	Insecure bool
	// SreeifierServer is the address of the Sreeification gRPC server.
	SreeifierServer string
}

func envOrDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Info(fmt.Sprintf("Using default value for %s: %s", key, def))
		v = def
	}
	return v
}

func Load() Config {
	return Config{
		Port:            envOrDefault("PORT", "8080"),
		Insecure:        envOrDefault("INSECURE", "") != "false",
		SreeifierServer: envOrDefault("SREEIFIER_SERVER", "sreeifier-vvgwyvu7bq-as.a.run.app:443"),
	}
}
