// SPDX-License-Identifier: AGPL-3.0-or-later
package panchang

import (
	"context"
	"errors"
	"fmt"
	"github.com/gopalmani/kripa/internal/ephemeris"
	"math"
	"time"
)

const Version = "kripa-panchang-lahiri-v1"
const Profile = "lahiri_upper_limb_v1"

type Request struct {
	Date      string  `json:"date"`
	Timezone  string  `json:"timezone"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Profile   string  `json:"profile"`
}
type Segment struct {
	Index      int       `json:"index"`
	Name       string    `json:"name"`
	ActiveFrom time.Time `json:"active_from"`
	EndsAt     time.Time `json:"ends_at"`
}
type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
type Result struct {
	Date               string     `json:"date"`
	Timezone           string     `json:"timezone"`
	Latitude           float64    `json:"latitude"`
	Longitude          float64    `json:"longitude"`
	Profile            string     `json:"profile"`
	CalculationVersion string     `json:"calculation_version"`
	EphemerisVersion   string     `json:"ephemeris_version"`
	ReviewStatus       string     `json:"review_status"`
	Sunrise            time.Time  `json:"sunrise"`
	Sunset             time.Time  `json:"sunset"`
	NextSunrise        time.Time  `json:"next_sunrise"`
	Moonrise           *time.Time `json:"moonrise"`
	Moonset            *time.Time `json:"moonset"`
	Vaar               string     `json:"vaar"`
	Paksha             string     `json:"paksha_at_sunrise"`
	Tithi              []Segment  `json:"tithi"`
	Nakshatra          []Segment  `json:"nakshatra"`
	Yoga               []Segment  `json:"yoga"`
	Karana             []Segment  `json:"karana"`
	RahuKalam          Window     `json:"rahu_kalam"`
	Yamaganda          Window     `json:"yamaganda"`
	Gulika             Window     `json:"gulika"`
	Calendar           Calendar   `json:"calendar"`
	Unsupported        []string   `json:"unsupported"`
	Conventions        []string   `json:"conventions"`
}

func (r Request) Validate() (time.Time, *time.Location, error) {
	if r.Profile != Profile {
		return time.Time{}, nil, fmt.Errorf("profile must be %s", Profile)
	}
	if !ephemeris.ValidCoordinate(r.Latitude, r.Longitude) {
		return time.Time{}, nil, fmt.Errorf("invalid coordinates")
	}
	loc, err := time.LoadLocation(r.Timezone)
	if err != nil || (r.Timezone == "" || r.Timezone == "Local") {
		return time.Time{}, nil, fmt.Errorf("invalid IANA timezone")
	}
	date, err := time.ParseInLocation("2006-01-02", r.Date, loc)
	if err != nil || date.Format("2006-01-02") != r.Date || date.Year() < 1900 || date.Year() > 2099 {
		return time.Time{}, nil, fmt.Errorf("date must be YYYY-MM-DD within 1900-2099")
	}
	return date, loc, nil
}
func Calculate(ctx context.Context, p ephemeris.Provider, r Request) (Result, error) {
	date, loc, err := r.Validate()
	if err != nil {
		return Result{}, err
	}
	out := Result{Date: r.Date, Timezone: r.Timezone, Latitude: r.Latitude, Longitude: r.Longitude, Profile: Profile, CalculationVersion: Version, EphemerisVersion: p.Version(), ReviewStatus: "astronomical_preview", Unsupported: []string{"kshaya_masa", "regional_calendar_variants", "vrat_parana", "personalized_muhurta"}, Conventions: []string{"Lahiri sidereal zodiac", "Apparent geocentric ecliptic positions", "Upper-limb sunrise with refraction", "Sea-level observer; pressure 1013.25 hPa; temperature 15 C", "Panchang day: sunrise to next sunrise", "Moonrise/moonset: requested civil day; null means no event", "Transitions solved numerically to 0.1 seconds; timestamps rounded to nearest second", "Calendar and religious review pending"}}
	err = p.WithSession(ctx, func(s ephemeris.Session) error {
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
		if rise >= end {
			return ephemeris.ErrNoEvent
		}
		set, err := s.RiseSet(rise, 0, r.Latitude, r.Longitude, false)
		if err != nil {
			return err
		}
		nextRise, err := s.RiseSet(end, 0, r.Latitude, r.Longitude, true)
		if err != nil {
			return err
		}
		if !(rise < set && set < nextRise && nextRise-rise < 2) {
			return ephemeris.ErrNoEvent
		}
		out.Sunrise = timestamp(s.Time(rise), loc)
		out.Sunset = timestamp(s.Time(set), loc)
		out.NextSunrise = timestamp(s.Time(nextRise), loc)
		for _, ev := range []struct {
			rise bool
			dst  **time.Time
		}{{true, &out.Moonrise}, {false, &out.Moonset}} {
			if err := ctx.Err(); err != nil {
				return err
			}
			jd, err := s.RiseSet(start, 1, r.Latitude, r.Longitude, ev.rise)
			if errors.Is(err, ephemeris.ErrNoEvent) {
				continue
			}
			if err != nil {
				return err
			}
			if jd >= start && jd < end {
				t := timestamp(s.Time(jd), loc)
				*ev.dst = &t
			}
		}
		for _, limb := range []struct {
			kind string
			dst  *[]Segment
		}{{"tithi", &out.Tithi}, {"nakshatra", &out.Nakshatra}, {"yoga", &out.Yoga}, {"karana", &out.Karana}} {
			segments, err := transitions(ctx, s, limb.kind, rise, nextRise, loc)
			if err != nil {
				return err
			}
			*limb.dst = segments
		}
		out.Paksha = "Shukla"
		if out.Tithi[0].Index > 15 {
			out.Paksha = "Krishna"
		}
		weekday := int(out.Sunrise.In(loc).Weekday())
		out.Vaar = weekdays[weekday]
		out.RahuKalam = period(out.Sunrise, out.Sunset, []int{8, 2, 7, 5, 6, 4, 3}[weekday])
		out.Yamaganda = period(out.Sunrise, out.Sunset, []int{5, 4, 3, 2, 1, 7, 6}[weekday])
		out.Gulika = period(out.Sunrise, out.Sunset, []int{7, 6, 5, 4, 3, 2, 1}[weekday])
		out.Calendar, err = calendar(ctx, s, &out, rise, set, nextRise, loc, weekday)
		return err
	})
	return out, err
}
func period(rise, set time.Time, n int) Window {
	d := set.Sub(rise) / 8
	return Window{rise.Add(time.Duration(n-1) * d).Round(time.Second), rise.Add(time.Duration(n) * d).Round(time.Second)}
}
func angle(s ephemeris.Session, kind string, jd float64) (float64, error) {
	moon, err := s.Position(jd, 1, true)
	if err != nil {
		return 0, err
	}
	if kind == "nakshatra" || kind == "moon_rashi" {
		return moon.Longitude, nil
	}
	sun, err := s.Position(jd, 0, true)
	if err != nil {
		return 0, err
	}
	if kind == "yoga" {
		return ephemeris.Normalize(moon.Longitude + sun.Longitude), nil
	}
	return ephemeris.Normalize(moon.Longitude - sun.Longitude), nil
}

// nextBoundary unwraps only forward motion within a two-day bracket. Sun/Moon
// phase, Moon longitude and their sum are monotonic over this supported window.
func nextBoundary(ctx context.Context, s ephemeris.Session, kind string, start, width float64) (float64, int, error) {
	a, err := angle(s, kind, start)
	if err != nil {
		return 0, 0, err
	}
	index := int(math.Floor(a / width))
	distance := float64(index+1)*width - a
	span := 2.0
	if kind == "moon_rashi" {
		span = 3
	}
	lo, hi := start, start+span
	high, err := angle(s, kind, hi)
	if err != nil {
		return 0, 0, err
	}
	if ephemeris.Normalize(high-a) < distance {
		return 0, 0, fmt.Errorf("transition not bracketed")
	}
	for i := 0; i < 40 && (hi-lo)*86400 > 0.1; i++ {
		if err := ctx.Err(); err != nil {
			return 0, 0, err
		}
		mid := (lo + hi) / 2
		v, err := angle(s, kind, mid)
		if err != nil {
			return 0, 0, err
		}
		if ephemeris.Normalize(v-a) >= distance {
			hi = mid
		} else {
			lo = mid
		}
	}
	return hi, index + 1, nil
}
func transitions(ctx context.Context, s ephemeris.Session, kind string, start, end float64, loc *time.Location) ([]Segment, error) {
	width := 12.0
	switch kind {
	case "nakshatra", "yoga":
		width = 360.0 / 27
	case "karana":
		width = 6
	case "moon_rashi":
		width = 30
	}
	out := make([]Segment, 0, 4)
	cursor := start
	active := timestamp(s.Time(start), loc)
	for i := 0; i < 8; i++ {
		boundary, index, err := nextBoundary(ctx, s, kind, cursor, width)
		if err != nil {
			return nil, err
		}
		finish := timestamp(s.Time(boundary), loc)
		out = append(out, Segment{index, limbName(kind, index), active, finish})
		if boundary >= end {
			return out, nil
		}
		active = finish
		cursor = boundary + 1e-7
	}
	return nil, fmt.Errorf("transition limit exceeded")
}

var weekdays = []string{"Ravivara", "Somavara", "Mangalavara", "Budhavara", "Guruvara", "Shukravara", "Shanivara"}
var tithis = []string{"Pratipada", "Dvitiya", "Tritiya", "Chaturthi", "Panchami", "Shashthi", "Saptami", "Ashtami", "Navami", "Dashami", "Ekadashi", "Dvadashi", "Trayodashi", "Chaturdashi"}
var nakshatras = []string{"Ashwini", "Bharani", "Krittika", "Rohini", "Mrigashira", "Ardra", "Punarvasu", "Pushya", "Ashlesha", "Magha", "Purva Phalguni", "Uttara Phalguni", "Hasta", "Chitra", "Swati", "Vishakha", "Anuradha", "Jyeshtha", "Mula", "Purva Ashadha", "Uttara Ashadha", "Shravana", "Dhanishtha", "Shatabhisha", "Purva Bhadrapada", "Uttara Bhadrapada", "Revati"}
var yogas = []string{"Vishkambha", "Priti", "Ayushman", "Saubhagya", "Shobhana", "Atiganda", "Sukarma", "Dhriti", "Shula", "Ganda", "Vriddhi", "Dhruva", "Vyaghata", "Harshana", "Vajra", "Siddhi", "Vyatipata", "Variyana", "Parigha", "Shiva", "Siddha", "Sadhya", "Shubha", "Shukla", "Brahma", "Indra", "Vaidhriti"}

func limbName(kind string, index int) string {
	switch kind {
	case "moon_rashi":
		return rashis[(index-1)%12]
	case "nakshatra":
		return nakshatras[(index-1)%27]
	case "yoga":
		return yogas[(index-1)%27]
	case "karana":
		switch index {
		case 1:
			return "Kimstughna"
		case 58:
			return "Shakuni"
		case 59:
			return "Chatushpada"
		case 60:
			return "Naga"
		}
		return []string{"Bava", "Balava", "Kaulava", "Taitila", "Gara", "Vanija", "Vishti"}[(index-2)%7]
	default:
		if index == 15 {
			return "Purnima"
		}
		if index == 30 {
			return "Amavasya"
		}
		paksha := "Shukla"
		if index > 15 {
			paksha = "Krishna"
		}
		return paksha + " " + tithis[(index-1)%15]
	}
}

// RFC 3339 cannot represent historical timezone offsets containing seconds.
// Use UTC for those instants so JSON serialization cannot silently shift them.
func timestamp(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	_, offset := local.Zone()
	if offset%60 != 0 {
		return t.UTC()
	}
	return local
}
