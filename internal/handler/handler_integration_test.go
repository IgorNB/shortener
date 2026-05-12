package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/IgorNB/shortener/internal/config"
	"github.com/IgorNB/shortener/internal/config/logger"
	"github.com/IgorNB/shortener/internal/repository"
	"github.com/IgorNB/shortener/internal/service"
	"github.com/stretchr/testify/assert"
)

type Step struct {
	method          string
	relativeUrl     string
	contentType     string
	contentEncoding string
	acceptEncoding  string
	body            string
}

type TestCase struct {
	name                string
	before              []Step
	step                Step
	assertStatus        int
	assertBody          func(body string)
	assertContentEncode string
}

func TestIntegrationHandler(t *testing.T) {
	cfg := config.New(os.Args[1:])

	tests := []TestCase{
		{
			name:   "POST success",
			before: []Step{},
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/",
				contentType: "text/plain",
				body:        "http://example1.com",
			},
			assertStatus: http.StatusCreated,
			assertBody: func(body string) {
				assert.NotNil(t, body)
			},
		},
		{
			name: "POST success (duplicate)",
			before: []Step{
				{
					method:      http.MethodPost,
					relativeUrl: "/",
					contentType: "text/plain",
					body:        "http://example2.com",
				},
			},
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/",
				contentType: "text/plain",
				body:        "http://example2.com",
			},
			assertStatus: http.StatusCreated,
			assertBody: func(body string) {
				assert.NotNil(t, body)
			},
		},
		{
			name: "POST failure - no content-type",
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/",
				contentType: "",
				body:        "http://example3.com",
			},
			assertStatus: http.StatusBadRequest,
			assertBody:   nil,
		},
		{
			name: "POST failure - empty body",
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/",
				contentType: "text/plain",
				body:        "",
			},
			assertStatus: http.StatusBadRequest,
			assertBody:   nil,
		},
		{
			name: "GET failure - non-existent short URL",
			step: Step{
				method:      http.MethodGet,
				relativeUrl: "/nonexistent",
			},
			assertStatus: http.StatusBadRequest,
			assertBody:   nil,
		},
		{
			name:   "POST /api/shorten success",
			before: []Step{},
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/api/shorten",
				contentType: "application/json",
				body:        `{"url":"http://example4.com"}`,
			},
			assertStatus: http.StatusCreated,
			assertBody: func(body string) {
				assert.NotNil(t, body)
			},
		},
		{
			name: "POST /api/shorten success (duplicate)",
			before: []Step{
				{
					method:      http.MethodPost,
					relativeUrl: "/api/shorten",
					contentType: "application/json",
					body:        `{"url":"http://example5.com"}`,
				},
			},
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/api/shorten",
				contentType: "application/json",
				body:        `{"url":"http://example5.com"}`,
			},
			assertStatus: http.StatusCreated,
			assertBody: func(body string) {
				assert.NotNil(t, body)
			},
		},
		{
			name: "POST /api/shorten failure - no content-type",
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/api/shorten",
				contentType: "",
				body:        `{"url":"http://example6.com"}`,
			},
			assertStatus: http.StatusBadRequest,
			assertBody:   nil,
		},
		{
			name: "POST /api/shorten failure - empty body",
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/api/shorten",
				contentType: "application/json",
				body:        "",
			},
			assertStatus: http.StatusBadRequest,
			assertBody:   nil,
		},
		{
			name: "POST success gzip request",
			step: Step{
				method:          http.MethodPost,
				relativeUrl:     "/",
				contentType:     "text/plain",
				contentEncoding: "gzip",
				body:            "http://example7.com",
			},
			assertStatus: http.StatusCreated,
			assertBody: func(body string) {
				assert.NotNil(t, body)
			},
		},
		{
			name: "POST /api/shorten gzip response",
			step: Step{
				method:         http.MethodPost,
				relativeUrl:    "/api/shorten",
				contentType:    "application/json",
				acceptEncoding: "gzip",
				body:           `{"url":"http://example8.com"}`,
			},
			assertStatus:        http.StatusCreated,
			assertContentEncode: "gzip",
			assertBody: func(body string) {
				assert.NotNil(t, body)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := repository.New(cfg.FileStoragePath)
			svc := service.New(repo)
			handler := New(svc, cfg.BaseURL)

			for _, b := range test.before {
				var beforeBody io.Reader = bytes.NewBufferString(b.body)

				if b.contentEncoding == "gzip" {
					beforeBody = gzipString(b.body)
				}

				req := httptest.NewRequest(
					b.method,
					b.relativeUrl,
					beforeBody,
				)

				if b.contentType != "" {
					req.Header.Set("Content-Type", b.contentType)
				}

				if b.contentEncoding != "" {
					req.Header.Set("Content-Encoding", b.contentEncoding)
				}

				if b.acceptEncoding != "" {
					req.Header.Set("Accept-Encoding", b.acceptEncoding)
				}

				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
			}

			var stepBody io.Reader = bytes.NewBufferString(test.step.body)

			if test.step.contentEncoding == "gzip" {
				stepBody = gzipString(test.step.body)
			}

			req := httptest.NewRequest(
				test.step.method,
				test.step.relativeUrl,
				stepBody,
			)

			if test.step.contentType != "" {
				req.Header.Set("Content-Type", test.step.contentType)
			}

			if test.step.contentEncoding != "" {
				req.Header.Set("Content-Encoding", test.step.contentEncoding)
			}

			if test.step.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", test.step.acceptEncoding)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, test.assertStatus, resp.StatusCode)
			assert.Equal(t, test.assertContentEncode, resp.Header.Get("Content-Encoding"))

			if test.assertBody != nil {
				var body string

				if resp.Header.Get("Content-Encoding") == "gzip" {
					body = readGzipBody(t, resp.Body)
				} else {
					buf := new(bytes.Buffer)
					_, _ = buf.ReadFrom(resp.Body)
					body = buf.String()
				}

				test.assertBody(body)
			}
		})
	}
}

func TestIntegrationGetExistingURL(t *testing.T) {
	cfg := config.New(os.Args[1:])
	logger.Init(cfg.LogLevel)
	repo := repository.New(cfg.FileStoragePath)
	svc := service.New(repo)
	handler := New(svc, cfg.BaseURL)

	postReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("http://example9.com"))
	postReq.Header.Set("Content-Type", "text/plain")
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	postResp := postRec.Result()
	defer postResp.Body.Close()

	assert.Equal(t, http.StatusCreated, postResp.StatusCode)

	shortURLBytes, err := io.ReadAll(postResp.Body)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Test stopped")
	}
	shortURL := string(shortURLBytes)
	parts := strings.Split(strings.TrimRight(shortURL, "/"), "/")
	shortID := parts[len(parts)-1]

	getReq := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	getResp := getRec.Result()
	defer getResp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, getResp.StatusCode)
	assert.Equal(t, "http://example9.com", getResp.Header.Get("Location"))
}

func TestIntegrationGetExistingURLViaAPI(t *testing.T) {
	cfg := config.New(os.Args[1:])
	logger.Init(cfg.LogLevel)
	repo := repository.New(cfg.FileStoragePath)
	svc := service.New(repo)
	handler := New(svc, cfg.BaseURL)

	origURL := "http://example10.com"
	reqBody := `{"url":"` + origURL + `"}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(reqBody))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	postResp := postRec.Result()
	defer postResp.Body.Close()

	assert.Equal(t, http.StatusCreated, postResp.StatusCode)
	assert.Equal(t, "application/json", postResp.Header.Get("Content-Type"))

	var rs struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(postResp.Body).Decode(&rs); err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, rs.Result)
	shortURL := rs.Result
	parts := strings.Split(strings.TrimRight(shortURL, "/"), "/")
	shortID := parts[len(parts)-1]

	getReq := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	getResp := getRec.Result()
	defer getResp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, getResp.StatusCode)
	assert.Equal(t, origURL, getResp.Header.Get("Location"))
}
