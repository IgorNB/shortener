package main

import (
	"net/http"
	"os"

	"github.com/IgorNB/shortener/internal/config"
	"github.com/IgorNB/shortener/internal/handler"
	"github.com/IgorNB/shortener/internal/middleware/logger"
	"github.com/IgorNB/shortener/internal/repository"
	"github.com/IgorNB/shortener/internal/service"
)

func main() {
	cfg := config.New(os.Args[1:])
	logger.Init(cfg.LogLevel)

	repo := repository.New(cfg.FileStoragePath)
	svc := service.New(repo)
	h := handler.New(svc, cfg.BaseURL)

	if err := http.ListenAndServe(cfg.ServerAddress, h); err != nil {
		logger.Log.Fatal().Err(err).Msg("server stopped")
	}
}
