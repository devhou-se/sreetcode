package config

import (
	"os"
)

type Config struct {
	// Insecure is true if the server should use an insecure connection.
	Insecure bool
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
		Insecure:        envOrDefault("INSECURE", "") != "",
		SreeifierServer: envOrDefault("SREEIFIER_SERVER", "sreeifier-vvgwyvu7bq-as.a.run.app:443"),
	}
}
