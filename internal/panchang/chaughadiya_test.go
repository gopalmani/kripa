package panchang

import (
	"testing"
	"time"
)

// Published weekday tables (first and last period of each day and night).
func TestChaughadiyaSequence(t *testing.T) {
	day := [][]string{
		{"Udveg", "Char", "Labh", "Amrit", "Kaal", "Shubh", "Rog", "Udveg"},
		{"Amrit", "Kaal", "Shubh", "Rog", "Udveg", "Char", "Labh", "Amrit"},
		{"Rog", "Udveg", "Char", "Labh", "Amrit", "Kaal", "Shubh", "Rog"},
		{"Labh", "Amrit", "Kaal", "Shubh", "Rog", "Udveg", "Char", "Labh"},
		{"Shubh", "Rog", "Udveg", "Char", "Labh", "Amrit", "Kaal", "Shubh"},
		{"Char", "Labh", "Amrit", "Kaal", "Shubh", "Rog", "Udveg", "Char"},
		{"Kaal", "Shubh", "Rog", "Udveg", "Char", "Labh", "Amrit", "Kaal"},
	}
	night := [][]string{
		{"Shubh", "Amrit", "Char", "Rog", "Kaal", "Labh", "Udveg", "Shubh"},
		{"Char", "Rog", "Kaal", "Labh", "Udveg", "Shubh", "Amrit", "Char"},
		{"Kaal", "Labh", "Udveg", "Shubh", "Amrit", "Char", "Rog", "Kaal"},
		{"Udveg", "Shubh", "Amrit", "Char", "Rog", "Kaal", "Labh", "Udveg"},
		{"Amrit", "Char", "Rog", "Kaal", "Labh", "Udveg", "Shubh", "Amrit"},
		{"Rog", "Kaal", "Labh", "Udveg", "Shubh", "Amrit", "Char", "Rog"},
		{"Labh", "Udveg", "Shubh", "Amrit", "Char", "Rog", "Kaal", "Labh"},
	}
	rise := time.Date(2026, 9, 27, 6, 10, 0, 0, time.UTC)
	set := rise.Add(12 * time.Hour)
	next := rise.Add(24*time.Hour + time.Minute)
	for weekday := range 7 {
		c := chaughadiya(rise, set, next, weekday)
		for i := range 8 {
			if c.Day[i].Name != day[weekday][i] || c.Night[i].Name != night[weekday][i] {
				t.Fatalf("weekday %d period %d: got %s/%s", weekday, i, c.Day[i].Name, c.Night[i].Name)
			}
		}
		if !c.Day[0].Start.Equal(rise) || !c.Day[7].End.Equal(set) || !c.Night[0].Start.Equal(set) || !c.Night[7].End.Equal(next) {
			t.Fatal("periods must cover sunrise to next sunrise")
		}
		for i := 1; i < 8; i++ {
			if !c.Day[i].Start.Equal(c.Day[i-1].End) || !c.Night[i].Start.Equal(c.Night[i-1].End) {
				t.Fatal("periods must be contiguous")
			}
		}
		if c.Day[0].Nature == "" || c.Day[1].End.Sub(c.Day[1].Start) != 90*time.Minute {
			t.Fatal("unexpected period")
		}
	}
}
