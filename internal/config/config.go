package config

import (
	"flag"
	"strings"
)

var (
	ServerAddress string
	BaseURL       string
)

func init() {
	flag.StringVar(&ServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&BaseURL, "b", "http://localhost:8080/", "Base URL for short links")
}

func Parse() {
	if !flag.Parsed() {
		flag.Parse()
	}
	if BaseURL != "" {
		BaseURL = strings.TrimSuffix(BaseURL, "/") + "/"
	}
}
