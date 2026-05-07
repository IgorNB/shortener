package main

import (
	"net/http"

	"github.com/IgorNB/shortener/internal/config"
	"github.com/IgorNB/shortener/internal/config/logger"
	"github.com/IgorNB/shortener/internal/handler"
	"github.com/IgorNB/shortener/internal/repository"
	"github.com/IgorNB/shortener/internal/service"
)

func main() {
	config.Parse()
	logger.Init(config.LogLevel)

	repo := repository.New()
	svc := service.New(repo)
	h := handler.New(svc, config.BaseURL)

	if err := http.ListenAndServe(config.ServerAddress, h); err != nil {
		logger.Log.Fatal().Err(err).Msg("server stopped")
	}
}
