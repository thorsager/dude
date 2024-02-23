package requestlogging

import (
	"github.com/thorsager/dude/requestid"
	"log"
	"net/http"
	"time"
)

type writerWrapper struct {
	actual     http.ResponseWriter
	written    int64
	statusCode int
}

func (w *writerWrapper) Header() http.Header {
	return w.actual.Header()
}

func (w *writerWrapper) Write(bytes []byte) (int, error) {
	written, err := w.actual.Write(bytes)
	w.written += int64(written)
	return written, err
}

func (w *writerWrapper) WriteHeader(statusCode int) {
	w.actual.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &writerWrapper{actual: w, statusCode: 200}
		next(ww, r)
		log.Printf("[%s] %s %s - %d (%d bytes) in %s",
			requestid.GetID(r.Context()),
			r.Method,
			r.URL.Path,
			ww.statusCode,
			ww.written,
			time.Since(start),
		)
	}
}
