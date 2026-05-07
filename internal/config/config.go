package config

import (
	"flag"
	"os"
	"strings"
)

var (
	ServerAddress string
	BaseURL       string
	LogLevel      string
)

func Parse() {
	if !flag.Parsed() {
		flag.StringVar(&ServerAddress, "a", "localhost:8080", "HTTP server address")
		flag.StringVar(&BaseURL, "b", "http://localhost:8080/", "Base URL for short links")
		flag.StringVar(&LogLevel, "log_level", "INFO", "log level")
		if env, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
			ServerAddress = env
		}
		if env, ok := os.LookupEnv("BASE_URL"); ok {
			BaseURL = env
		}
		if env, ok := os.LookupEnv("LOG_LEVEL"); ok {
			LogLevel = env
		}
		flag.Parse()
	}
	if BaseURL != "" {
		BaseURL = strings.TrimSuffix(BaseURL, "/") + "/"
	}
}
