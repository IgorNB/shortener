package compress

import (
	"compress/gzip"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/IgorNB/shortener/internal/config"
	"github.com/IgorNB/shortener/internal/config/logger"
)

const ContentEncoding = "Content-Encoding"
const AcceptEncoding = "Accept-Encoding"
const ContentEncodingGzip = "gzip"

func Compress(handler http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := r.Header.Get(ContentEncoding)
		requestGzip := contentEncoding == ContentEncodingGzip

		if requestGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body = cr
			defer func() {
				if err := cr.Close(); err != nil {
					logger.Log.Error().Err(err).Msg("Cannot free resource")
				}
			}()
		}

		cw := newCompressWriter(
			w,
			strings.Contains(r.Header.Get(AcceptEncoding), ContentEncodingGzip),
		)

		defer func() {
			if err := cw.Close(); err != nil {
				logger.Log.Error().Err(err).Msg("Cannot close gzip writer")
			}
		}()

		handler.ServeHTTP(cw, r)
	}
	return http.HandlerFunc(logFn)
}

type compressWriter struct {
	w                  http.ResponseWriter
	acceptEncodingGzip bool
	zw                 *gzip.Writer //lazy
}

func newCompressWriter(w http.ResponseWriter, acceptEncodingGzip bool) *compressWriter {
	return &compressWriter{w: w, acceptEncodingGzip: acceptEncodingGzip}
}

func (cw *compressWriter) Header() http.Header {
	return cw.w.Header()
}

func (cw *compressWriter) Write(p []byte) (int, error) {
	cw.lazyInitGzip()
	if cw.zw != nil {
		return cw.zw.Write(p)
	}
	return cw.w.Write(p)
}

func (cw *compressWriter) WriteHeader(statusCode int) {
	cw.lazyInitGzip()
	cw.w.WriteHeader(statusCode)
}

// !we know the RESPONSE content type only on WriteHeader(statusCode int) or Write(p []byte), so we lazily decide to create a gzip writer
func (cw *compressWriter) lazyInitGzip() {
	if cw.acceptEncodingGzip && cw.zw == nil {
		ct := cw.w.Header().Get(config.ContentType)
		if mediaType, _, _ := mime.ParseMediaType(ct); mediaType == config.ContentTypeJson || mediaType == config.ContentTypeTextHtml {
			cw.w.Header().Set(ContentEncoding, ContentEncodingGzip)
			cw.zw = gzip.NewWriter(cw.w)
		}
	}
}

func (cw *compressWriter) Close() error {
	if cw.zw == nil {
		return nil
	}
	return cw.zw.Close()
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (cr *compressReader) Read(p []byte) (n int, err error) {
	return cr.zr.Read(p)
}

func (cr *compressReader) Close() error {
	return errors.Join(
		cr.zr.Close(),
		cr.r.Close(),
	)
}
