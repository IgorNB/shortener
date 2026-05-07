package logger

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	Log  zerolog.Logger
	once sync.Once
)

func Init(logLevel string) {
	once.Do(func() {
		level, err := zerolog.ParseLevel(strings.ToLower(logLevel))
		if err != nil {
			level = zerolog.InfoLevel
		}
		Log = zerolog.New(os.Stdout).Level(level).With().Timestamp().Logger()
	})
}

func Logging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		uri := r.RequestURI
		method := r.Method
		writer := loggingResponseWriter{
			ResponseWriter: w,
			status:         0,
			size:           0,
		}
		h.ServeHTTP(&writer, r)
		duration := time.Since(start)
		Log.Info().
			Str("uri", uri).
			Str("method", method).
			Dur("duration", duration).
			Int("status", writer.status).
			Int("rs_size", writer.size).
			Msg("")
	}
	return http.HandlerFunc(logFn)
}

type (
	loggingResponseWriter struct {
		http.ResponseWriter
		status int
		size   int
	}
)

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	//if WriteHeader(statusCode int) was not called explicitly, then default is 200, so we must do the same
	if w.status == 0 {
		w.status = http.StatusOK
	}
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	//ResponseWriter.WriteHeader ignores second statusCode writes, so we must do the same
	if r.status == 0 {
		r.status = statusCode
	}
	r.ResponseWriter.WriteHeader(statusCode)
}
