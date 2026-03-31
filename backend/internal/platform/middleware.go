package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type ctxKey int

const requestIDKey ctxKey = 1

// RequestIDFromContext returns X-Request-ID value if set by WithRequestID.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("150405.000000000")))
	}
	return hex.EncodeToString(b)
}

// WithRequestID assigns X-Request-ID and stores it on the request context.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if rr.status == 0 {
		rr.status = http.StatusOK
	}
	n, err := rr.ResponseWriter.Write(b)
	rr.bytes += n
	return n, err
}

// WithAccessLog writes one JSON line per request to accessWriter (e.g. access.log).
func WithAccessLog(accessWriter io.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, status: 0}
		next.ServeHTTP(rr, r)

		status := rr.status
		if status == 0 {
			status = http.StatusOK
		}

		rec := map[string]any{
			"ts":        time.Now().UTC().Format(time.RFC3339Nano),
			"method":    r.Method,
			"path":      r.URL.Path,
			"query":     r.URL.RawQuery,
			"status":    status,
			"bytes":     rr.bytes,
			"duration":  time.Since(start).String(),
			"remote_ip": r.RemoteAddr,
			"req_id":    RequestIDFromContext(r.Context()),
			"user_agent": r.UserAgent(),
		}
		line, err := json.Marshal(rec)
		if err != nil {
			slog.Error("access log marshal failed", "err", err)
			return
		}
		if accessWriter != nil {
			if _, werr := accessWriter.Write(append(line, '\n')); werr != nil {
				slog.Error("access log write failed", "err", werr)
			}
		}
	})
}
