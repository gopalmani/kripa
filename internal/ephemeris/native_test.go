package ephemeris

import (
	"context"
	"math"
	"os"
	"testing"
	"time"
)

func TestNativeRoundtripAndJ2000(t *testing.T) {
	path := os.Getenv("SWISS_EPHEMERIS_PATH")
	if path == "" {
		t.Skip("set SWISS_EPHEMERIS_PATH")
	}
	p, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	err = p.WithSession(context.Background(), func(s Session) error {
		instant := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
		jd, err := s.JulianDay(instant)
		if err != nil {
			return err
		}
		if math.Abs(s.Time(jd).Sub(instant).Seconds()) > 1 {
			t.Fatal("time conversion")
		}
		pos, err := s.Position(jd, 0, false)
		if err != nil {
			return err
		}
		if math.Abs(pos.Longitude-280.36892) > .01 {
			t.Fatalf("J2000 sun longitude %f", pos.Longitude)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestMissingDataFails(t *testing.T) {
	if _, err := New(t.TempDir()); err == nil {
		t.Fatal("accepted absent data")
	}
}
