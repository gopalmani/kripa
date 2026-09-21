// SPDX-License-Identifier: AGPL-3.0-or-later
package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gopalmani/kripa/internal/chart"
	"github.com/gopalmani/kripa/internal/ephemeris"
	"github.com/gopalmani/kripa/internal/memo"
	"github.com/gopalmani/kripa/internal/panchang"
	"github.com/rs/zerolog"
)

const SourceURL = "https://github.com/gopalmani/kripa"

type Config struct {
	Version      string
	Token        string
	Timeout      time.Duration
	MaxInflight  int
	CacheEntries int
	CacheTTL     time.Duration
	Logger       zerolog.Logger
}
type Server struct {
	cfg       Config
	engine    ephemeris.Provider
	cache     *memo.Cache
	slots     chan struct{}
	requests  [3]atomic.Uint64
	failures  [3]atomic.Uint64
	duration  [3][7]atomic.Uint64
	cacheHits atomic.Uint64
	rejected  atomic.Uint64
}

func New(engine ephemeris.Provider, cfg Config) *Server {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	if cfg.MaxInflight <= 0 {
		cfg.MaxInflight = 32
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 24 * time.Hour
	}
	return &Server{cfg: cfg, engine: engine, cache: memo.New(cfg.CacheEntries, cfg.CacheTTL), slots: make(chan struct{}, cfg.MaxInflight)}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "live"}) })
	mux.HandleFunc("GET /v1/meta", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"service": "kripa", "version": s.cfg.Version, "ephemeris_version": s.engine.Version(), "ephemeris_commit": ephemeris.SwissCommit, "data_version": ephemeris.DataVersion, "source_url": SourceURL, "license": "AGPL-3.0-or-later", "profiles": []string{"western_tropical_v1", panchang.Profile}, "date_range": "1900-01-01/2099-12-31", "panchang_review_status": "astronomical_preview"})
	})
	mux.Handle("GET /health/ready", s.protected(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ready", "ephemeris": s.engine.Version()})
	})))
	mux.Handle("GET /metrics", s.protected(http.HandlerFunc(s.metrics)))
	mux.Handle("POST /v1/charts", s.protected(s.calculate(0)))
	mux.Handle("POST /v1/panchang", s.protected(s.calculate(1)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Link", "<"+SourceURL+">; rel=\"source\"")
		mux.ServeHTTP(w, r)
	})
}
func (s *Server) protected(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Token != "" {
			actual := sha256.Sum256([]byte(r.Header.Get("Authorization")))
			expected := sha256.Sum256([]byte("Bearer " + s.cfg.Token))
			if subtle.ConstantTimeCompare(actual[:], expected[:]) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeError(w, 401, "unauthorized")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || typ != "application/json" {
		return errMedia
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("expected one JSON object")
	}
	return nil
}

var errMedia = errors.New("content type must be application/json")

func (s *Server) calculate(kind int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		status := 200
		cached := false
		route := "charts"
		if kind == 1 {
			route = "panchang"
		}
		defer func() {
			if recover() != nil {
				status = 500
				writeError(w, status, "internal_error")
			}
			elapsed := time.Since(started)
			s.requests[kind].Add(1)
			if status >= 400 {
				s.failures[kind].Add(1)
			}
			bucket := 6
			for i, lim := range []time.Duration{time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond, 25 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond} {
				if elapsed <= lim {
					bucket = i
					break
				}
			}
			s.duration[kind][bucket].Add(1)
			s.cfg.Logger.Debug().Str("route", route).Int("status", status).Bool("cache_hit", cached).Dur("duration_ms", elapsed).Msg("request")
		}()
		fail := func(code int, msg string) { status = code; writeError(w, code, msg) }
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
		default:
			s.rejected.Add(1)
			w.Header().Set("Retry-After", "1")
			fail(503, "overloaded")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), s.cfg.Timeout)
		defer cancel()
		var data []byte
		var err error
		if kind == 0 {
			var req chart.Request
			if err = decode(w, r, &req); err != nil {
				fail(decodeStatus(err), "invalid_json_request")
				return
			}
			if _, _, err = req.Resolve(); err != nil {
				fail(422, err.Error())
				return
			}
			var value chart.Chart
			value, err = chart.Calculate(ctx, s.engine, req)
			if err == nil {
				data, err = json.Marshal(value)
			}
		} else {
			var req panchang.Request
			if err = decode(w, r, &req); err != nil {
				fail(decodeStatus(err), "invalid_json_request")
				return
			}
			if _, _, err = req.Validate(); err != nil {
				fail(422, err.Error())
				return
			}
			input, _ := json.Marshal(req)
			digest := sha256.Sum256(input)
			key := panchang.Version + ":" + hex.EncodeToString(digest[:])
			data, cached, err = s.cache.Do(ctx, key, func() ([]byte, error) {
				value, err := panchang.Calculate(ctx, s.engine, req)
				if err != nil {
					return nil, err
				}
				return json.Marshal(value)
			})
			if cached && err == nil {
				s.cacheHits.Add(1)
			}
		}
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			switch {
			case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
				fail(504, "calculation_deadline_exceeded")
			case errors.Is(err, ephemeris.ErrNoEvent):
				fail(422, "sunrise_or_sunset_unavailable_for_location")
			default:
				fail(503, "calculation_unavailable")
			}
			return
		}
		if cached {
			w.Header().Set("X-Kripa-Cache", "HIT")
		} else {
			w.Header().Set("X-Kripa-Cache", "MISS")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(data)
	})
}
func decodeStatus(err error) int {
	if errors.Is(err, errMedia) {
		return 415
	}
	var max *http.MaxBytesError
	if errors.As(err, &max) {
		return 413
	}
	return 400
}
func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprintln(w, "# TYPE kripa_requests_total counter\n# TYPE kripa_errors_total counter\n# TYPE kripa_request_duration_seconds histogram")
	for k, route := range []string{"charts", "panchang"} {
		_, _ = fmt.Fprintf(w, "kripa_requests_total{route=%q} %d\nkripa_errors_total{route=%q} %d\n", route, s.requests[k].Load(), route, s.failures[k].Load())
		var total uint64
		for i, bound := range []string{"0.001", "0.005", "0.01", "0.025", "0.05", "0.1", "+Inf"} {
			total += s.duration[k][i].Load()
			_, _ = fmt.Fprintf(w, "kripa_request_duration_seconds_bucket{route=%q,le=%q} %d\n", route, bound, total)
		}
		_, _ = fmt.Fprintf(w, "kripa_request_duration_seconds_count{route=%q} %d\n", route, total)
	}
	_, _ = fmt.Fprintf(w, "kripa_cache_hits_total %d\nkripa_overload_rejections_total %d\nkripa_inflight %d\n", s.cacheHits.Load(), s.rejected.Load(), len(s.slots))
}
