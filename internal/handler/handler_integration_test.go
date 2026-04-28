//go:build integration
// +build integration

package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IgorNB/shortener/internal/repository"
	"github.com/IgorNB/shortener/internal/service"
	"github.com/stretchr/testify/assert"
)

type Step struct {
	method      string
	relativeUrl string
	contentType string
	body        string
}

type TestCase struct {
	name         string
	before       []Step
	step         Step
	assertStatus int
	assertBody   func(body string)
}

func TestHandler(t *testing.T) {
	// Success cases first, then failures
	tests := []TestCase{
		{
			name:   "POST success",
			before: []Step{},
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/",
				contentType: "text/plain",
				body:        "http://example.com",
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
					body:        "http://example.com",
				},
			},
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/",
				contentType: "text/plain",
				body:        "http://example.com",
			},
			assertStatus: http.StatusCreated,
			assertBody: func(body string) {
				assert.NotNil(t, body)
			},
		},
		// Failure cases
		{
			name: "POST failure - no content-type",
			step: Step{
				method:      http.MethodPost,
				relativeUrl: "/",
				contentType: "",
				body:        "http://example.com",
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			repo := repository.New()
			svc := service.New(repo)
			handler := New(svc, "http://localhost:8080/")

			// подготовка данных
			for _, b := range test.before {
				req := httptest.NewRequest(
					b.method,
					b.relativeUrl,
					bytes.NewBufferString(b.body),
				)

				if b.contentType != "" {
					req.Header.Set("Content-Type", b.contentType)
				}

				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
			}

			// основной шаг
			req := httptest.NewRequest(
				test.step.method,
				test.step.relativeUrl,
				bytes.NewBufferString(test.step.body),
			)

			if test.step.contentType != "" {
				req.Header.Set("Content-Type", test.step.contentType)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			resp := w.Result()

			assert.Equal(t, test.assertStatus, resp.StatusCode)

			if test.assertBody != nil {
				buf := new(bytes.Buffer)
				_, _ = buf.ReadFrom(resp.Body)
				test.assertBody(buf.String())
			}
		})
	}
}

func TestGetExistingURL(t *testing.T) {
	repo := repository.New()
	svc := service.New(repo)
	handler := New(svc, "http://localhost:8080/")

	postReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("http://example.com"))
	postReq.Header.Set("Content-Type", "text/plain")
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	postResp := postRec.Result()
	defer postResp.Body.Close()

	assert.Equal(t, http.StatusCreated, postResp.StatusCode)

	shortURLBytes, _ := io.ReadAll(postResp.Body)
	shortURL := string(shortURLBytes)
	parts := strings.Split(strings.TrimRight(shortURL, "/"), "/")
	shortID := parts[len(parts)-1]

	getReq := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	getResp := getRec.Result()
	defer getResp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, getResp.StatusCode)
	assert.Equal(t, "http://example.com", getResp.Header.Get("Location"))
}
