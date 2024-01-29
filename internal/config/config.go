package config

import (
	"os"
)

type Config struct {
	// SreeifierServer is the address of the Sreeification gRPC server.
	SreeifierServer string
}

func Load() Config {
	return Config{
		SreeifierServer: os.Getenv("SREEIFIER_SERVER"),
	}
}
