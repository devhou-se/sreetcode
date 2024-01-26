package config

type Config struct {
	// SreeificationServer is the address of the Sreeification gRPC server.
	SreeificationServer string
}

func Load() Config {
	return Config{
		SreeificationServer: "host.docker.internal:50051",
	}
}
