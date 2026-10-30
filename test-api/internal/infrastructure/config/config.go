package config

import (
	"os"
)

type Config struct {
	HTTPAddr string
}

func Load() (Config, error) {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	config := Config{HTTPAddr: addr}
	return config, nil
}
