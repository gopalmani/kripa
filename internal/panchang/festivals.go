// SPDX-License-Identifier: AGPL-3.0-or-later
package panchang

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gopalmani/kripa/internal/ephemeris"
)

// YearRequest asks for every observance in a Gregorian year at one place.
type YearRequest struct {
	Year      int     `json:"year"`
	Timezone  string  `json:"timezone"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Profile   string  `json:"profile"`
}

// Occurrence is an observance on a civil date (the Panchang day that begins
// at that date's sunrise).
type Occurrence struct {
	Date string `json:"date"`
	Observance
}

type YearResult struct {
	Year               int          `json:"year"`
	Timezone           string       `json:"timezone"`
	Latitude           float64      `json:"latitude"`
	Longitude          float64      `json:"longitude"`
	Profile            string       `json:"profile"`
	CalculationVersion string       `json:"calculation_version"`
	CalendarVersion    string       `json:"calendar_version"`
	EphemerisVersion   string       `json:"ephemeris_version"`
	ReviewStatus       string       `json:"review_status"`
	Conventions        []string     `json:"conventions"`
	Festivals          []Occurrence `json:"festivals"`
}

func (r YearRequest) Validate() (*time.Location, error) {
	if r.Year < 1900 || r.Year > 2099 {
		return nil, fmt.Errorf("year must be within 1900-2099")
	}
	_, loc, err := Request{Date: fmt.Sprintf("%04d-01-01", r.Year), Timezone: r.Timezone, Latitude: r.Latitude, Longitude: r.Longitude, Profile: r.Profile}.Validate()
	return loc, err
}

// Year scans each day of the year with only the quantities the observance
// rules need. Work is done in monthly sessions so a year request never holds
// the (serialised) ephemeris for long.
func Year(ctx context.Context, p ephemeris.Provider, r YearRequest) (YearResult, error) {
	loc, err := r.Validate()
	if err != nil {
		return YearResult{}, err
	}
	out := YearResult{Year: r.Year, Timezone: r.Timezone, Latitude: r.Latitude, Longitude: r.Longitude, Profile: Profile, CalculationVersion: Version, CalendarVersion: CalendarVersion, EphemerisVersion: p.Version(), ReviewStatus: "rule_preview", Festivals: []Occurrence{}, Conventions: []string{
		"Dates are civil dates of the Panchang day (sunrise to next sunrise) at the requested place",
		"Lunar festivals use the Amanta month and the tithi at the decisive time of each rule; festivals are not observed in an Adhika month",
		"Regional solar months begin by Tamil (ingress before sunset), Malayalam (before 3/5 of daytime), Bengali/Assamese (next civil day) or sunrise-day (Odia, Punjabi) rules",
		"Kshaya/vriddhi tithi, Vaishnava Ekadashi, Bhadra and parana rules are not applied; confirm with a local Panchang and Purohit",
		"Rule preview: not a religious ruling for any tradition or region",
	}}
	var lunar struct {
		prev, next float64
		index      int
		adhika     bool
	}
	for m := time.January; m <= time.December; m++ {
		err := p.WithSession(ctx, func(s ephemeris.Session) error {
			first := time.Date(r.Year, m, 1, 0, 0, 0, 0, loc)
			for date := first; date.Month() == m; date = date.AddDate(0, 0, 1) {
				if err := ctx.Err(); err != nil {
					return err
				}
				start, err := s.JulianDay(date.UTC())
				if err != nil {
					return err
				}
				end, err := s.JulianDay(date.AddDate(0, 0, 1).UTC())
				if err != nil {
					return err
				}
				rise, err := s.RiseSet(start, 0, r.Latitude, r.Longitude, true)
				if err != nil {
					return err
				}
				set, err := s.RiseSet(rise, 0, r.Latitude, r.Longitude, false)
				if err != nil {
					return err
				}
				nextRise, err := s.RiseSet(end, 0, r.Latitude, r.Longitude, true)
				if err != nil {
					return err
				}
				prevSet, err := s.RiseSet(rise-1, 0, r.Latitude, r.Longitude, false)
				if err != nil {
					return err
				}
				prevRise, err := s.RiseSet(rise-1.2, 0, r.Latitude, r.Longitude, true)
				if err != nil || prevRise >= rise {
					prevRise = 0
				}
				if rise >= end || !(prevSet < rise && rise < set && set < nextRise && nextRise-rise < 2) {
					return ephemeris.ErrNoEvent
				}
				if !(lunar.prev <= rise && rise < lunar.next) {
					if lunar.index, lunar.prev, lunar.next, lunar.adhika, err = lunarMonth(ctx, s, rise); err != nil {
						return err
					}
				}
				var sk *Sankranti
				a, err := sunSign(s, rise)
				if err != nil {
					return err
				}
				b, err := sunSign(s, nextRise)
				if err != nil {
					return err
				}
				if a != b {
					at, err := sankranti(ctx, s, rise, nextRise, b)
					if err != nil {
						return err
					}
					sk = &Sankranti{Rashi: rashis[b], At: timestamp(s.Time(at), loc)}
				}
				d := &day{prevRise: prevRise, ctx: ctx, s: s, lat: r.Latitude, lon: r.Longitude, loc: loc, date: date, rise: rise, set: set, nextRise: nextRise, prevSet: prevSet, weekday: int(date.Weekday()), month: lunar.index, adhika: lunar.adhika, sankranti: sk}
				d.tithi = d.tithiAt(rise)
				civil := date.Format("2006-01-02")
				for _, o := range observances(d) {
					out.Festivals = append(out.Festivals, Occurrence{Date: civil, Observance: o})
				}
			}
			return nil
		})
		if err != nil {
			return YearResult{}, err
		}
	}
	sort.SliceStable(out.Festivals, func(i, j int) bool { return out.Festivals[i].Date < out.Festivals[j].Date })
	return out, nil
}
