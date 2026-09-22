package api

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gopalmani/kripa/internal/ephemeris"
	"github.com/rs/zerolog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const chartJSON = `{"date":"2000-01-01","time":"12:00","timezone":"UTC","time_status":"exact","latitude":51.5,"longitude":0,"profile":"western_tropical_v1"}`
const panchangJSON = `{"date":"2026-09-21","timezone":"Asia/Kolkata","latitude":12.9716,"longitude":77.5946,"profile":"lahiri_upper_limb_v1"}`

type unavailable struct{}

func (unavailable) Version() string { return "test" }
func (unavailable) WithSession(ctx context.Context, fn func(ephemeris.Session) error) error {
	return ephemeris.ErrUnavailable
}
func TestValidationAndAuth(t *testing.T) {
	var logs bytes.Buffer
	s := New(unavailable{}, Config{Token: "secret", CacheEntries: 8, Logger: zerolog.New(&logs).Level(zerolog.DebugLevel)}).Handler()
	for _, tc := range []struct {
		body, token, ctype string
		status             int
	}{{chartJSON, "", "application/json", 401}, {chartJSON, "secret", "text/plain", 415}, {chartJSON + `{}`, "secret", "application/json", 400}, {`{"extra":1}`, "secret", "application/json", 400}, {strings.Replace(chartJSON, "2000-01-01", "invalid", 1), "secret", "application/json", 422}, {strings.Repeat(" ", 9000) + chartJSON, "secret", "application/json", 413}, {chartJSON, "secret", "application/json", 503}} {
		r := httptest.NewRequest(http.MethodPost, "/v1/charts", strings.NewReader(tc.body))
		r.Header.Set("Content-Type", tc.ctype)
		r.Header.Set("Authorization", "Bearer "+tc.token)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Errorf("got %d want %d: %s", w.Code, tc.status, w.Body.String())
		}
	}
	if strings.Contains(logs.String(), "2000-01-01") || strings.Contains(logs.String(), "secret") {
		t.Fatal("sensitive inputs in logs")
	}
	for _, path := range []string{"/health/live", "/v1/meta"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatal(path)
		}
	}
}
func TestNativeConcurrentChartAndPanchang(t *testing.T) {
	path := os.Getenv("SWISS_EPHEMERIS_PATH")
	if path == "" {
		t.Skip("set SWISS_EPHEMERIS_PATH")
	}
	p, err := ephemeris.New(path)
	if err != nil {
		t.Fatal(err)
	}
	h := New(p, Config{MaxInflight: 64, CacheEntries: 0, Timeout: 10 * time.Second, Logger: zerolog.Nop()}).Handler()
	invoke := func(path, body string) []byte {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Errorf("%s: %d %s", path, w.Code, w.Body.String())
		}
		return w.Body.Bytes()
	}
	expectedChart := invoke("/v1/charts", chartJSON)
	expectedPanchang := invoke("/v1/panchang", panchangJSON)
	var wg sync.WaitGroup
	for i := 0; i < 60; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path, body, want := "/v1/charts", chartJSON, expectedChart
			if i%2 == 1 {
				path, body, want = "/v1/panchang", panchangJSON, expectedPanchang
			}
			got := invoke(path, body)
			if !bytes.Equal(got, want) {
				t.Error("cross-request contamination")
			}
		}(i)
	}
	wg.Wait()
}
func BenchmarkHTTPPanchangWarm(b *testing.B) {
	path := os.Getenv("SWISS_EPHEMERIS_PATH")
	if path == "" {
		b.Skip("set SWISS_EPHEMERIS_PATH")
	}
	p, err := ephemeris.New(path)
	if err != nil {
		b.Fatal(err)
	}
	h := New(p, Config{CacheEntries: 512, Logger: zerolog.Nop()}).Handler()
	call := func() {
		r := httptest.NewRequest("POST", "/v1/panchang", strings.NewReader(panchangJSON))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			b.Fatal(w.Body.String())
		}
		if !json.Valid(w.Body.Bytes()) {
			b.Fatal("invalid JSON")
		}
	}
	call()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call()
	}
}

func TestRequiredFieldsAndJSONShape(t *testing.T) {
	h := New(unavailable{}, Config{Logger: zerolog.Nop()}).Handler()
	for _, body := range []string{"null", "[]", strings.Replace(chartJSON, `"longitude":0`, `"longitude":0,"Longitude":1`, 1), strings.Replace(chartJSON, `"latitude":51.5,`, "", 1), strings.Replace(chartJSON, `"latitude":51.5`, `"latitude":null`, 1), strings.Replace(chartJSON, `"longitude":0`, `"longitude":0,"longitude":1`, 1)} {
		r := httptest.NewRequest("POST", "/v1/charts", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Errorf("shape accepted: %d", w.Code)
		}
	}
}
func TestReadinessAndRouting(t *testing.T) {
	h := New(unavailable{}, Config{Logger: zerolog.Nop()}).Handler()
	ids := map[string]bool{}
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/health/ready", 503}, {"GET", "/missing", 404}, {"GET", "/v1/charts", 405}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status || !json.Valid(w.Body.Bytes()) {
			t.Fatalf("route: %d %s", w.Code, w.Body.String())
		}
		id := w.Header().Get("X-Request-ID")
		if id == "" || ids[id] {
			t.Fatal("missing or repeated request ID")
		}
		ids[id] = true
	}
}

type blockedProvider struct{ entered, release chan struct{} }

func (blockedProvider) Version() string { return "blocked" }
func (p blockedProvider) WithSession(ctx context.Context, fn func(ephemeris.Session) error) error {
	select {
	case p.entered <- struct{}{}:
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.release:
		return ephemeris.ErrUnavailable
	}
}
func TestOverloadAndDeadline(t *testing.T) {
	p := blockedProvider{make(chan struct{}, 1), make(chan struct{})}
	h := New(p, Config{MaxInflight: 1, Timeout: 50 * time.Millisecond, Logger: zerolog.Nop()}).Handler()
	call := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/v1/charts", strings.NewReader(chartJSON))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- call() }()
	<-p.entered
	w := call()
	if w.Code != 503 || w.Header().Get("Retry-After") != "1" {
		t.Fatal("no overload response")
	}
	if w := <-done; w.Code != 504 {
		t.Fatalf("deadline status %d", w.Code)
	}
}

func TestNativeCacheInputIsolation(t *testing.T) {
	path := os.Getenv("SWISS_EPHEMERIS_PATH")
	if path == "" {
		t.Skip("set SWISS_EPHEMERIS_PATH")
	}
	p, err := ephemeris.New(path)
	if err != nil {
		t.Fatal(err)
	}
	h := New(p, Config{CacheEntries: 8, Logger: zerolog.Nop()}).Handler()
	bodies := []string{panchangJSON, strings.Replace(panchangJSON, "12.9716", "12.97161", 1), strings.Replace(panchangJSON, "77.5946", "77.59461", 1), strings.Replace(panchangJSON, "2026-09-21", "2026-09-22", 1), strings.Replace(panchangJSON, "Asia/Kolkata", "UTC", 1)}
	for pass := 0; pass < 2; pass++ {
		for _, body := range bodies {
			r := httptest.NewRequest("POST", "/v1/panchang", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			want := "MISS"
			if pass == 1 {
				want = "HIT"
			}
			if w.Code != 200 || w.Header().Get("X-Kripa-Cache") != want {
				t.Fatalf("cache isolation %d %s", w.Code, w.Header().Get("X-Kripa-Cache"))
			}
			var input, output map[string]any
			_ = json.Unmarshal([]byte(body), &input)
			_ = json.Unmarshal(w.Body.Bytes(), &output)
			for _, key := range []string{"date", "timezone", "latitude", "longitude", "profile"} {
				if input[key] != output[key] {
					t.Fatalf("cached response changed %s", key)
				}
			}
		}
	}
}
