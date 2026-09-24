package api

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestProductionTelemetryPrivacy(t *testing.T) {
	var logs bytes.Buffer
	h := New(unavailable{}, Config{Token: "private-test-token", Logger: zerolog.New(&logs).Level(zerolog.InfoLevel)}).Handler()
	for _, token := range []string{"invalid-token", "private-test-token"} {
		r := httptest.NewRequest("POST", "/v1/charts?secret=query-marker", strings.NewReader(chartJSON))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Request-ID", "untrusted-user-marker")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Header().Get("X-Powered-By") != "KRIPA" {
			t.Fatal("missing attribution")
		}
	}
	if !strings.Contains(logs.String(), `"route":"charts"`) {
		t.Fatal("request not logged at production level")
	}
	for _, secret := range []string{"2000-01-01", "51.5", "private-test-token", "invalid-token", "query-marker", "untrusted-user-marker", "192.0.2.1"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("sensitive marker logged: %s", secret)
		}
	}
	r := httptest.NewRequest("GET", "/metrics", nil)
	r.Header.Set("Authorization", "Bearer private-test-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "kripa_auth_rejections_total 1") {
		t.Fatal("auth rejection not counted")
	}
}
