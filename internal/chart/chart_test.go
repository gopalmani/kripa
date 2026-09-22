package chart

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gopalmani/kripa/internal/ephemeris"
	"math"
	"os"
	"reflect"
	"testing"
)

func request() Request {
	return Request{Date: "2000-01-01", Time: "12:00", Timezone: "UTC", TimeStatus: "exact", Profile: "western_tropical_v1", Latitude: 51.5, Longitude: 0}
}
func TestResolve(t *testing.T) {
	r := request()
	instant, timed, err := r.Resolve()
	if err != nil || !timed || instant.Format("2006-01-02T15:04:05Z") != "2000-01-01T12:00:00Z" {
		t.Fatalf("resolve: %v %v", instant, err)
	}
	r.Timezone = "America/New_York"
	for _, c := range []struct{ date, clock string }{{"2024-03-10", "02:30"}, {"2024-11-03", "01:30"}} {
		r.Date, r.Time = c.date, c.clock
		if _, _, err := r.Resolve(); err == nil {
			t.Fatalf("accepted DST gap/overlap %v", c)
		}
	}
	r = request()
	r.TimeStatus = "unknown"
	r.Time = ""
	_, timed, err = r.Resolve()
	if err != nil || timed {
		t.Fatal("unknown time handling")
	}
	r.TimeStatus = "exact"
	if _, _, err = r.Resolve(); err == nil {
		t.Fatal("accepted missing exact time")
	}
}
func TestHouseWrap(t *testing.T) {
	c := []float64{0, 350, 20, 50, 80, 110, 140, 170, 200, 230, 260, 290, 320}
	for lon, want := range map[float64]int{355: 1, 10: 1, 25: 2, 345: 12} {
		if got := houseFor(lon, c); got != want {
			t.Fatalf("%v: got %d want %d", lon, got, want)
		}
	}
}
func native(t testing.TB) *ephemeris.Native {
	t.Helper()
	path := os.Getenv("SWISS_EPHEMERIS_PATH")
	if path == "" {
		t.Skip("set SWISS_EPHEMERIS_PATH for native tests")
	}
	p, err := ephemeris.New(path)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestNativeChart(t *testing.T) {
	p := native(t)
	r := request()
	c, err := Calculate(context.Background(), p, r)
	if err != nil {
		t.Fatal(err)
	}
	if c.SunSign != "Capricorn" || len(c.Planets) != 13 || len(c.Houses) != 12 || c.AscendantSign == nil {
		t.Fatalf("incomplete chart: %+v", c)
	}
	r.Time = ""
	r.TimeStatus = "unknown"
	c, err = Calculate(context.Background(), p, r)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Houses) != 0 || c.AscendantSign != nil {
		t.Fatal("invented houses for unknown birth time")
	}
}
func BenchmarkChart(b *testing.B) {
	p := native(b)
	r := request()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Calculate(context.Background(), p, r); err != nil {
			b.Fatal(err)
		}
	}
}

// Expected results were produced by the actual supplied Astrel SwissEngine in
// an isolated module. This is upstream compatibility, not independent accuracy.
func TestAstrelReferenceCharts(t *testing.T) {
	p := native(t)
	data, err := os.ReadFile("../../docs/fixtures/astrel-charts.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Tolerance float64 `json:"angular_tolerance_degrees"`
		Cases     []struct {
			Request
			Expected Chart `json:"expected"`
		}
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for i, tc := range fixture.Cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			tc.Request.Profile = "western_tropical_v1"
			got, err := Calculate(context.Background(), p, tc.Request)
			if err != nil {
				t.Fatal(err)
			}
			want := tc.Expected
			if got.Metadata.HouseSystem != want.Metadata.HouseSystem || got.Metadata.DataQuality != want.Metadata.DataQuality || got.SunSign != want.SunSign || len(got.Planets) != len(want.Planets) || len(got.Houses) != len(want.Houses) || len(got.Aspects) != len(want.Aspects) {
				t.Fatal("chart structure diverged")
			}
			near := func(a, b float64) {
				t.Helper()
				if math.Abs(a-b) > fixture.Tolerance {
					t.Errorf("angular difference %g > %g", math.Abs(a-b), fixture.Tolerance)
				}
			}
			for i, a := range got.Planets {
				b := want.Planets[i]
				if a.Planet != b.Planet || a.Sign != b.Sign || a.Retrograde != b.Retrograde || !reflect.DeepEqual(a.House, b.House) {
					t.Errorf("planet %s diverged", a.Planet)
				}
				near(a.Degree, b.Degree)
			}
			for i, a := range got.Houses {
				b := want.Houses[i]
				if a.Sign != b.Sign || a.House != b.House {
					t.Error("house diverged")
				}
				near(a.Degree, b.Degree)
			}
			for i, a := range got.Aspects {
				b := want.Aspects[i]
				if a.PlanetA != b.PlanetA || a.PlanetB != b.PlanetB || a.Aspect != b.Aspect {
					t.Error("aspect diverged")
				}
				near(a.Orb, b.Orb)
			}
			if !reflect.DeepEqual(got.AscendantSign, want.AscendantSign) || !reflect.DeepEqual(got.MCSign, want.MCSign) {
				t.Error("angle sign diverged")
			}
			if got.AscendantDegree != nil {
				near(*got.AscendantDegree, *want.AscendantDegree)
				near(*got.MCDegree, *want.MCDegree)
			}
		})
	}
}
func TestNonHourDSTAndLocalZone(t *testing.T) {
	r := request()
	r.Timezone = "Australia/Lord_Howe"
	for _, tc := range []struct{ date, clock string }{{"2024-04-07", "01:45"}, {"2024-10-06", "02:15"}} {
		r.Date, r.Time = tc.date, tc.clock
		if _, _, err := r.Resolve(); err == nil {
			t.Fatal("accepted half-hour DST gap/overlap")
		}
	}
	r = request()
	r.Timezone = "Local"
	if _, _, err := r.Resolve(); err == nil {
		t.Fatal("accepted host-dependent zone")
	}
}
