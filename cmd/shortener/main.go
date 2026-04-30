package main

import (
	"log"
	"net/http"

	"github.com/IgorNB/shortener/internal/config"
	"github.com/IgorNB/shortener/internal/handler"
	"github.com/IgorNB/shortener/internal/repository"
	"github.com/IgorNB/shortener/internal/service"
)

func main() {
	config.Parse()

	repo := repository.New()
	svc := service.New(repo)
	h := handler.New(svc, config.BaseURL)

	if err := http.ListenAndServe(config.ServerAddress, h); err != nil {
		log.Fatal(err)
	}
}
