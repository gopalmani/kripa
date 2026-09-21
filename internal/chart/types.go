// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2024-2026 AstroMatch contributors
// Adapted for KRIPA; see NOTICE.md.
package chart

type PlanetPosition struct {
	Planet     string  `json:"planet"`
	Sign       string  `json:"sign"`
	Degree     float64 `json:"degree"`
	House      *int    `json:"house"`
	Retrograde bool    `json:"retrograde"`
}
type HousePosition struct {
	House  int     `json:"house"`
	Sign   string  `json:"sign"`
	Degree float64 `json:"degree"`
}
type Aspect struct {
	PlanetA string  `json:"planet_a"`
	PlanetB string  `json:"planet_b"`
	Aspect  string  `json:"aspect"`
	Orb     float64 `json:"orb"`
}
type ChartMetadata struct {
	CalculationVersion string `json:"calculation_version"`
	EphemerisVersion   string `json:"ephemeris_version"`
	HouseSystem        string `json:"house_system"`
	DataQuality        string `json:"data_quality"`
	InputHash          string `json:"-"`
}
type Chart struct {
	Metadata        ChartMetadata    `json:"metadata"`
	Planets         []PlanetPosition `json:"planets"`
	Houses          []HousePosition  `json:"houses"`
	Aspects         []Aspect         `json:"aspects"`
	SunSign         string           `json:"sun_sign"`
	AscendantSign   *string          `json:"ascendant"`
	AscendantDegree *float64         `json:"ascendant_degree"`
	MCSign          *string          `json:"mc"`
	MCDegree        *float64         `json:"mc_degree"`
}
