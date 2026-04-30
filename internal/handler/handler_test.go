//go:build !integration

package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IgorNB/shortener/internal/config"
	"github.com/IgorNB/shortener/internal/handler/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	methodGetOrCreate = "GetOrCreate"
	methodGetOrigURL  = "GetOrigURL"
)

func TestHandler(t *testing.T) {
	config.Parse()
	tests := []struct {
		name        string
		method      string
		path        string
		contentType string
		body        string
		setupMock   func(m *mocks.URLService)
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "POST success",
			method:      http.MethodPost,
			path:        "/",
			contentType: "text/plain",
			body:        "http://example.com",
			setupMock: func(m *mocks.URLService) {
				m.On(methodGetOrCreate, "http://example.com").Return("EwHXdJfB").Once()
			},
			wantStatus: http.StatusCreated,
			wantBody:   "EwHXdJfB",
		},
		{
			name:        "POST success (duplicate)",
			method:      http.MethodPost,
			path:        "/",
			contentType: "text/plain",
			body:        "http://example.com",
			setupMock: func(m *mocks.URLService) {
				m.On(methodGetOrCreate, "http://example.com").Return("EwHXdJfB").Once()
			},
			wantStatus: http.StatusCreated,
			wantBody:   "EwHXdJfB",
		},
		{
			name:       "POST failure - no content-type",
			method:     http.MethodPost,
			path:       "/",
			body:       "http://example.com",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "POST failure - empty body",
			method:      http.MethodPost,
			path:        "/",
			contentType: "text/plain",
			body:        "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:   "GET failure - non-existent short URL",
			method: http.MethodGet,
			path:   "/nonexistent",
			setupMock: func(m *mocks.URLService) {
				m.On(methodGetOrigURL, "nonexistent").Return("").Once()
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mocks.URLService)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()

			New(svc, config.BaseURL).ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantBody != "" {
				body, err := io.ReadAll(res.Body)
				assert.NoError(t, err)
				assert.Equal(t, config.BaseURL+tt.wantBody, string(body))
			}
			svc.AssertExpectations(t)
		})
	}
}

func TestGetExistingURL(t *testing.T) {
	config.Parse()
	const (
		origURL = "http://example.com"
		shortID = "EwHXdJfB"
	)

	svc := new(mocks.URLService)
	svc.On(methodGetOrigURL, shortID).Return(origURL).Once()

	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	rr := httptest.NewRecorder()

	New(svc, config.BaseURL).ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
	assert.Equal(t, origURL, res.Header.Get("Location"))
	svc.AssertExpectations(t)
}
