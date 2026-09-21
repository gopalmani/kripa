// SPDX-License-Identifier: AGPL-3.0-or-later
package ephemeris

import (
	"context"
	"errors"
	"math"
	"time"
)

const SwissCommit = "56351c57e33916651a1aa32cfc6225e05c0ad865"
const DataVersion = "swisseph-2.10.3a-1800-2399"

var ErrUnavailable = errors.New("ephemeris unavailable")
var ErrNoEvent = errors.New("rise or set event unavailable")

type Position struct{ Longitude, Speed float64 }
type Session interface {
	JulianDay(time.Time) (float64, error)
	Time(float64) time.Time
	Position(jd float64, body int, sidereal bool) (Position, error)
	Houses(jd, latitude, longitude float64, system byte) ([13]float64, [10]float64, error)
	RiseSet(jd float64, body int, latitude, longitude float64, rise bool) (float64, error)
}
type Provider interface {
	WithSession(context.Context, func(Session) error) error
	Version() string
}

func Normalize(x float64) float64 {
	x = math.Mod(x, 360)
	if x < 0 {
		x += 360
	}
	return x
}
func ValidCoordinate(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && !math.IsInf(lat, 0) && !math.IsInf(lon, 0) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}
