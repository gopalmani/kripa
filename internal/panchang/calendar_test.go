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
		"2026-03-04": "Holi",
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

// Known convention gap: Bhadra avoidance is not applied, so Holika Dahan
// follows pradosh-vyapini Purnima (2 March) where many calendars shift to
// 3 March. Pinned so any future Bhadra rule is a deliberate change.
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
