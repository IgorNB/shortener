//go:build !integration

package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/IgorNB/shortener/internal/config"
	"github.com/IgorNB/shortener/internal/handler/mocks"
	"github.com/IgorNB/shortener/internal/middleware/logger"
	"github.com/stretchr/testify/assert"
)

const (
	methodGetOrCreate = "GetOrCreate"
	methodGetOrigURL  = "GetOrigURL"
)

func gzipString(s string) *bytes.Buffer {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, _ = gz.Write([]byte(s))
	_ = gz.Close()
	return &b
}

func readGzipBody(t *testing.T, r io.Reader) string {
	gr, err := gzip.NewReader(r)
	assert.NoError(t, err)
	defer gr.Close()

	data, err := io.ReadAll(gr)
	assert.NoError(t, err)

	return string(data)
}

func TestHandler(t *testing.T) {
	cfg := config.New(os.Args[1:])
	logger.Init(cfg.LogLevel)

	tests := []struct {
		name                string
		method              string
		path                string
		contentType         string
		contentEncoding     string
		acceptEncoding      string
		body                string
		setupMock           func(m *mocks.URLService)
		wantStatus          int
		wantBody            string
		wantContentEncoding string
	}{
		{
			name:        "POST success",
			method:      http.MethodPost,
			path:        "/",
			contentType: "text/plain",
			body:        "http://example1.com",
			setupMock: func(m *mocks.URLService) {
				m.On(methodGetOrCreate, "http://example1.com").Return("EwHXdJfB").Once()
			},
			wantStatus: http.StatusCreated,
			wantBody:   cfg.BaseURL + "EwHXdJfB",
		},
		{
			name:        "POST success (duplicate)",
			method:      http.MethodPost,
			path:        "/",
			contentType: "text/plain",
			body:        "http://example2.com",
			setupMock: func(m *mocks.URLService) {
				m.On(methodGetOrCreate, "http://example2.com").Return("EwHXdJfB").Once()
			},
			wantStatus: http.StatusCreated,
			wantBody:   cfg.BaseURL + "EwHXdJfB",
		},
		{
			name:       "POST failure - no content-type",
			method:     http.MethodPost,
			path:       "/",
			body:       "http://example3.com",
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
		{
			name:        "POST /api/shorten success",
			method:      http.MethodPost,
			path:        "/api/shorten",
			contentType: "application/json",
			body:        `{"url":"http://example4.com"}`,
			setupMock: func(m *mocks.URLService) {
				m.On(methodGetOrCreate, "http://example4.com").Return("EwHXdJfB").Once()
			},
			wantStatus: http.StatusCreated,
			wantBody:   `{"result":"` + cfg.BaseURL + `EwHXdJfB"}`,
		},
		{
			name:        "POST /api/shorten success (duplicate)",
			method:      http.MethodPost,
			path:        "/api/shorten",
			contentType: "application/json",
			body:        `{"url":"http://example5.com"}`,
			setupMock: func(m *mocks.URLService) {
				m.On(methodGetOrCreate, "http://example5.com").Return("EwHXdJfB").Once()
			},
			wantStatus: http.StatusCreated,
			wantBody:   `{"result":"` + cfg.BaseURL + `EwHXdJfB"}`,
		},
		{
			name:        "POST /api/shorten failure - no content-type",
			method:      http.MethodPost,
			path:        "/api/shorten",
			contentType: "",
			body:        `{"url":"http://example6.com"}`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "POST /api/shorten failure - empty body",
			method:      http.MethodPost,
			path:        "/api/shorten",
			contentType: "application/json",
			body:        "",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:            "POST success gzip request",
			method:          http.MethodPost,
			path:            "/",
			contentType:     "text/plain",
			contentEncoding: "gzip",
			body:            "http://example7.com",
			setupMock: func(m *mocks.URLService) {
				m.On(methodGetOrCreate, "http://example7.com").Return("EwHXdJfB").Once()
			},
			wantStatus: http.StatusCreated,
			wantBody:   cfg.BaseURL + "EwHXdJfB",
		},
		{
			name:           "POST /api/shorten gzip response",
			method:         http.MethodPost,
			path:           "/api/shorten",
			contentType:    "application/json",
			acceptEncoding: "gzip",
			body:           `{"url":"http://example8.com"}`,
			setupMock: func(m *mocks.URLService) {
				m.On(methodGetOrCreate, "http://example8.com").Return("EwHXdJfB").Once()
			},
			wantStatus:          http.StatusCreated,
			wantBody:            `{"result":"` + cfg.BaseURL + `EwHXdJfB"}`,
			wantContentEncoding: "gzip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mocks.URLService)
			if tt.setupMock != nil {
				tt.setupMock(svc)
			}

			var body io.Reader = bytes.NewBufferString(tt.body)

			if tt.contentEncoding == "gzip" {
				body = gzipString(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}

			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			rr := httptest.NewRecorder()

			New(svc, cfg.BaseURL).ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, tt.wantContentEncoding, res.Header.Get("Content-Encoding"))

			if tt.wantBody != "" {
				var responseBody string

				if res.Header.Get("Content-Encoding") == "gzip" {
					responseBody = readGzipBody(t, res.Body)
				} else {
					bodyBytes, err := io.ReadAll(res.Body)
					assert.NoError(t, err)
					responseBody = string(bodyBytes)
				}

				assert.Equal(t, tt.wantBody, responseBody)
			}

			svc.AssertExpectations(t)
		})
	}
}

func TestGetExistingURL(t *testing.T) {
	cfg := config.New(os.Args[1:])
	logger.Init(cfg.LogLevel)
	const (
		origURL = "http://example9.com"
		shortID = "EwHXdJfB"
	)

	svc := new(mocks.URLService)
	svc.On(methodGetOrigURL, shortID).Return(origURL).Once()

	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	rr := httptest.NewRecorder()

	New(svc, cfg.BaseURL).ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
	assert.Equal(t, origURL, res.Header.Get("Location"))
	svc.AssertExpectations(t)
}

func TestGetExistingURLViaAPI(t *testing.T) {
	cfg := config.New(os.Args[1:])
	logger.Init(cfg.LogLevel)
	const (
		origURL = "http://example10.com"
		shortID = "EwHXdJfB"
	)

	svc := new(mocks.URLService)
	svc.On(methodGetOrCreate, origURL).Return(shortID).Once()

	reqBody := `{"url":"` + origURL + `"}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(reqBody))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	New(svc, cfg.BaseURL).ServeHTTP(postRec, postReq)

	postResp := postRec.Result()
	defer postResp.Body.Close()

	assert.Equal(t, http.StatusCreated, postResp.StatusCode)
	assert.Equal(t, "application/json", postResp.Header.Get("Content-Type"))

	var rs struct {
		Result string `json:"result"`
	}
	err := json.NewDecoder(postResp.Body).Decode(&rs)
	assert.NoError(t, err)
	assert.NotEmpty(t, rs.Result)

	parts := strings.Split(strings.TrimRight(rs.Result, "/"), "/")
	actualShortID := parts[len(parts)-1]

	svc.On(methodGetOrigURL, actualShortID).Return(origURL).Once()

	getReq := httptest.NewRequest(http.MethodGet, "/"+actualShortID, nil)
	getRec := httptest.NewRecorder()
	New(svc, cfg.BaseURL).ServeHTTP(getRec, getReq)

	getResp := getRec.Result()
	defer getResp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, getResp.StatusCode)
	assert.Equal(t, origURL, getResp.Header.Get("Location"))
	svc.AssertExpectations(t)
}
