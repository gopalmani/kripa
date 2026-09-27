package panchang

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
)

func year(t *testing.T, y int, lat, lon float64) YearResult {
	t.Helper()
	v, err := Year(context.Background(), native(t), YearRequest{Year: y, Timezone: "Asia/Kolkata", Latitude: lat, Longitude: lon, Profile: Profile})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// Compares computed dates with published New Delhi calendars for 2025-2027.
// Known convention differences are listed; anything else fails.
func TestFestivalReferenceDelhi(t *testing.T) {
	data, err := os.ReadFile("../../docs/fixtures/festival-reference-delhi.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		Years map[string]map[string][]string `json:"years"`
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	known := map[string]string{
		// 2026: Bhadra covers the Purnima evening of 2 March until 05:28 and
		// Purnima ends before the next pradosh. The classical rule (Holika in
		// Bhadra puchha) gives 2 March; published calendars move it to 3 March,
		// the day of a total lunar eclipse. Holi follows Holika Dahan.
		"2026 holika_dahan": "Bhadra/eclipse-year convention",
		"2026 holi":         "follows Holika Dahan",
		// Ashtami prevails at nishita only on 24 August; published Smarta
		// calendars keep 25 August (Ashtami at sunrise, Rohini that day).
		"2027 krishna_janmashtami": "Rohini/udaya Ashtami preference not modelled",
	}
	total, matched := 0, 0
	for ys, want := range ref.Years {
		y, _ := strconv.Atoi(ys)
		got := map[string][]string{}
		for _, o := range year(t, y, 28.6139, 77.209).Festivals {
			got[o.ID] = append(got[o.ID], o.Date)
		}
		ids := make([]string, 0, len(want))
		for id := range want {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			total++
			ok := false
			for _, d := range got[id] {
				for _, w := range want[id] {
					ok = ok || d == w
				}
			}
			if ok {
				matched++
				continue
			}
			if reason, listed := known[ys+" "+id]; listed {
				t.Logf("known difference %s %s: got %v, reference %v (%s)", ys, id, got[id], want[id], reason)
				continue
			}
			t.Errorf("%s %s: got %v, reference %v", ys, id, got[id], want[id])
		}
	}
	t.Logf("%d/%d reference dates matched", matched, total)
}

func TestYearIsFastAndOrdered(t *testing.T) {
	started := time.Now()
	v := year(t, 2026, 13.0827, 80.2707)
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("year scan took %v", elapsed)
	}
	for i := 1; i < len(v.Festivals); i++ {
		if v.Festivals[i].Date < v.Festivals[i-1].Date {
			t.Fatal("festivals not ordered by date")
		}
	}
	if v.ReviewStatus != "rule_preview" || v.CalendarVersion != CalendarVersion {
		t.Fatalf("metadata %+v", v)
	}
}

// No festival is reported on nearby days (vriddhi tithi or a nakshatra that
// spans two sunrises) anywhere in India's main regions.
func TestNoRepeatedFestivals(t *testing.T) {
	for y := 2025; y <= 2030; y++ {
		for _, loc := range [][2]float64{{28.6139, 77.209}, {13.0827, 80.2707}, {22.5726, 88.3639}, {9.9312, 76.2673}} {
			last := map[string]time.Time{}
			for _, o := range year(t, y, loc[0], loc[1]).Festivals {
				if o.Recurring {
					continue
				}
				date, _ := time.Parse("2006-01-02", o.Date)
				if prev, ok := last[o.ID]; ok && date.Sub(prev) < 20*24*time.Hour {
					t.Errorf("%d %v: %s on %s and %s", y, loc, o.ID, prev.Format("2006-01-02"), o.Date)
				}
				last[o.ID] = date
			}
		}
	}
}
