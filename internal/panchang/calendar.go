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
const CalendarVersion = "kripa-calendar-rules-v2"

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
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Scope       string   `json:"scope"`
	Tradition   string   `json:"tradition"`
	Regions     []string `json:"regions"`
	RegionCodes []string `json:"region_codes"`
	Communities []string `json:"communities"`
	Recurring   bool     `json:"recurring"`
	Tithi       string   `json:"tithi,omitempty"`
	Rule        string   `json:"rule"`
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
		"Regional solar months begin by Tamil (ingress before sunset), Malayalam (before 3/5 of daytime), Bengali/Assamese (next civil day) or sunrise-day (Odia, Punjabi) rules",
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

	muhurta := (set - rise) / 15
	if weekday != 3 {
		cal.Muhurta.Abhijit = &Window{timestamp(s.Time(rise+7*muhurta), loc).Round(time.Second), timestamp(s.Time(rise+8*muhurta), loc).Round(time.Second)}
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

	prevRise, err := s.RiseSet(rise-1.2, 0, out.Latitude, out.Longitude, true)
	if err != nil || prevRise >= rise {
		prevRise = 0
	}
	d := &day{prevRise: prevRise, ctx: ctx, s: s, lat: out.Latitude, lon: out.Longitude, loc: loc, date: civilDate(out.Sunrise, loc), rise: rise, set: set, nextRise: nextRise, prevSet: prevSet, weekday: weekday, tithi: out.Tithi[0].Index, month: idx, adhika: adhika, sankranti: cal.Sankranti}
	cal.Observances = observances(d)
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
	atAparahna // 4th of 5 daytime parts
	atPurvahna // end of the forenoon (2/5 of daytime)
)

// day carries what the observance rules need for one sunrise-to-sunrise day.
// Both the daily Panchang and the year scan build it.
type day struct {
	ctx                          context.Context
	s                            ephemeris.Session
	lat, lon                     float64
	loc                          *time.Location
	date                         time.Time // civil midnight in loc
	rise, set, nextRise, prevSet float64
	prevRise                     float64 // 0 when unknown
	weekday                      int
	tithi                        int // at sunrise
	month                        int // Amanta month index
	adhika                       bool
	sankranti                    *Sankranti
	solarStarts                  map[[2]int]time.Time // memo: [sign, rule] -> month start
}

func civilDate(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

func (d *day) tithiAt(jd float64) int {
	a, err := angle(d.s, "tithi", jd)
	if err != nil {
		return 0
	}
	return int(math.Floor(a/12)) + 1
}

func (d *day) nakshatraAt(jd float64) int {
	a, err := angle(d.s, "nakshatra", jd)
	if err != nil {
		return 0
	}
	return int(math.Floor(a/(360.0/27))) + 1
}

// solarStart returns the civil date on which the regional solar month of sign
// begins, for the ingress within about three weeks of this day.
func (d *day) solarStart(sign int, rule solarRule) (time.Time, bool) {
	key := [2]int{sign, int(rule)}
	if t, ok := d.solarStarts[key]; ok {
		return t, !t.IsZero()
	}
	t, ok := d.findSolarStart(sign, rule)
	if d.solarStarts == nil {
		d.solarStarts = map[[2]int]time.Time{}
	}
	if !ok {
		t = time.Time{}
	}
	d.solarStarts[key] = t
	return t, ok
}

func (d *day) findSolarStart(sign int, rule solarRule) (time.Time, bool) {
	sun, err := d.s.Position(d.rise, 0, true)
	if err != nil {
		return time.Time{}, false
	}
	delta := math.Mod(float64(sign)*30-sun.Longitude+540, 360) - 180
	if math.Abs(delta) > 24 {
		return time.Time{}, false
	}
	guess := d.rise + delta/0.9856
	lo, hi := guess-3, guess+3
	a, err := sunSign(d.s, lo)
	if err != nil || a != (sign+11)%12 {
		return time.Time{}, false
	}
	if b, err := sunSign(d.s, hi); err != nil || b != sign {
		return time.Time{}, false
	}
	at, err := sankranti(d.ctx, d.s, lo, hi, sign)
	if err != nil {
		return time.Time{}, false
	}
	instant := d.s.Time(at)
	civil := civilDate(instant, d.loc)
	if rule == solarBengali {
		return civil.AddDate(0, 0, 1), true
	}
	midnight, err := d.s.JulianDay(civil.UTC())
	if err != nil {
		return time.Time{}, false
	}
	rise, err := d.s.RiseSet(midnight, 0, d.lat, d.lon, true)
	if err != nil {
		return time.Time{}, false
	}
	set, err := d.s.RiseSet(rise, 0, d.lat, d.lon, false)
	if err != nil {
		return time.Time{}, false
	}
	switch rule {
	case solarSunriseDay, solarTamil:
		if at < set {
			return civil, true
		}
	case solarMalayalam:
		if at < rise+0.6*(set-rise) {
			return civil, true
		}
	}
	return civil.AddDate(0, 0, 1), true
}

// solarMonth returns the regional solar month in force on this day.
func (d *day) solarMonth(rule solarRule) (int, bool) {
	sign, err := sunSign(d.s, d.rise)
	if err != nil {
		return 0, false
	}
	// Near an ingress the new sign may not have begun its month yet, or the
	// next month may already have begun (Sunrise-day rule before sunrise).
	if next, ok := d.solarStart((sign+1)%12, rule); ok && !d.date.Before(next) {
		return (sign + 1) % 12, true
	}
	if start, ok := d.solarStart(sign, rule); ok && d.date.Before(start) {
		return (sign + 11) % 12, true
	}
	return sign, true
}

// Bhadra (Vishti karana) overlapping the pradosh window of the evening that
// starts at set.
func (d *day) bhadraAt(set, night float64) bool {
	for _, jd := range []float64{set, set + night, set + 2*night} {
		a, err := angle(d.s, "tithi", jd)
		if err == nil && limbName("karana", int(math.Floor(a/6))+1) == "Vishti" {
			return true
		}
	}
	return false
}

// Holika Dahan follows the Bhadra rule on the Phalguna Purnima evenings: the
// Purnima evening whose pradosh is free of Bhadra (Vishti karana); if Bhadra
// covers it and Purnima still prevails at the next pradosh, the next evening;
// otherwise the first evening, after Bhadra (in its puchha).
func (d *day) holikaChosen(offset int) bool {
	night := (d.nextRise - d.set) / 15
	evening := func(k int) float64 {
		if k == -1 {
			return d.prevSet
		}
		return d.set + float64(k)
	}
	purnima := func(k int) bool {
		set := evening(k)
		return d.tithiAt(set) == 15 || d.tithiAt(set+2*night) == 15
	}
	// choice returns the evening (relative index) chosen for a Purnima that
	// first prevails at evening k, or a sentinel when k does not qualify.
	choice := func(k int) int {
		if !purnima(k) {
			return 99
		}
		if d.bhadraAt(evening(k), night) && purnima(k+1) {
			return k + 1
		}
		return k
	}
	// Phalguna Purnima and the following Pratipada both fall in Amanta Phalguna.
	if d.month != 11 || d.adhika {
		return false
	}
	return (choice(offset) == offset && choice(offset-1) != offset-1) || choice(offset-1) == offset
}

// sankrantiDay returns the ingress observed today under the Sankranti-day
// rule (before sunset: that civil day; after sunset: the next).
func (d *day) sankrantiDay() *Sankranti {
	for _, jd := range []float64{d.rise - 1, d.rise, d.nextRise} {
		sign, err := sunSign(d.s, jd)
		if err != nil {
			continue
		}
		if start, ok := d.solarStart(sign, solarSunriseDay); ok && start.Equal(d.date) {
			return &Sankranti{Rashi: rashis[sign]}
		}
	}
	return nil
}

// followsAdhika reports whether this Nija month was preceded by an Adhika
// month of the same name.
func (d *day) followsAdhika() bool {
	idx, _, _, adhika, err := lunarMonth(d.ctx, d.s, d.rise-29.5)
	return err == nil && adhika && idx == d.month
}

// fridayBeforePurnima: a Friday in the Shukla half of the given Amanta month
// whose Purnima ends within the next seven days.
func (d *day) fridayBeforePurnima(month int) bool {
	if d.weekday != 5 || d.adhika || d.month != month || d.tithi > 15 {
		return false
	}
	a, err := angle(d.s, "tithi", d.rise+7)
	return err == nil && a >= 180
}

// Ekadashi names by Purnimanta month [Krishna, Shukla].
var ekadashis = [12][2]string{
	{"Papamochani", "Kamada"}, {"Varuthini", "Mohini"}, {"Apara", "Nirjala"}, {"Yogini", "Devshayani"},
	{"Kamika", "Shravana Putrada"}, {"Aja", "Parsva"}, {"Indira", "Papankusha"}, {"Rama", "Devutthana"},
	{"Utpanna", "Mokshada"}, {"Saphala", "Pausha Putrada"}, {"Shattila", "Jaya"}, {"Vijaya", "Amalaki"},
}

func observances(d *day) []Observance {
	night := (d.nextRise - d.set) / 15
	// Each decisive time is a window; a tithi qualifies if it prevails at
	// either edge. Nishita is the 8th of 15 night muhurtas.
	daytime := d.set - d.rise
	at := map[decisive][]int{
		atSunrise:  {d.tithi},
		atMidday:   {d.tithiAt((d.rise + d.set) / 2)},
		atPradosh:  {d.tithiAt(d.set), d.tithiAt(d.set + 2*night)},
		atNishita:  {d.tithiAt(d.set + 7*night), d.tithiAt(d.set + 8*night)},
		atAparahna: {d.tithiAt(d.rise + 0.6*daytime), d.tithiAt(d.rise + 0.8*daytime)},
		atPurvahna: {d.tithiAt(d.rise + 0.4*daytime)},
	}
	prevNight := (d.rise - d.prevSet) / 15
	previous := map[decisive][]int{
		atPradosh: {d.tithiAt(d.prevSet), d.tithiAt(d.prevSet + 2*prevNight)},
		atNishita: {d.tithiAt(d.prevSet + 7*prevNight), d.tithiAt(d.prevSet + 8*prevNight)},
	}
	if d.prevRise > 0 {
		prevDay := d.prevSet - d.prevRise
		previous[atSunrise] = []int{d.tithiAt(d.prevRise)}
		previous[atMidday] = []int{d.tithiAt((d.prevRise + d.prevSet) / 2)}
		previous[atAparahna] = []int{d.tithiAt(d.prevRise + 0.6*prevDay), d.tithiAt(d.prevRise + 0.8*prevDay)}
		previous[atPurvahna] = []int{d.tithiAt(d.prevRise + 0.4*prevDay)}
	}
	has := func(when decisive, tithi int) bool { return contains(at[when], tithi) }
	score := func(values []int, tithi int) int {
		n := 0
		for _, v := range values {
			if v == tithi {
				n++
			}
		}
		return n
	}
	// The following day's windows (next evening approximated one day later;
	// the error is minutes, not enough to change which tithi prevails).
	next := map[decisive][]int{
		atSunrise:  {d.tithiAt(d.nextRise)},
		atMidday:   {d.tithiAt((d.rise+d.set)/2 + 1)},
		atPradosh:  {d.tithiAt(d.set + 1), d.tithiAt(d.set + 1 + 2*night)},
		atNishita:  {d.tithiAt(d.set + 1 + 7*night), d.tithiAt(d.set + 1 + 8*night)},
		atAparahna: {d.tithiAt(d.rise + 1 + 0.6*daytime), d.tithiAt(d.rise + 1 + 0.8*daytime)},
		atPurvahna: {d.tithiAt(d.rise + 1 + 0.4*daytime)},
	}
	// When a tithi touches the decisive window on two consecutive days, the
	// day on which it covers more of the window is used; on a tie, the first.
	decisiveDay := func(when decisive, tithi int, later bool) bool {
		today := score(at[when], tithi)
		if later {
			// Some rules take the later of two qualifying days.
			return today > 0 && score(next[when], tithi) == 0
		}
		return today > 0 && score(previous[when], tithi) < today && score(next[when], tithi) <= today
	}
	// A tithi that begins after today's decisive time and ends before
	// tomorrow's never touches either window (kshaya); it is kept on the day
	// it begins.
	kshayaDay := func(when decisive, tithi int) bool {
		w, n := at[when], next[when]
		return len(w) > 0 && len(n) > 0 && w[len(w)-1] == (tithi+28)%30+1 && n[0] == tithi%30+1
	}
	firstSunrise := func(tithi int) bool { return d.tithi == tithi && !contains(previous[atSunrise], tithi) }
	nakshatraAt := func(when decisive) int {
		if when == atPradosh {
			return d.nakshatraAt(d.set)
		}
		return d.nakshatraAt(d.rise)
	}
	obs := []Observance{}
	add := func(o Observance) {
		o.Recurring = recurringIDs[o.ID]
		if o.RegionCodes == nil {
			o.RegionCodes = codesIndia
		}
		o.Communities = nonNil(o.Communities)
		obs = append(obs, o)
	}
	entry := func(f festival, tithi string) Observance {
		return Observance{ID: f.id, Name: f.name, Category: f.category, Scope: f.scope, Tradition: f.tradition, Regions: f.regions, RegionCodes: f.codes, Communities: f.communities, Tithi: tithi, Rule: f.ruleText()}
	}
	for _, f := range catalogue {
		switch f.kind {
		case lunarTithi:
			// A kshaya tithi (begins after sunrise and ends before the next)
			// never prevails at sunrise; it is observed on the day it begins.
			// A kshaya Pratipada belongs to the month that begins that day.
			month := d.month
			kshaya := kshayaDay(f.when, f.tithi)
			if f.later {
				// Later-day rules keep a tithi that misses both windows on the
				// day after it begins (the day it prevails at sunrise).
				p, w := previous[f.when], at[f.when]
				kshaya = len(p) > 0 && p[len(p)-1] == (f.tithi+28)%30+1 && w[0] == f.tithi%30+1
			}
			if kshaya && f.tithi == 1 {
				month = (d.month + 1) % 12
			}
			if (d.adhika && !f.adhika) || f.month != month {
				continue
			}
			// Festivals kept in an Adhika month are not repeated in the Nija month.
			if f.adhika && !d.adhika && d.followsAdhika() {
				continue
			}
			// A night-window tithi that never touches the window on either
			// evening is observed on the day it prevails at sunrise.
			fallback := (f.when == atPradosh || f.when == atNishita) && d.tithi == f.tithi &&
				!contains(previous[f.when], f.tithi) && !has(f.when, f.tithi)
			if decisiveDay(f.when, f.tithi, f.later) || fallback || kshaya {
				add(entry(f, limbName("tithi", f.tithi)))
			}
		case holikaDahan:
			if d.holikaChosen(0) {
				add(entry(f, "Purnima"))
			}
		case dayAfterHolika:
			if d.holikaChosen(-1) {
				add(entry(f, limbName("tithi", d.tithi)))
			}
		case solarDayOne:
			if start, ok := d.solarStart(f.month, f.solar); ok && d.date.Equal(start.AddDate(0, 0, f.offset)) {
				add(entry(f, ""))
			}
		case solarNakshatra:
			if nakshatraAt(f.when) != f.nakshatra {
				continue
			}
			if m, ok := d.solarMonth(f.solar); ok && m == f.month {
				add(entry(f, ""))
			}
		case solarTithi:
			if !decisiveDay(f.when, f.tithi, false) {
				continue
			}
			if m, ok := d.solarMonth(f.solar); ok && m == f.month {
				add(entry(f, limbName("tithi", f.tithi)))
			}
		case fridayBeforePurnima:
			if d.fridayBeforePurnima(f.month) {
				add(entry(f, limbName("tithi", d.tithi)))
			}
		}
	}
	sunrise := d.tithi
	if (sunrise == 11 || sunrise == 26) && firstSunrise(sunrise) {
		p := d.month
		k := 1
		if sunrise == 26 {
			p, k = (d.month+1)%12, 0
		}
		name := ekadashis[p][k] + " Ekadashi"
		if d.adhika {
			name = map[int]string{1: "Padmini Ekadashi", 0: "Parama Ekadashi"}[k]
		}
		add(Observance{ID: "ekadashi", Name: name, Category: "vrat", Scope: "widely_observed", Tradition: "Smarta (sunrise tithi)", Regions: allIndia, Tithi: limbName("tithi", sunrise), Rule: "Ekadashi prevailing at sunrise; Vaishnava observance may differ"})
	}
	if pd := at[atPradosh][0]; has(atPradosh, 13) || has(atPradosh, 28) {
		if pd != 13 && pd != 28 {
			pd = at[atPradosh][1]
		}
		name := map[int]string{0: "Ravi Pradosh Vrat", 1: "Soma Pradosh Vrat", 2: "Bhauma Pradosh Vrat", 6: "Shani Pradosh Vrat"}[d.weekday]
		if name == "" {
			name = "Pradosh Vrat"
		}
		add(Observance{ID: "pradosh", Name: name, Category: "vrat", Scope: "widely_observed", Tradition: "Shaiva", Regions: allIndia, Tithi: limbName("tithi", pd), Rule: "Trayodashi prevailing at pradosh (after sunset)"})
	}
	if has(atPradosh, 19) && !(d.month == 6 && !d.adhika) {
		add(Observance{ID: "sankashti_chaturthi", Name: "Sankashti Chaturthi", Category: "vrat", Scope: "widely_observed", Tradition: "Smarta", Regions: allIndia, Tithi: limbName("tithi", 19), Rule: "Krishna Chaturthi prevailing after sunset (moonrise)"})
	}
	if has(atMidday, 4) && !(d.month == 5 && !d.adhika) {
		add(Observance{ID: "vinayaka_chaturthi", Name: "Vinayaka Chaturthi", Category: "vrat", Scope: "widely_observed", Tradition: "Smarta", Regions: allIndia, Tithi: limbName("tithi", 4), Rule: "Shukla Chaturthi prevailing at midday"})
	}
	if has(atNishita, 29) && !(d.month == 10 && !d.adhika) {
		add(Observance{ID: "masik_shivaratri", Name: "Masik Shivaratri", Category: "vrat", Scope: "widely_observed", Tradition: "Shaiva", Regions: allIndia, Tithi: limbName("tithi", 29), Rule: "Krishna Chaturdashi prevailing during nishita"})
	}
	switch {
	case sunrise == 15 && firstSunrise(15):
		add(Observance{ID: "purnima", Name: months[d.month] + " Purnima", Category: "panchang_event", Scope: "widely_observed", Tradition: "Panchang", Regions: allIndia, Tithi: "Purnima", Rule: "Purnima prevailing at sunrise"})
	case sunrise == 30 && firstSunrise(30):
		add(Observance{ID: "amavasya", Name: months[d.month] + " Amavasya", Category: "panchang_event", Scope: "widely_observed", Tradition: "Panchang", Regions: allIndia, Tithi: "Amavasya", Rule: "Amavasya prevailing at sunrise"})
	}
	if sk := d.sankrantiDay(); sk != nil {
		name := sk.Rashi + " Sankranti"
		category, id := "panchang_event", "sankranti"
		switch sk.Rashi {
		case "Makara":
			name, category, id = "Makar Sankranti", "festival", "makar_sankranti"
		case "Mesha":
			name = "Mesha Sankranti · Solar New Year"
		}
		add(Observance{ID: id, Name: name, Category: category, Scope: "widely_observed", Tradition: "Solar", Regions: allIndia, Rule: "Sidereal solar ingress between this sunrise and the next"})
	}
	return obs
}

func (f festival) ruleText() string {
	when := map[decisive]string{atSunrise: "sunrise", atMidday: "midday", atPradosh: "pradosh", atNishita: "nishita", atAparahna: "aparahna", atPurvahna: "purvahna"}[f.when]
	switch f.kind {
	case solarDayOne:
		text := fmt.Sprintf("First day of the solar month of %s (%s rule)", rashis[f.month], solarRuleNames[f.solar])
		switch f.offset {
		case -1:
			return "Day before the " + text[:1] + text[1:]
		case 1:
			return "Day after the " + text
		}
		return text
	case solarNakshatra:
		return fmt.Sprintf("%s nakshatra at %s during the solar month of %s (%s rule)", nakshatras[f.nakshatra-1], when, rashis[f.month], solarRuleNames[f.solar])
	case solarTithi:
		return fmt.Sprintf("%s prevailing at %s during the solar month of %s (%s rule)", limbName("tithi", f.tithi), when, rashis[f.month], solarRuleNames[f.solar])
	case fridayBeforePurnima:
		return fmt.Sprintf("Friday on or before %s Purnima (Shukla paksha)", months[f.month])
	case holikaDahan:
		return "Phalguna Purnima prevailing at pradosh, avoiding Bhadra (Vishti karana)"
	case dayAfterHolika:
		return "Day after Holika Dahan (Phalguna Krishna Pratipada)"
	}
	return fmt.Sprintf("%s %s prevailing at %s (Amanta month)", months[f.month], limbName("tithi", f.tithi), when)
}

func contains(values []int, v int) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}
