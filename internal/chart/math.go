// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2024-2026 AstroMatch contributors
// Adapted from Astrel chart mathematics; see NOTICE.md.
package chart

import "math"

func planetPosition(name string, longitude float64, retrograde bool) PlanetPosition {
	return PlanetPosition{
		Planet: name, Sign: zodiac(longitude), Degree: signDegree(longitude), Retrograde: retrograde,
	}
}

var zodiacSigns = []string{
	"Aries", "Taurus", "Gemini", "Cancer", "Leo", "Virgo",
	"Libra", "Scorpio", "Sagittarius", "Capricorn", "Aquarius", "Pisces",
}

func zodiac(longitude float64) string {
	return zodiacSigns[int(normalizeDegree(longitude)/30)]
}

func signDegree(longitude float64) float64 {
	return normalizeDegree(longitude) - math.Floor(normalizeDegree(longitude)/30)*30
}

func normalizeDegree(degree float64) float64 {
	degree = math.Mod(degree, 360)
	if degree < 0 {
		degree += 360
	}
	return degree
}

func houseFor(longitude float64, cusps []float64) int {
	for house := 1; house <= 12; house++ {
		next := house + 1
		if next == 13 {
			next = 1
		}
		start := normalizeDegree(cusps[house])
		span := normalizeDegree(cusps[next] - start)
		if normalizeDegree(longitude-start) < span {
			return house
		}
	}
	return 12
}

func calculateAspects(longitudes map[string]float64) []Aspect {
	type aspectRule struct {
		name   string
		angle  float64
		maxOrb float64
	}
	rules := []aspectRule{
		{"conjunction", 0, 8},
		{"sextile", 60, 6},
		{"square", 90, 7},
		{"trine", 120, 8},
		{"opposition", 180, 8},
	}
	names := make([]string, 0, len(longitudes))
	for _, body := range bodies {
		names = append(names, body.name)
	}
	names = append(names, "South Node")
	out := make([]Aspect, 0)
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			separation := math.Abs(longitudes[names[i]] - longitudes[names[j]])
			if separation > 180 {
				separation = 360 - separation
			}
			for _, rule := range rules {
				orb := math.Abs(separation - rule.angle)
				if orb <= rule.maxOrb {
					out = append(out, Aspect{
						PlanetA: names[i], PlanetB: names[j], Aspect: rule.name, Orb: orb,
					})
					break
				}
			}
		}
	}
	return out
}
