package config

import (
	"flag"
	"os"
	"strings"
)

const ContentType = "Content-Type"

const ContentTypeTextPlain = "text/plain"

const ContentTypeTextHtml = "text/html"

const ContentTypeJson = "application/json"

type Config struct {
	ServerAddress   string
	BaseURL         string
	LogLevel        string
	FileStoragePath string
}

func New(args []string) *Config {
	cfg := &Config{}
	fs := flag.NewFlagSet("shortener", flag.ContinueOnError)

	fs.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "HTTP server address")
	fs.StringVar(&cfg.BaseURL, "b", "http://localhost:8080/", "Base URL for short links")
	fs.StringVar(&cfg.LogLevel, "log_level", "INFO", "log level")
	fs.StringVar(&cfg.FileStoragePath, "f", "shortener_db.txt", "Shortener DB file path. It will be created on app start if not exists")
	_ = fs.Parse(args)
	if env, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.ServerAddress = env
	}
	if env, ok := os.LookupEnv("BASE_URL"); ok {
		cfg.BaseURL = env
	}
	if env, ok := os.LookupEnv("LOG_LEVEL"); ok {
		cfg.LogLevel = env
	}
	if env, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = env
	}
	if cfg.BaseURL != "" {
		cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/") + "/"
	}

	return cfg
}
