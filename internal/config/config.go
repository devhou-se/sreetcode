package config

type Config struct {
	// SreeificationServer is the address of the Sreeification gRPC server.
	SreeificationServer string
}

func Load() Config {
	return Config{
		//SreeificationServer: getEnv("SREEIFICATION_SERVER", "localhost:50051"),
		SreeificationServer: "127.0.0.1:50051",
	}
}
