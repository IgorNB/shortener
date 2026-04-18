package main

import (
	"net/http"

	"github.com/IgorNB/shortener/internal/handler"
	"github.com/IgorNB/shortener/internal/repository"
	"github.com/IgorNB/shortener/internal/service"
)

const BaseURL = "http://localhost:8080/"

func main() {
	repo := repository.New()
	svc := service.New(repo)
	h := handler.New(svc, BaseURL)

	if err := http.ListenAndServe(":8080", h); err != nil {
		panic(err)
	}
}
