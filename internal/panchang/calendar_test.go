package panchang

import (
	"context"
	"testing"
)

func delhi(t *testing.T, date string) Result {
	t.Helper()
	v, err := Calculate(context.Background(), native(t), Request{Date: date, Timezone: "Asia/Kolkata", Latitude: 28.6139, Longitude: 77.209, Profile: Profile})
	if err != nil {
		t.Fatal(date, err)
	}
	return v
}

func names(v Result) map[string]bool {
	out := map[string]bool{}
	for _, o := range v.Calendar.Observances {
		out[o.Name] = true
	}
	return out
}

// Rule-output regressions for Delhi 2026. The expected dates match widely
// published 2026 calendars; they pin the rules, they do not certify them.
func TestDelhiObservances2026(t *testing.T) {
	for date, want := range map[string]string{
		"2026-01-14": "Makar Sankranti",
		"2026-02-15": "Maha Shivaratri",
		"2026-03-19": "Chaitra Navaratri begins", // kshaya Pratipada, kept on the day it begins
		"2026-08-26": "Onam · Thiruvonam",
		"2026-10-16": "Durga Puja · Maha Shashthi",
		"2026-05-25": "Ganga Dussehra", // kept in Adhika Jyeshtha
		"2026-08-28": "Raksha Bandhan",
		"2026-09-04": "Krishna Janmashtami",
		"2026-09-14": "Ganesh Chaturthi",
		"2026-10-06": "Indira Ekadashi",
		"2026-10-29": "Karva Chauth",
		"2026-11-08": "Diwali · Lakshmi Puja",
	} {
		if v := delhi(t, date); !names(v)[want] {
			t.Errorf("%s: want %q, got %+v", date, want, v.Calendar.Observances)
		}
	}
	// Janmashtami must not also be reported on the preceding Saptami.
	if names(delhi(t, "2026-09-03"))["Krishna Janmashtami"] {
		t.Error("Janmashtami reported on Saptami")
	}
}

// Bhadra covers the whole Purnima pradosh of 2 March 2026 and Purnima ends
// before the next pradosh, so the classical rule keeps Holika Dahan on 2 March
// (in Bhadra puchha); published calendars show 3 March, an eclipse day.
// Pinned so a future change to this convention is deliberate.
func TestHolikaDahanWithoutBhadraRule(t *testing.T) {
	if !names(delhi(t, "2026-03-02"))["Holika Dahan"] {
		t.Fatal("pradosh-vyapini Purnima rule changed")
	}
}

func TestLunarMonthAndSamvat(t *testing.T) {
	v := delhi(t, "2026-09-26")
	c := v.Calendar
	if c.LunarMonth.Amanta != "Bhadrapada" || c.LunarMonth.Purnimanta != "Bhadrapada" || c.LunarMonth.Adhika {
		t.Fatalf("month %+v", c.LunarMonth)
	}
	if c.Samvat.Vikram != 2083 || c.Samvat.Shaka != 1948 {
		t.Fatalf("samvat %+v", c.Samvat)
	}
	if c.SunRashi != "Kanya" || len(c.MoonRashi) == 0 {
		t.Fatalf("rashi %s %+v", c.SunRashi, c.MoonRashi)
	}
	if !(c.LunarMonth.StartsAt.Before(v.Sunrise) && v.Sunrise.Before(c.LunarMonth.EndsAt)) {
		t.Fatal("month does not bracket sunrise")
	}
	// Krishna paksha belongs to the next Purnimanta month.
	if k := delhi(t, "2026-10-06").Calendar.LunarMonth; k.Amanta != "Bhadrapada" || k.Purnimanta != "Ashvina" {
		t.Fatalf("krishna month %+v", k)
	}
	// 2026 contains Adhika Jyeshtha (17 May - 15 June).
	if a := delhi(t, "2026-05-20").Calendar; !a.LunarMonth.Adhika || a.LunarMonth.Amanta != "Jyeshtha" {
		t.Fatalf("adhika %+v", a.LunarMonth)
	}
	// Magha falls in the Samvat year that began the previous spring.
	if j := delhi(t, "2026-02-01").Calendar.Samvat; j.Vikram != 2082 || j.Shaka != 1947 {
		t.Fatalf("january samvat %+v", j)
	}
}

func TestMuhurta(t *testing.T) {
	v := delhi(t, "2026-09-26")
	m := v.Calendar.Muhurta
	if m.Abhijit == nil {
		t.Fatal("abhijit missing on Saturday")
	}
	day := v.Sunset.Sub(v.Sunrise) / 15
	if d := m.Abhijit.End.Sub(m.Abhijit.Start) - day; d > 1e9 || d < -1e9 {
		t.Fatalf("abhijit length %v vs muhurta %v", m.Abhijit.End.Sub(m.Abhijit.Start), day)
	}
	if !(m.Brahma.End.Before(v.Sunrise) && m.Brahma.Start.Before(m.Brahma.End)) {
		t.Fatalf("brahma %+v", m.Brahma)
	}
	if delhi(t, "2026-09-30").Calendar.Muhurta.Abhijit != nil {
		t.Fatal("abhijit reported on Wednesday")
	}
}

func TestSankrantiInstant(t *testing.T) {
	v := delhi(t, "2026-01-14")
	sk := v.Calendar.Sankranti
	if sk == nil || sk.Rashi != "Makara" || sk.At.Before(v.Sunrise) || !sk.At.Before(v.NextSunrise) {
		t.Fatalf("sankranti %+v", sk)
	}
}

func TestRegionalSolarCalendars2026(t *testing.T) {
	for date, want := range map[string]string{
		"2026-01-13": "Lohri",
		"2026-01-14": "Thai Pongal",
		"2026-01-15": "Magh Bihu",
		"2026-04-14": "Puthandu · Tamil New Year",
		"2026-04-15": "Poila Baishakh · Bengali New Year",
		"2026-09-17": "Vishwakarma Puja",
		"2026-08-28": "Varalakshmi Vratam",
		"2026-11-08": "Kali Puja",
	} {
		if v := delhi(t, date); !names(v)[want] {
			t.Errorf("%s: want %q, got %+v", date, want, v.Calendar.Observances)
		}
	}
	// Regional festivals carry state codes; pan-Indian ones carry "IN".
	for _, o := range delhi(t, "2026-11-15").Calendar.Observances {
		if o.ID == "chhath_puja" && (len(o.RegionCodes) == 0 || o.RegionCodes[0] != "IN-BR") {
			t.Fatalf("chhath codes %v", o.RegionCodes)
		}
	}
}
