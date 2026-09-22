package panchang

import (
	"context"
	"encoding/json"
	"github.com/gopalmani/kripa/internal/ephemeris"
	"math"
	"os"
	"testing"
	"time"
)

// Independent linear-orbit fixture gives analytically known transition times.
type linearSession struct{}

func (linearSession) JulianDay(t time.Time) (float64, error) { return float64(t.Unix()) / 86400, nil }
func (linearSession) Time(jd float64) time.Time              { return time.Unix(int64(jd*86400), 0).UTC() }
func (linearSession) Position(jd float64, body int, _ bool) (ephemeris.Position, error) {
	if body == 0 {
		return ephemeris.Position{Longitude: ephemeris.Normalize(jd), Speed: 1}, nil
	}
	return ephemeris.Position{Longitude: ephemeris.Normalize(350 + 13*jd), Speed: 13}, nil
}
func (linearSession) Houses(float64, float64, float64, byte) ([13]float64, [10]float64, error) {
	return [13]float64{}, [10]float64{}, nil
}
func (linearSession) RiseSet(float64, int, float64, float64, bool) (float64, error) { return 0, nil }
func TestTransitionWrap(t *testing.T) {
	end, index, err := nextBoundary(context.Background(), linearSession{}, "tithi", 0, 12)
	if err != nil || index != 30 || math.Abs(end-10.0/12)*86400 > 0.11 {
		t.Fatalf("wrap transition %v %v %v", end, index, err)
	}
	segments, err := transitions(context.Background(), linearSession{}, "karana", 0, 1, time.UTC)
	if err != nil || len(segments) != 3 || segments[0].Index != 59 || segments[1].Index != 60 || segments[2].Index != 1 {
		t.Fatalf("multiple transitions: %+v %v", segments, err)
	}
}
func TestKaranaCycle(t *testing.T) {
	for i, want := range map[int]string{1: "Kimstughna", 2: "Bava", 8: "Vishti", 9: "Bava", 57: "Vishti", 58: "Shakuni", 59: "Chatushpada", 60: "Naga"} {
		if got := limbName("karana", i); got != want {
			t.Fatalf("%d got %s", i, got)
		}
	}
}
func TestCancelledSearch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := nextBoundary(ctx, linearSession{}, "tithi", 0, 12); err == nil {
		t.Fatal("ignored cancellation")
	}
}
func native(t testing.TB) *ephemeris.Native {
	t.Helper()
	path := os.Getenv("SWISS_EPHEMERIS_PATH")
	if path == "" {
		t.Skip("set SWISS_EPHEMERIS_PATH")
	}
	p, err := ephemeris.New(path)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestNativeAcrossIndia(t *testing.T) {
	p := native(t)
	for _, loc := range []struct{ lat, lon float64 }{{12.9716, 77.5946}, {28.6139, 77.209}, {19.076, 72.8777}, {22.5726, 88.3639}, {34.0837, 74.7973}, {8.0883, 77.5385}} {
		for _, date := range []string{"2024-04-08", "2024-06-21", "2024-12-21", "2026-09-21"} {
			r := Request{Date: date, Timezone: "Asia/Kolkata", Latitude: loc.lat, Longitude: loc.lon, Profile: Profile}
			v, err := Calculate(context.Background(), p, r)
			if err != nil {
				t.Fatalf("%v %s: %v", loc, date, err)
			}
			if !v.Sunset.After(v.Sunrise) || !v.NextSunrise.After(v.Sunset) {
				t.Fatal("invalid daylight window")
			}
			for _, segments := range [][]Segment{v.Tithi, v.Nakshatra, v.Yoga, v.Karana} {
				if len(segments) == 0 {
					t.Fatal("missing segments")
				}
				for i, s := range segments {
					if !s.EndsAt.After(s.ActiveFrom) {
						t.Fatal("nonpositive segment")
					}
					if i > 0 && !segments[i-1].EndsAt.Equal(s.ActiveFrom) {
						t.Fatal("transition gap")
					}
				}
			}
		}
	}
}

// Source-backed comparisons, not a complete Panchang accuracy certification.
func TestUSNOPhases(t *testing.T) {
	p := native(t)
	data, err := os.ReadFile("../../docs/fixtures/usno-phases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Latitude, Longitude float64
		Timezone            string
		Tolerance           float64 `json:"tolerance_seconds"`
		Cases               []struct {
			Date     string
			Index    int       `json:"tithi_index"`
			Expected time.Time `json:"expected_utc"`
		}
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, tc := range fixture.Cases {
		t.Run(tc.Date, func(t *testing.T) {
			v, err := Calculate(context.Background(), p, Request{Date: tc.Date, Timezone: fixture.Timezone, Latitude: fixture.Latitude, Longitude: fixture.Longitude, Profile: Profile})
			if err != nil {
				t.Fatal(err)
			}
			for _, segment := range v.Tithi {
				if segment.Index == tc.Index {
					if math.Abs(segment.EndsAt.Sub(tc.Expected).Seconds()) > fixture.Tolerance {
						t.Fatalf("phase %s expected %s", segment.EndsAt, tc.Expected)
					}
					return
				}
			}
			t.Fatal("missing phase segment")
		})
	}
}
func TestDaytimeWindows(t *testing.T) {
	rise := time.Date(2024, 1, 1, 6, 0, 0, 0, time.UTC)
	set := rise.Add(12 * time.Hour)
	for n := 1; n <= 8; n++ {
		w := period(rise, set, n)
		if w.Start != rise.Add(time.Duration(n-1)*90*time.Minute) || w.End.Sub(w.Start) != 90*time.Minute {
			t.Fatalf("window %d: %+v", n, w)
		}
	}
}

func BenchmarkPanchangCold(b *testing.B) {
	p := native(b)
	r := Request{Date: "2026-09-21", Timezone: "Asia/Kolkata", Latitude: 12.9716, Longitude: 77.5946, Profile: Profile}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Calculate(context.Background(), p, r); err != nil {
			b.Fatal(err)
		}
	}
}

func TestUSNOSunriseSunset(t *testing.T) {
	p := native(t)
	data, err := os.ReadFile("../../docs/fixtures/usno-sun.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Request
		Sunrise, Sunset time.Time
		Tolerance       float64 `json:"tolerance_seconds"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	fixture.Profile = Profile
	got, err := Calculate(context.Background(), p, fixture.Request)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Sunrise.Sub(fixture.Sunrise).Seconds()) > fixture.Tolerance || math.Abs(got.Sunset.Sub(fixture.Sunset).Seconds()) > fixture.Tolerance {
		t.Fatalf("USNO solar event discrepancy: %s %s", got.Sunrise, got.Sunset)
	}
}
