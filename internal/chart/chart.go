// SPDX-License-Identifier: AGPL-3.0-or-later
// Chart assembly adapted from AstroMatch contributors, copyright 2024-2026.
package chart

import (
	"context"
	"fmt"
	"github.com/gopalmani/kripa/internal/ephemeris"
	"time"
)

const Version = "kripa-tropical-v1"

type Request struct {
	Date       string  `json:"date"`
	Time       string  `json:"time,omitempty"`
	Timezone   string  `json:"timezone"`
	TimeStatus string  `json:"time_status"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Profile    string  `json:"profile"`
}
type body struct {
	name string
	id   int
}

var bodies = []body{{"Sun", 0}, {"Moon", 1}, {"Mercury", 2}, {"Venus", 3}, {"Mars", 4}, {"Jupiter", 5}, {"Saturn", 6}, {"Uranus", 7}, {"Neptune", 8}, {"Pluto", 9}, {"North Node", 11}, {"Chiron", 15}}

// Resolve rejects DST gaps and overlaps rather than silently choosing one birth instant.
func (r Request) Resolve() (time.Time, bool, error) {
	if r.Profile != "western_tropical_v1" {
		return time.Time{}, false, fmt.Errorf("profile must be western_tropical_v1")
	}
	if !ephemeris.ValidCoordinate(r.Latitude, r.Longitude) {
		return time.Time{}, false, fmt.Errorf("invalid coordinates")
	}
	loc, err := time.LoadLocation(r.Timezone)
	if err != nil || (r.Timezone == "" || r.Timezone == "Local") {
		return time.Time{}, false, fmt.Errorf("invalid IANA timezone")
	}
	date, err := time.Parse("2006-01-02", r.Date)
	if err != nil || date.Year() < 1900 || date.Year() > 2099 {
		return time.Time{}, false, fmt.Errorf("date must be YYYY-MM-DD within 1900-2099")
	}
	timed := r.TimeStatus != "unknown"
	if r.TimeStatus != "exact" && r.TimeStatus != "approximate" && r.TimeStatus != "unknown" {
		return time.Time{}, false, fmt.Errorf("time_status must be exact, approximate or unknown")
	}
	clock := "12:00:00"
	if timed {
		clock = r.Time
		if len(clock) == 5 {
			clock += ":00"
		}
	} else if r.Time != "" {
		return time.Time{}, false, fmt.Errorf("time must be omitted when unknown")
	}
	wall := r.Date + "T" + clock
	parsed, err := time.Parse("2006-01-02T15:04:05", wall)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("time must be HH:MM or HH:MM:SS")
	}
	// Collect the zone offsets on either side of transitions, including non-hour changes.
	offsets := map[int]bool{}
	for _, delta := range []time.Duration{-48 * time.Hour, -24 * time.Hour, 0, 24 * time.Hour, 48 * time.Hour} {
		_, off := parsed.Add(delta).In(loc).Zone()
		offsets[off] = true
	}
	var candidates []time.Time
	for off := range offsets {
		t := parsed.Add(-time.Duration(off) * time.Second)
		if t.In(loc).Format("2006-01-02T15:04:05") == wall {
			candidates = append(candidates, t)
		}
	}
	if len(candidates) != 1 {
		return time.Time{}, false, fmt.Errorf("local time is nonexistent or ambiguous; choose an unambiguous time")
	}
	return candidates[0].UTC(), timed, nil
}
func Calculate(ctx context.Context, p ephemeris.Provider, r Request) (Chart, error) {
	instant, timed, err := r.Resolve()
	if err != nil {
		return Chart{}, err
	}
	var out Chart
	err = p.WithSession(ctx, func(s ephemeris.Session) error {
		jd, err := s.JulianDay(instant)
		if err != nil {
			return err
		}
		longitudes := map[string]float64{}
		retro := map[string]bool{}
		planets := make([]PlanetPosition, 0, 13)
		for _, b := range bodies {
			if err := ctx.Err(); err != nil {
				return err
			}
			pos, err := s.Position(jd, b.id, false)
			if err != nil {
				return err
			}
			longitudes[b.name] = pos.Longitude
			retro[b.name] = pos.Speed < 0
			planets = append(planets, planetPosition(b.name, pos.Longitude, pos.Speed < 0))
		}
		longitudes["South Node"] = normalizeDegree(longitudes["North Node"] + 180)
		planets = append(planets, planetPosition("South Node", longitudes["South Node"], retro["North Node"]))
		out = Chart{Metadata: ChartMetadata{CalculationVersion: Version, EphemerisVersion: p.Version(), HouseSystem: "unavailable", DataQuality: "standard"}, Planets: planets, Houses: []HousePosition{}, Aspects: calculateAspects(longitudes), SunSign: zodiac(longitudes["Sun"])}
		if !timed {
			return nil
		}
		cusps, angles, err := s.Houses(jd, r.Latitude, r.Longitude, 'P')
		system := "placidus"
		if err != nil {
			cusps, angles, err = s.Houses(jd, r.Latitude, r.Longitude, 'W')
			system = "whole_sign"
		}
		if err != nil {
			return err
		}
		out.Metadata.HouseSystem = system
		if r.TimeStatus == "exact" {
			out.Metadata.DataQuality = "high"
		}
		for i := 1; i <= 12; i++ {
			out.Houses = append(out.Houses, HousePosition{i, zodiac(cusps[i]), signDegree(cusps[i])})
		}
		for i := range out.Planets {
			h := houseFor(longitudes[out.Planets[i].Planet], cusps[:])
			out.Planets[i].House = &h
		}
		asc, mc := zodiac(angles[0]), zodiac(angles[1])
		ascD, mcD := signDegree(angles[0]), signDegree(angles[1])
		out.AscendantSign = &asc
		out.MCSign = &mc
		out.AscendantDegree = &ascD
		out.MCDegree = &mcD
		return nil
	})
	return out, err
}
