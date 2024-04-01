package metricsandlogging

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/thorsager/dude/requestid"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "dude_http_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path"})
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dude_http_requests_total",
		Help: "Number of HTTP requests.",
	}, []string{"path", "method", "status"})
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

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &writerWrapper{actual: w, statusCode: 200}
		next.ServeHTTP(ww, r)
		duration := time.Since(start)

		log.Printf("[%s] %s - \"%s %s %s\" %d %d %s %s (%s)",
			requestid.GetID(r.Context()),
			ipFromRemoteAddr(r.RemoteAddr),
			r.Method,
			r.URL.Path,
			r.Proto,
			ww.statusCode,
			ww.written,
			qne(r.Referer()),
			qne(r.UserAgent()),
			duration,
		)
		httpDuration.WithLabelValues(r.URL.Path).Observe(duration.Seconds())
		httpRequestsTotal.WithLabelValues(r.URL.Path, r.Method, http.StatusText(ww.statusCode)).Inc()
	})
}

func ipFromRemoteAddr(s string) string {
	if i := strings.LastIndex(s, ":"); i != -1 {
		return s[:i]
	}
	return s
}

func qne(s string) string {
	if s != "" {
		return "\"" + s + "\""
	}
	return s
}
