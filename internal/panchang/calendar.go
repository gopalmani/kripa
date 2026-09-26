// SPDX-License-Identifier: AGPL-3.0-or-later
package panchang

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gopalmani/kripa/internal/ephemeris"
)

// CalendarVersion identifies the lunar-month, muhurta and observance rules.
// Observances are rule previews: they follow common published conventions but
// have not received specialist religious review for every tradition or region.
const CalendarVersion = "kripa-calendar-rules-v1"

type LunarMonth struct {
	Amanta     string    `json:"amanta"`
	Purnimanta string    `json:"purnimanta"`
	Adhika     bool      `json:"adhika"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
}
type Samvat struct {
	Vikram int `json:"vikram"`
	Shaka  int `json:"shaka"`
}
type Muhurta struct {
	// Abhijit is omitted on Wednesdays under the common convention.
	Abhijit *Window `json:"abhijit"`
	Brahma  Window  `json:"brahma"`
}
type Sankranti struct {
	Rashi string    `json:"rashi"`
	At    time.Time `json:"at"`
}
type Observance struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Category  string   `json:"category"`
	Scope     string   `json:"scope"`
	Tradition string   `json:"tradition"`
	Regions   []string `json:"regions"`
	Tithi     string   `json:"tithi,omitempty"`
	Rule      string   `json:"rule"`
}
type Calendar struct {
	Version      string       `json:"version"`
	ReviewStatus string       `json:"review_status"`
	LunarMonth   LunarMonth   `json:"lunar_month"`
	Samvat       Samvat       `json:"samvat"`
	SunRashi     string       `json:"sun_rashi"`
	MoonRashi    []Segment    `json:"moon_rashi"`
	Sankranti    *Sankranti   `json:"sankranti"`
	Muhurta      Muhurta      `json:"muhurta"`
	Observances  []Observance `json:"observances"`
	Conventions  []string     `json:"conventions"`
}

// Month names are indexed from Chaitra. Rashis are indexed from Mesha.
var months = []string{"Chaitra", "Vaishakha", "Jyeshtha", "Ashadha", "Shravana", "Bhadrapada", "Ashvina", "Kartika", "Margashirsha", "Pausha", "Magha", "Phalguna"}
var rashis = []string{"Mesha", "Vrishabha", "Mithuna", "Karka", "Simha", "Kanya", "Tula", "Vrishchika", "Dhanu", "Makara", "Kumbha", "Meena"}

const synodicMonth = 29.530588853

// elongation returns the signed Moon-Sun angle in (-180, 180]; it rises through
// zero at each new moon.
func elongation(s ephemeris.Session, jd float64) (float64, error) {
	a, err := angle(s, "tithi", jd)
	if err != nil {
		return 0, err
	}
	if a > 180 {
		a -= 360
	}
	return a, nil
}

// newMoon returns the conjunction nearest to guess (searched within ±3 days).
func newMoon(ctx context.Context, s ephemeris.Session, guess float64) (float64, error) {
	lo, hi := guess-3, guess+3
	flo, err := elongation(s, lo)
	if err != nil {
		return 0, err
	}
	fhi, err := elongation(s, hi)
	if err != nil {
		return 0, err
	}
	if !(flo < 0 && fhi > 0) {
		return 0, fmt.Errorf("new moon not bracketed")
	}
	for i := 0; i < 50 && (hi-lo)*86400 > 0.1; i++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		mid := (lo + hi) / 2
		v, err := elongation(s, mid)
		if err != nil {
			return 0, err
		}
		if v >= 0 {
			hi = mid
		} else {
			lo = mid
		}
	}
	return hi, nil
}

// bracketingNewMoons returns the conjunctions before and after jd.
func bracketingNewMoons(ctx context.Context, s ephemeris.Session, jd float64) (float64, float64, error) {
	a, err := angle(s, "tithi", jd)
	if err != nil {
		return 0, 0, err
	}
	prev, err := newMoon(ctx, s, jd-a/360*synodicMonth)
	if err != nil {
		return 0, 0, err
	}
	if prev > jd {
		if prev, err = newMoon(ctx, s, prev-synodicMonth); err != nil {
			return 0, 0, err
		}
	}
	next, err := newMoon(ctx, s, prev+synodicMonth)
	if err != nil {
		return 0, 0, err
	}
	return prev, next, nil
}

func sunSign(s ephemeris.Session, jd float64) (int, error) {
	sun, err := s.Position(jd, 0, true)
	if err != nil {
		return 0, err
	}
	return int(math.Floor(sun.Longitude/30)) % 12, nil
}

// lunarMonth names the Amanta month from the Sun's sidereal sign at the new
// moon that begins it. A month without a Sankranti is Adhika.
func lunarMonth(ctx context.Context, s ephemeris.Session, jd float64) (index int, prev, next float64, adhika bool, err error) {
	prev, next, err = bracketingNewMoons(ctx, s, jd)
	if err != nil {
		return
	}
	a, err := sunSign(s, prev)
	if err != nil {
		return
	}
	b, err := sunSign(s, next)
	if err != nil {
		return
	}
	return (a + 1) % 12, prev, next, a == b, nil
}

func calendar(ctx context.Context, s ephemeris.Session, out *Result, rise, set, nextRise float64, loc *time.Location, weekday int) (Calendar, error) {
	cal := Calendar{Version: CalendarVersion, ReviewStatus: "rule_preview", Conventions: []string{
		"Lunar month named from the sidereal solar sign at the beginning new moon (Amanta); Adhika when no Sankranti occurs",
		"Purnimanta month equals the Amanta month in Shukla paksha and the following month in Krishna paksha",
		"Vikram and Shaka samvat change at Chaitra Shukla Pratipada (Chaitradi); some regions begin at Kartika",
		"Abhijit: 8th of 15 daytime muhurtas, omitted on Wednesday; Brahma: 14th of 15 night muhurtas before sunrise",
		"Observances use the tithi prevailing at a decisive time (sunrise, midday, pradosh or nishita) and are not regional rulings",
		"Kshaya/vriddhi tithi, Vaishnava Ekadashi, Bhadra and parana rules are not applied; confirm with a local Panchang and Purohit",
	}}
	idx, prev, next, adhika, err := lunarMonth(ctx, s, rise)
	if err != nil {
		return cal, err
	}
	krishna := out.Tithi[0].Index > 15
	pIdx := idx
	if krishna {
		pIdx = (idx + 1) % 12
	}
	cal.LunarMonth = LunarMonth{Amanta: months[idx], Purnimanta: months[pIdx], Adhika: adhika, StartsAt: timestamp(s.Time(prev), loc), EndsAt: timestamp(s.Time(next), loc)}

	// The Chaitra that began this lunar year started idx nija months earlier.
	yearStart := s.Time(prev - float64(idx)*synodicMonth).In(loc)
	cal.Samvat = Samvat{Vikram: yearStart.Year() + 57, Shaka: yearStart.Year() - 78}

	sr, err := sunSign(s, rise)
	if err != nil {
		return cal, err
	}
	cal.SunRashi = rashis[sr]
	sn, err := sunSign(s, nextRise)
	if err != nil {
		return cal, err
	}
	if sn != sr {
		at, err := sankranti(ctx, s, rise, nextRise, sn)
		if err != nil {
			return cal, err
		}
		cal.Sankranti = &Sankranti{Rashi: rashis[sn], At: timestamp(s.Time(at), loc)}
	}
	if cal.MoonRashi, err = transitions(ctx, s, "moon_rashi", rise, nextRise, loc); err != nil {
		return cal, err
	}

	day := (set - rise) / 15
	if weekday != 3 {
		cal.Muhurta.Abhijit = &Window{timestamp(s.Time(rise+7*day), loc).Round(time.Second), timestamp(s.Time(rise+8*day), loc).Round(time.Second)}
	}
	prevSet, err := s.RiseSet(rise-1, 0, out.Latitude, out.Longitude, false)
	if err != nil {
		return cal, err
	}
	if prevSet >= rise || rise-prevSet > 1 {
		return cal, ephemeris.ErrNoEvent
	}
	night := (rise - prevSet) / 15
	cal.Muhurta.Brahma = Window{timestamp(s.Time(rise-2*night), loc).Round(time.Second), timestamp(s.Time(rise-night), loc).Round(time.Second)}

	cal.Observances = observances(out, idx, adhika, weekday, prevSet, rise, set, nextRise, s, cal.Sankranti)
	return cal, nil
}

func sankranti(ctx context.Context, s ephemeris.Session, lo, hi float64, sign int) (float64, error) {
	for i := 0; i < 50 && (hi-lo)*86400 > 0.1; i++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		mid := (lo + hi) / 2
		v, err := sunSign(s, mid)
		if err != nil {
			return 0, err
		}
		if v == sign {
			hi = mid
		} else {
			lo = mid
		}
	}
	return hi, nil
}

type decisive int

const (
	atSunrise decisive = iota
	atMidday
	atPradosh
	atNishita
)

type rule struct {
	id, name, category, scope, tradition string
	regions                              []string
	month                                int // Amanta month index; -1 for every month
	tithi                                int // 1-30
	when                                 decisive
}

var allIndia = []string{"All India"}

// Rules use Amanta month indices (Chaitra=0). Krishna-paksha festivals are
// therefore listed under the Amanta month that precedes their Purnimanta name.
var rules = []rule{
	{"vasant_panchami", "Vasant Panchami", "festival", "widely_observed", "Smarta", allIndia, 10, 5, atSunrise},
	{"maha_shivaratri", "Maha Shivaratri", "festival", "widely_observed", "Shaiva", allIndia, 10, 29, atNishita},
	{"holika_dahan", "Holika Dahan", "festival", "widely_observed", "Smarta", []string{"North and West India"}, 11, 15, atPradosh},
	{"holi", "Holi", "festival", "widely_observed", "Smarta", []string{"North and West India"}, 11, 16, atSunrise},
	{"chaitra_navaratri", "Chaitra Navaratri begins", "festival", "widely_observed", "Shakta", allIndia, 0, 1, atSunrise},
	{"ugadi", "Ugadi", "regional", "regional", "Regional new year", []string{"Karnataka", "Andhra Pradesh", "Telangana"}, 0, 1, atSunrise},
	{"gudi_padwa", "Gudi Padwa", "regional", "regional", "Regional new year", []string{"Maharashtra", "Goa"}, 0, 1, atSunrise},
	{"rama_navami", "Rama Navami", "jayanti", "widely_observed", "Vaishnava", allIndia, 0, 9, atMidday},
	{"hanuman_jayanti", "Hanuman Jayanti", "jayanti", "regional", "North Indian convention", []string{"North India"}, 0, 15, atSunrise},
	{"akshaya_tritiya", "Akshaya Tritiya", "festival", "widely_observed", "Smarta", allIndia, 1, 3, atSunrise},
	{"parashurama_jayanti", "Parashurama Jayanti", "jayanti", "widely_observed", "Vaishnava", allIndia, 1, 3, atPradosh},
	{"narasimha_jayanti", "Narasimha Jayanti", "jayanti", "widely_observed", "Vaishnava", allIndia, 1, 14, atPradosh},
	{"guru_purnima", "Guru Purnima", "festival", "widely_observed", "Smarta", allIndia, 3, 15, atSunrise},
	{"nag_panchami", "Nag Panchami", "festival", "regional", "Smarta", []string{"North and West India"}, 4, 5, atSunrise},
	{"raksha_bandhan", "Raksha Bandhan", "festival", "widely_observed", "Smarta", allIndia, 4, 15, atSunrise},
	{"krishna_janmashtami", "Krishna Janmashtami", "jayanti", "widely_observed", "Smarta", allIndia, 4, 23, atNishita},
	{"ganesh_chaturthi", "Ganesh Chaturthi", "festival", "widely_observed", "Smarta", allIndia, 5, 4, atMidday},
	{"anant_chaturdashi", "Anant Chaturdashi", "festival", "widely_observed", "Smarta", allIndia, 5, 14, atSunrise},
	{"pitru_paksha", "Pitru Paksha begins", "panchang_event", "widely_observed", "Smarta", allIndia, 5, 16, atSunrise},
	{"sarva_pitru_amavasya", "Sarva Pitru Amavasya", "panchang_event", "widely_observed", "Smarta", allIndia, 5, 30, atSunrise},
	{"sharad_navaratri", "Sharad Navaratri begins", "festival", "widely_observed", "Shakta", allIndia, 6, 1, atSunrise},
	{"durga_ashtami", "Durga Ashtami", "festival", "widely_observed", "Shakta", allIndia, 6, 8, atSunrise},
	{"maha_navami", "Maha Navami", "festival", "widely_observed", "Shakta", allIndia, 6, 9, atSunrise},
	{"vijayadashami", "Vijayadashami", "festival", "widely_observed", "Smarta", allIndia, 6, 10, atMidday},
	{"sharad_purnima", "Sharad Purnima", "festival", "widely_observed", "Smarta", allIndia, 6, 15, atPradosh},
	{"valmiki_jayanti", "Valmiki Jayanti", "jayanti", "widely_observed", "Smarta", allIndia, 6, 15, atSunrise},
	{"karva_chauth", "Karva Chauth", "vrat", "regional", "North Indian convention", []string{"North India"}, 6, 19, atPradosh},
	{"dhanteras", "Dhanteras", "festival", "widely_observed", "Smarta", allIndia, 6, 28, atPradosh},
	{"naraka_chaturdashi", "Naraka Chaturdashi", "festival", "widely_observed", "Smarta", allIndia, 6, 29, atSunrise},
	{"diwali", "Diwali · Lakshmi Puja", "festival", "widely_observed", "Smarta", allIndia, 6, 30, atPradosh},
	{"govardhan_puja", "Govardhan Puja", "festival", "regional", "Vaishnava", []string{"North India"}, 7, 1, atSunrise},
	{"bhai_dooj", "Bhai Dooj", "festival", "widely_observed", "Smarta", allIndia, 7, 2, atMidday},
	{"chhath_puja", "Chhath Puja", "regional", "regional", "Regional", []string{"Bihar", "Jharkhand", "Eastern Uttar Pradesh"}, 7, 6, atSunrise},
	{"kartika_purnima", "Kartika Purnima · Dev Deepawali", "festival", "widely_observed", "Smarta", allIndia, 7, 15, atSunrise},
	{"vivah_panchami", "Vivah Panchami", "regional", "regional", "Vaishnava", []string{"North India", "Nepal"}, 8, 5, atSunrise},
	{"dattatreya_jayanti", "Dattatreya Jayanti", "jayanti", "regional", "Smarta", []string{"Maharashtra", "Karnataka", "Gujarat"}, 8, 15, atPradosh},
}

// Ekadashi names by Purnimanta month [Krishna, Shukla].
var ekadashis = [12][2]string{
	{"Papamochani", "Kamada"}, {"Varuthini", "Mohini"}, {"Apara", "Nirjala"}, {"Yogini", "Devshayani"},
	{"Kamika", "Shravana Putrada"}, {"Aja", "Parsva"}, {"Indira", "Papankusha"}, {"Rama", "Devutthana"},
	{"Utpanna", "Mokshada"}, {"Saphala", "Pausha Putrada"}, {"Shattila", "Jaya"}, {"Vijaya", "Amalaki"},
}

func observances(out *Result, month int, adhika bool, weekday int, prevSet, rise, set, nextRise float64, s ephemeris.Session, sk *Sankranti) []Observance {
	tithiAt := func(jd float64) int {
		a, err := angle(s, "tithi", jd)
		if err != nil {
			return 0
		}
		return int(math.Floor(a/12)) + 1
	}
	night := (nextRise - set) / 15
	// Each decisive time is a window; a tithi qualifies if it prevails at
	// either edge. Nishita is the 8th of 15 night muhurtas.
	at := map[decisive][]int{
		atSunrise: {out.Tithi[0].Index},
		atMidday:  {tithiAt((rise + set) / 2)},
		atPradosh: {tithiAt(set), tithiAt(set + 2*night)},
		atNishita: {tithiAt(set + 7*night), tithiAt(set + 8*night)},
	}
	prevNight := (rise - prevSet) / 15
	previous := map[decisive][]int{
		atPradosh: {tithiAt(prevSet), tithiAt(prevSet + 2*prevNight)},
		atNishita: {tithiAt(prevSet + 7*prevNight), tithiAt(prevSet + 8*prevNight)},
	}
	has := func(when decisive, tithi int) bool {
		for _, v := range at[when] {
			if v == tithi {
				return true
			}
		}
		return false
	}
	obs := []Observance{}
	add := func(o Observance) { obs = append(obs, o) }
	if !adhika {
		for _, r := range rules {
			// A night-window tithi that never touches the window on either
			// evening is observed on the day it prevails at sunrise.
			fallback := (r.when == atPradosh || r.when == atNishita) && out.Tithi[0].Index == r.tithi &&
				!contains(previous[r.when], r.tithi) && !has(r.when, r.tithi)
			if r.month == month && (has(r.when, r.tithi) || fallback) {
				add(Observance{r.id, r.name, r.category, r.scope, r.tradition, r.regions, limbName("tithi", r.tithi), ruleText(r.when, month, r.tithi)})
			}
		}
	}
	sunrise := out.Tithi[0].Index
	if sunrise == 11 || sunrise == 26 {
		p := month
		k := 1
		if sunrise == 26 {
			p, k = (month+1)%12, 0
		}
		name := ekadashis[p][k] + " Ekadashi"
		if adhika {
			name = map[int]string{1: "Padmini Ekadashi", 0: "Parama Ekadashi"}[k]
		}
		add(Observance{"ekadashi", name, "vrat", "widely_observed", "Smarta (sunrise tithi)", allIndia, limbName("tithi", sunrise), "Ekadashi prevailing at sunrise; Vaishnava observance may differ"})
	}
	if pd := at[atPradosh][0]; has(atPradosh, 13) || has(atPradosh, 28) {
		if pd != 13 && pd != 28 {
			pd = at[atPradosh][1]
		}
		name := map[int]string{0: "Ravi Pradosh Vrat", 1: "Soma Pradosh Vrat", 2: "Bhauma Pradosh Vrat", 6: "Shani Pradosh Vrat"}[weekday]
		if name == "" {
			name = "Pradosh Vrat"
		}
		add(Observance{"pradosh", name, "vrat", "widely_observed", "Shaiva", allIndia, limbName("tithi", pd), "Trayodashi prevailing at pradosh (after sunset)"})
	}
	if has(atPradosh, 19) && !(month == 6 && !adhika) {
		add(Observance{"sankashti_chaturthi", "Sankashti Chaturthi", "vrat", "widely_observed", "Smarta", allIndia, limbName("tithi", 19), "Krishna Chaturthi prevailing after sunset (moonrise)"})
	}
	if has(atMidday, 4) && !(month == 5 && !adhika) {
		add(Observance{"vinayaka_chaturthi", "Vinayaka Chaturthi", "vrat", "widely_observed", "Smarta", allIndia, limbName("tithi", 4), "Shukla Chaturthi prevailing at midday"})
	}
	if has(atNishita, 29) && !(month == 10 && !adhika) {
		add(Observance{"masik_shivaratri", "Masik Shivaratri", "vrat", "widely_observed", "Shaiva", allIndia, limbName("tithi", 29), "Krishna Chaturdashi prevailing during nishita"})
	}
	switch sunrise {
	case 15:
		add(Observance{"purnima", months[month] + " Purnima", "panchang_event", "widely_observed", "Panchang", allIndia, "Purnima", "Purnima prevailing at sunrise"})
	case 30:
		add(Observance{"amavasya", months[month] + " Amavasya", "panchang_event", "widely_observed", "Panchang", allIndia, "Amavasya", "Amavasya prevailing at sunrise"})
	}
	if sk != nil {
		name := sk.Rashi + " Sankranti"
		category := "panchang_event"
		switch sk.Rashi {
		case "Makara":
			name, category = "Makar Sankranti", "festival"
		case "Mesha":
			name = "Mesha Sankranti · Solar New Year"
		}
		add(Observance{"sankranti", name, category, "widely_observed", "Solar", allIndia, "", "Sidereal solar ingress between this sunrise and the next"})
	}
	return obs
}

func ruleText(when decisive, month, tithi int) string {
	at := map[decisive]string{atSunrise: "sunrise", atMidday: "midday", atPradosh: "pradosh", atNishita: "nishita"}[when]
	return fmt.Sprintf("%s %s prevailing at %s (Amanta month)", months[month], limbName("tithi", tithi), at)
}

func contains(values []int, v int) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}
