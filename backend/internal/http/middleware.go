package http

import (
	"compress/gzip"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func cors(allowed []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && originAllowed(allowed, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Content-Encoding")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func originAllowed(allowed []string, origin string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}

func gzipJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		det := &gzipDetector{ResponseWriter: w, r: r}
		next.ServeHTTP(det, r)
		if det.gz != nil {
			_ = det.gz.Close()
		}
	})
}

type gzipDetector struct {
	http.ResponseWriter
	r       *http.Request
	gz      *gzip.Writer
	started bool
}

func (d *gzipDetector) WriteHeader(status int) {
	ct := d.Header().Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		d.Header().Set("Content-Encoding", "gzip")
		d.Header().Add("Vary", "Accept-Encoding")
		d.Header().Del("Content-Length")
		d.gz = gzip.NewWriter(d.ResponseWriter)
		d.started = true
	}
	d.ResponseWriter.WriteHeader(status)
}

func (d *gzipDetector) Write(b []byte) (int, error) {
	if !d.started {
		// Header not sent yet: infer from Content-Type.
		ct := d.Header().Get("Content-Type")
		if strings.Contains(ct, "application/json") {
			d.Header().Set("Content-Encoding", "gzip")
			d.Header().Add("Vary", "Accept-Encoding")
			d.Header().Del("Content-Length")
			d.gz = gzip.NewWriter(d.ResponseWriter)
			d.started = true
		}
	}
	if d.gz != nil {
		return d.gz.Write(b)
	}
	return d.ResponseWriter.Write(b)
}

func (d *gzipDetector) Flush() {
	if d.gz != nil {
		_ = d.gz.Flush()
	}
	if f, ok := d.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func Chain(next http.Handler, allowed []string) http.Handler {
	h := gzipJSON(next)
	h = cors(allowed, h)
	h = requestLogger(h)
	h = recoverer(h)
	return h
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sr, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sr.status,
			"duration", time.Since(start).String(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "err", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
