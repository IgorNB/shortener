package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

func New(svc URLService, baseURL string) http.Handler {
	h := &URLHandler{
		svc:     svc,
		baseURL: baseURL,
	}

	r := chi.NewRouter()

	r.Use(middleware.Recoverer)

	r.NotFound(h.badRequestHandler)
	r.MethodNotAllowed(h.badRequestHandler)

	r.Post("/", h.handlePost)
	r.Get("/{id}", h.handleGet)

	return r
}

func (h *URLHandler) badRequestHandler(rw http.ResponseWriter, rq *http.Request) {
	rw.WriteHeader(http.StatusBadRequest)
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
	shortID := chi.URLParam(rq, "id")

	origURL := h.svc.GetOrigURL(shortID)
	if origURL == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	http.Redirect(rw, rq, origURL, http.StatusTemporaryRedirect)
}
