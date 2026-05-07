package handler

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/IgorNB/shortener/internal/config/logger"
	"github.com/IgorNB/shortener/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const contentTypeTextPlain = "text/plain"

const contentTypeJson = "application/json"

//go:generate mockery --name URLService --output ./mocks --outpkg mocks
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

	r.Use(logger.Logging, middleware.Recoverer)

	r.NotFound(h.badRequestHandler)
	r.MethodNotAllowed(h.badRequestHandler)

	r.Post("/", h.handlePost)
	r.Post("/api/shorten", h.handleJsonPost)

	r.Get("/{id}", h.handleGet)

	return r
}

func (h *URLHandler) badRequestHandler(rw http.ResponseWriter, rq *http.Request) {
	rw.WriteHeader(http.StatusBadRequest)
}

func (h *URLHandler) handlePost(rw http.ResponseWriter, rq *http.Request) {
	mediaType, _, _ := mime.ParseMediaType(rq.Header.Get("Content-Type"))
	if mediaType != contentTypeTextPlain {
		http.Error(rw, "invalid content type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(rq.Body)
	if err != nil {
		http.Error(rw, "invalid body", http.StatusBadRequest)
		return
	}

	origURL := string(body)
	if strings.TrimSpace(origURL) == "" {
		http.Error(rw, "invalid body", http.StatusBadRequest)
		return
	}

	shortID := h.svc.GetOrCreate(origURL)
	if shortID == "" {
		http.Error(rw, "failed to shorten url", http.StatusBadRequest)
		return
	}

	resURL, err := url.JoinPath(h.baseURL, shortID)
	if err != nil {
		http.Error(rw, "failed to shorten url", http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", contentTypeTextPlain)
	rw.WriteHeader(http.StatusCreated)
	_, _ = rw.Write([]byte(resURL))
}

func (h *URLHandler) handleJsonPost(rw http.ResponseWriter, rq *http.Request) {
	mediaType, _, _ := mime.ParseMediaType(rq.Header.Get("Content-Type"))
	if mediaType != contentTypeJson {
		http.Error(rw, "invalid content type", http.StatusBadRequest)
		return
	}

	var shortenRq model.ShortenRq
	if err := json.NewDecoder(rq.Body).Decode(&shortenRq); err != nil {
		http.Error(rw, "invalid body", http.StatusBadRequest)
		return
	}

	origURL := shortenRq.Url
	if strings.TrimSpace(origURL) == "" {
		http.Error(rw, "invalid body", http.StatusBadRequest)
		return
	}

	shortID := h.svc.GetOrCreate(origURL)
	if shortID == "" {
		http.Error(rw, "failed to shorten url", http.StatusBadRequest)
		return
	}

	resURL, err := url.JoinPath(h.baseURL, shortID)
	if err != nil {
		http.Error(rw, "failed to shorten url", http.StatusBadRequest)
		return
	}

	marshal, err := json.Marshal(model.ShortenRs{Result: resURL})
	if err != nil {
		http.Error(rw, "failed to shorten url", http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", contentTypeJson)
	rw.WriteHeader(http.StatusCreated)
	_, _ = rw.Write(marshal)
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
