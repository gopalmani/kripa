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
	}{{chartJSON, "", "application/json", 401}, {chartJSON, "secret", "text/plain", 415}, {chartJSON + `{}`, "secret", "application/json", 400}, {`{"extra":1}`, "secret", "application/json", 400}, {`{"date":"invalid"}`, "secret", "application/json", 422}, {strings.Repeat(" ", 9000) + chartJSON, "secret", "application/json", 413}, {chartJSON, "secret", "application/json", 503}} {
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
	h := New(p, Config{MaxInflight: 64, CacheEntries: 8, Timeout: 10 * time.Second, Logger: zerolog.Nop()}).Handler()
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
	for i := 0; i < 20; i++ {
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
