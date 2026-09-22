package ephemeris

import (
	"context"
	"errors"
	"math"
	"os"
	"sync"
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

func TestRejectOverrideAndFallback(t *testing.T) {
	t.Setenv("SE_EPHE_PATH", t.TempDir())
	if _, err := New(os.Getenv("SWISS_EPHEMERIS_PATH")); err == nil {
		t.Fatal("accepted native path override")
	}
	t.Setenv("SE_EPHE_PATH", "")
	// Bypass startup deliberately to exercise return-flag fallback detection.
	p := &Native{path: t.TempDir()}
	err := p.WithSession(context.Background(), func(s Session) error { _, err := s.Position(2451545, 0, false); return err })
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("accepted Moshier fallback: %v", err)
	}
}

func TestRepeatedMixedSessions(t *testing.T) {
	path := os.Getenv("SWISS_EPHEMERIS_PATH")
	if path == "" {
		t.Skip("set SWISS_EPHEMERIS_PATH")
	}
	p, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	calculate := func(sidereal bool) (Position, error) {
		var result Position
		err := p.WithSession(context.Background(), func(s Session) error { var err error; result, err = s.Position(2451545, 1, sidereal); return err })
		return result, err
	}
	tropical, err := calculate(false)
	if err != nil {
		t.Fatal(err)
	}
	sidereal, err := calculate(true)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(tropical.Longitude-sidereal.Longitude) < 20 {
		t.Fatal("sidereal flag ineffective")
	}
	var wg sync.WaitGroup
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sid := i%2 == 1
			want := tropical
			if sid {
				want = sidereal
			}
			for j := 0; j < 3; j++ {
				got, err := calculate(sid)
				if err != nil || got != want {
					t.Errorf("mixed session diverged: %+v %v", got, err)
				}
			}
		}(i)
	}
	wg.Wait()
}
