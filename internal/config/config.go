package config

import (
	"flag"
	"strings"
)

var (
	ServerAddress string
	BaseURL       string
)

func Parse() {
	if !flag.Parsed() {
		flag.StringVar(&ServerAddress, "a", "localhost:8080", "HTTP server address")
		flag.StringVar(&BaseURL, "b", "http://localhost:8080/", "Base URL for short links")
		flag.Parse()
	}
	if BaseURL != "" {
		BaseURL = strings.TrimSuffix(BaseURL, "/") + "/"
	}
}
