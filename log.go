package logmiddleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aleksmvp/logger"
	"github.com/prometheus/client_golang/prometheus"
)

type AccessLogMiddleware struct {
	Logger logger.ILogger
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

var (
	hits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hits",
	}, []string{"status", "path", "method"})

	timings = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "method_timings",
		Help: "Per method timing",
	}, []string{"method"})
)

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func NewAccessLogMiddleware(logger logger.ILogger) AccessLogMiddleware {
	err := prometheus.Register(hits)
	if err != nil {
		logger.LogError("middleware", "NewAccessLogMiddleware", err)
	}
	err = prometheus.Register(timings)
	if err != nil {
		logger.LogError("middleware", "NewAccessLogMiddleware", err)
	}
	return AccessLogMiddleware{Logger: logger}
}

func (m AccessLogMiddleware) Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rec := statusRecorder{w, 200}
		next.ServeHTTP(&rec, r)

		url := strings.Split(r.URL.String(), "?")[0]

		hits.WithLabelValues(strconv.Itoa(rec.status), url, r.Method).Inc()
		timings.WithLabelValues(r.URL.String()).Observe(time.Since(start).Seconds())

		m.Logger.LogAccess(r, rec.status, time.Since(start))
	})
}