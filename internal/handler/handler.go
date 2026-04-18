package handler

import (
	"io"
	"net/http"
	"strings"
)

const contentTypeTextPlain = "text/plain"

type URLService interface {
	GetOrCreate(origURL string) string
	GetOrigURL(shortID string) string
}

type URLHandler struct {
	svc     URLService
	baseURL string
}

func New(svc URLService, baseURL string) *URLHandler {
	return &URLHandler{svc: svc, baseURL: baseURL}
}

func (h *URLHandler) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	switch rq.Method {
	case http.MethodPost:
		h.handlePost(rw, rq)
	case http.MethodGet:
		h.handleGet(rw, rq)
	default:
		rw.WriteHeader(http.StatusBadRequest)
	}
}

func (h *URLHandler) handlePost(rw http.ResponseWriter, rq *http.Request) {
	if !strings.HasPrefix(rq.Header.Get("Content-Type"), contentTypeTextPlain) {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(rq.Body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	origURL := string(body)
	if strings.TrimSpace(origURL) == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	shortID := h.svc.GetOrCreate(origURL)
	if shortID == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", contentTypeTextPlain)
	rw.WriteHeader(http.StatusCreated)
	_, _ = rw.Write([]byte(h.baseURL + shortID))
}

func (h *URLHandler) handleGet(rw http.ResponseWriter, rq *http.Request) {
	if len(rq.URL.Path) <= 1 {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	shortID := rq.URL.Path[1:]

	origURL := h.svc.GetOrigURL(shortID)
	if origURL == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	http.Redirect(rw, rq, origURL, http.StatusTemporaryRedirect)
}
