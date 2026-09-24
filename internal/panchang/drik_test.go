package panchang

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

// One explicitly located, minute-rounded astronomy page, not calendar parity.
func TestDrikAstronomySanity(t *testing.T) {
	p := native(t)
	data, err := os.ReadFile("../../docs/fixtures/drik-astronomy.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Request
		Tolerance float64 `json:"tolerance_seconds"`
	}
	var expected map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	result, err := Calculate(context.Background(), p, fixture.Request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Moonrise == nil || result.Moonset == nil {
		t.Fatal("missing lunar events")
	}
	for key, actual := range map[string]time.Time{"sunrise": result.Sunrise, "sunset": result.Sunset, "next_sunrise": result.NextSunrise, "moonrise": *result.Moonrise, "moonset": *result.Moonset} {
		var reference time.Time
		if err := json.Unmarshal(expected[key], &reference); err != nil {
			t.Fatal(err)
		}
		delta := actual.Sub(reference).Seconds()
		t.Logf("%s signed difference from minute reference: %.0f seconds", key, delta)
		if math.Abs(delta) > fixture.Tolerance {
			t.Errorf("%s outside reference sanity tolerance: %.0fs", key, delta)
		}
	}
}
