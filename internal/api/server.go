// SPDX-License-Identifier: AGPL-3.0-or-later
package api

import (
	"bytes"
	"context"
	"crypto/rand"
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
	cfg             Config
	engine          ephemeris.Provider
	cache           *memo.Cache
	slots           chan struct{}
	requests        [3]atomic.Uint64
	failures        [3]atomic.Uint64
	duration        [3][7]atomic.Uint64
	cacheHits       atomic.Uint64
	rejected        atomic.Uint64
	requestSequence atomic.Uint64
	requestPrefix   string
	cacheMisses     atomic.Uint64
	durationSum     [3]atomic.Uint64
	calcCount       atomic.Uint64
	calcNanos       atomic.Uint64
	queueNanos      atomic.Uint64
	nativeFailures  atomic.Uint64
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
	var prefix [8]byte
	if _, err := rand.Read(prefix[:]); err != nil {
		panic(err)
	}
	s := &Server{cfg: cfg, requestPrefix: hex.EncodeToString(prefix[:]), engine: engine, cache: memo.New(cfg.CacheEntries, cfg.CacheTTL), slots: make(chan struct{}, cfg.MaxInflight)}
	s.engine = measuredProvider{Provider: engine, server: s}
	return s
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "live"}) })
	mux.HandleFunc("GET /v1/meta", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"service": "kripa", "version": s.cfg.Version, "ephemeris_version": s.engine.Version(), "ephemeris_commit": ephemeris.SwissCommit, "data_version": ephemeris.DataVersion, "source_url": SourceURL, "license": "AGPL-3.0-or-later", "profiles": []string{"western_tropical_v1", panchang.Profile}, "date_range": "1900-01-01/2099-12-31", "panchang_review_status": "astronomical_preview"})
	})
	mux.Handle("GET /health/ready", s.protected(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
		default:
			s.rejected.Add(1)
			w.Header().Set("Retry-After", "1")
			writeError(w, 503, "overloaded")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), s.cfg.Timeout)
		defer cancel()
		err := s.engine.WithSession(ctx, func(session ephemeris.Session) error {
			jd, err := session.JulianDay(time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC))
			if err != nil {
				return err
			}
			for _, body := range []int{0, 1, 15} {
				if _, err := session.Position(jd, body, false); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			writeError(w, 503, "ephemeris_unavailable")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ready", "ephemeris": s.engine.Version()})
	})))
	mux.Handle("GET /metrics", s.protected(http.HandlerFunc(s.metrics)))
	mux.Handle("POST /v1/charts", s.protected(s.calculate(0)))
	mux.Handle("POST /v1/panchang", s.protected(s.calculate(1)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", fmt.Sprintf("%s-%d", s.requestPrefix, s.requestSequence.Add(1)))
		defer func() {
			if recover() != nil {
				writeError(w, 500, "internal_error")
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Link", "<"+SourceURL+">; rel=\"source\"")
		// Avoid ServeMux's plain-text 404/405 responses for this JSON API.
		methods := map[string]string{"/health/live": "GET", "/health/ready": "GET", "/v1/meta": "GET", "/metrics": "GET", "/v1/charts": "POST", "/v1/panchang": "POST"}
		method, exists := methods[r.URL.Path]
		if !exists {
			writeError(w, 404, "not_found")
			return
		}
		if r.Method != method && !(method == "GET" && r.Method == "HEAD") {
			if method == "GET" {
				method = "GET, HEAD"
			}
			w.Header().Set("Allow", method)
			writeError(w, 405, "method_not_allowed")
			return
		}
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
	// Validate object shape before decoding: encoding/json otherwise accepts
	// null and silently lets duplicate keys overwrite earlier values.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	shape := json.NewDecoder(bytes.NewReader(body))
	token, err := shape.Token()
	if err != nil || token != json.Delim('{') {
		return errors.New("expected object")
	}
	fields := map[string]json.RawMessage{}
	allowed := map[string]bool{"date": true, "timezone": true, "latitude": true, "longitude": true, "profile": true}
	if _, ok := dst.(*chart.Request); ok {
		allowed["time"] = true
		allowed["time_status"] = true
	}
	for shape.More() {
		token, err := shape.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return errors.New("expected field")
		}
		if !allowed[key] {
			return errors.New("unknown field")
		}
		if _, exists := fields[key]; exists {
			return errors.New("duplicate field")
		}
		var value json.RawMessage
		if err := shape.Decode(&value); err != nil {
			return err
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return errors.New("null field")
		}
		fields[key] = value
	}
	if _, err := shape.Token(); err != nil {
		return err
	}
	required := []string{"date", "timezone", "latitude", "longitude", "profile"}
	if _, ok := dst.(*chart.Request); ok {
		required = append(required, "time_status")
	}
	for _, key := range required {
		if _, ok := fields[key]; !ok {
			return errors.New("missing required field")
		}
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
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
			s.durationSum[kind].Add(uint64(elapsed))
			s.cfg.Logger.Debug().Str("request_id", w.Header().Get("X-Request-ID")).Str("method", r.Method).Str("route", route).Int("status", status).Bool("cache_hit", cached).Dur("duration_ms", elapsed).Msg("request")
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
			key := panchang.Version + ":" + s.engine.Version() + ":" + ephemeris.SwissCommit + ":" + ephemeris.DataVersion + ":" + hex.EncodeToString(digest[:])
			data, cached, err = s.cache.Do(ctx, key, func() ([]byte, error) {
				s.cacheMisses.Add(1)
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
		_, _ = fmt.Fprintf(w, "kripa_request_duration_seconds_sum{route=%q} %g\n", route, float64(s.durationSum[k].Load())/1e9)
	}
	_, _ = fmt.Fprintf(w, "kripa_cache_hits_total %d\nkripa_overload_rejections_total %d\nkripa_inflight %d\n", s.cacheHits.Load(), s.rejected.Load(), len(s.slots))
	_, _ = fmt.Fprintf(w, "kripa_cache_misses_total %d\nkripa_calculation_duration_seconds_sum %g\nkripa_calculation_duration_seconds_count %d\nkripa_native_queue_wait_seconds_total %g\nkripa_native_failures_total %d\n", s.cacheMisses.Load(), float64(s.calcNanos.Load())/1e9, s.calcCount.Load(), float64(s.queueNanos.Load())/1e9, s.nativeFailures.Load())
}

// Measure time waiting for the native gate separately from time inside a session.
// Counts include readiness probes; no input values enter metric labels.
type measuredProvider struct {
	ephemeris.Provider
	server *Server
}

func (p measuredProvider) WithSession(ctx context.Context, fn func(ephemeris.Session) error) error {
	timed := func(session ephemeris.Session) error {
		start := time.Now()
		defer func() { p.server.calcCount.Add(1); p.server.calcNanos.Add(uint64(time.Since(start))) }()
		return fn(session)
	}
	var err error
	if provider, ok := p.Provider.(interface {
		WithSessionTiming(context.Context, func(ephemeris.Session) error, func(time.Duration)) error
	}); ok {
		err = provider.WithSessionTiming(ctx, timed, func(wait time.Duration) { p.server.queueNanos.Add(uint64(wait)) })
	} else {
		err = p.Provider.WithSession(ctx, timed)
	}
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ephemeris.ErrNoEvent) {
		p.server.nativeFailures.Add(1)
	}
	return err
}
