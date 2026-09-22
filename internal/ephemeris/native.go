// SPDX-License-Identifier: AGPL-3.0-or-later
package ephemeris

/*
#cgo CFLAGS: -I${SRCDIR}/../../.deps/swisseph
#cgo LDFLAGS: ${SRCDIR}/../../.deps/swisseph/libswe.a -lm -ldl
#include <stdlib.h>
#include "swephexp.h"
*/
import "C"

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"
)

// A process-wide gate protects Swiss state even when multiple Providers exist.
// Pin the goroutine for the whole transaction: upstream may use thread-local state.
var nativeGate = make(chan struct{}, 1)

type Native struct{ path, version string }
type nativeSession struct{}

func New(path string) (*Native, error) {
	if os.Getenv("SE_EPHE_PATH") != "" {
		return nil, fmt.Errorf("SE_EPHE_PATH overrides Swiss configuration; unset it and use SWISS_EPHEMERIS_PATH")
	}
	if path == "" {
		return nil, fmt.Errorf("ephemeris path is required")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if len(path) > 242 {
		return nil, fmt.Errorf("ephemeris path exceeds Swiss native limit")
	}
	for _, name := range []string{"sepl_18.se1", "semo_18.se1", "seas_18.se1"} {
		info, err := os.Stat(filepath.Join(path, name))
		if err != nil || !info.Mode().IsRegular() || info.Size() < 1024 {
			return nil, fmt.Errorf("required ephemeris file unavailable: %s", name)
		}
	}
	n := &Native{path: path}
	err = n.WithSession(context.Background(), func(s Session) error {
		var version [256]C.char
		C.swe_version(&version[0])
		n.version = C.GoString(&version[0])
		jd, err := s.JulianDay(time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC))
		if err != nil {
			return err
		}
		for _, body := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 15} {
			if _, err := s.Position(jd, body, false); err != nil {
				return err
			}
		}
		return nil
	})
	return n, err
}
func (n *Native) Version() string { return n.version }
func (n *Native) WithSession(ctx context.Context, fn func(Session) error) error {
	return n.WithSessionTiming(ctx, fn, nil)
}

// WithSessionTiming reports only native admission wait, excluding path setup.
func (n *Native) WithSessionTiming(ctx context.Context, fn func(Session) error, waited func(time.Duration)) error {
	started := time.Now()
	select {
	case nativeGate <- struct{}{}:
	case <-ctx.Done():
		if waited != nil {
			waited(time.Since(started))
		}
		return ctx.Err()
	}
	defer func() { <-nativeGate }()
	if waited != nil {
		waited(time.Since(started))
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	path := C.CString(n.path)
	defer C.free(unsafe.Pointer(path))
	C.swe_set_ephe_path(path)
	C.swe_set_sid_mode(C.SE_SIDM_LAHIRI, 0, 0)
	defer C.swe_close()
	return fn(nativeSession{})
}
func (nativeSession) JulianDay(t time.Time) (float64, error) {
	t = t.UTC()
	var jd [2]C.double
	var msg [256]C.char
	sec := float64(t.Second()) + float64(t.Nanosecond())/1e9
	rc := C.swe_utc_to_jd(C.int32(t.Year()), C.int32(t.Month()), C.int32(t.Day()), C.int32(t.Hour()), C.int32(t.Minute()), C.double(sec), C.SE_GREG_CAL, &jd[0], &msg[0])
	if rc < 0 {
		return 0, fmt.Errorf("%w: UTC conversion", ErrUnavailable)
	}
	return float64(jd[1]), nil
}
func (nativeSession) Time(jd float64) time.Time {
	var y, m, d, h, mi C.int32
	var sec C.double
	C.swe_jdut1_to_utc(C.double(jd), C.SE_GREG_CAL, &y, &m, &d, &h, &mi, &sec)
	return time.Date(int(y), time.Month(m), int(d), int(h), int(mi), int(sec), int((float64(sec)-float64(int(sec)))*1e9), time.UTC).Round(time.Second)
}
func (nativeSession) Position(jd float64, body int, sidereal bool) (Position, error) {
	flags := C.int32(C.SEFLG_SWIEPH | C.SEFLG_SPEED)
	if sidereal {
		flags |= C.SEFLG_SIDEREAL
	}
	var pos [6]C.double
	var msg [256]C.char
	rc := C.swe_calc_ut(C.double(jd), C.int32(body), flags, &pos[0], &msg[0])
	if rc < 0 || rc&C.SEFLG_SWIEPH == 0 {
		return Position{}, fmt.Errorf("%w: Swiss data required for body %d", ErrUnavailable, body)
	}
	return Position{Normalize(float64(pos[0])), float64(pos[3])}, nil
}
func (nativeSession) Houses(jd, lat, lon float64, system byte) ([13]float64, [10]float64, error) {
	var c [13]C.double
	var a [10]C.double
	var cusps [13]float64
	var angles [10]float64
	rc := C.swe_houses_ex(C.double(jd), 0, C.double(lat), C.double(lon), C.int(system), &c[0], &a[0])
	if rc < 0 {
		return cusps, angles, fmt.Errorf("houses unavailable")
	}
	for i := range cusps {
		cusps[i] = float64(c[i])
	}
	for i := range angles {
		angles[i] = float64(a[i])
	}
	return cusps, angles, nil
}
func (s nativeSession) RiseSet(jd float64, body int, lat, lon float64, rise bool) (float64, error) {
	// Assert data availability first; rise_trans itself returns a status, not ephemeris flags.
	if _, err := s.Position(jd, body, false); err != nil {
		return 0, err
	}
	var geo = [3]C.double{C.double(lon), C.double(lat), 0}
	var result C.double
	var msg [256]C.char
	mode := C.int32(C.SE_CALC_SET)
	if rise {
		mode = C.SE_CALC_RISE
	}
	// Upper limb, refraction, sea-level observer; fixed standard atmosphere.
	rc := C.swe_rise_trans(C.double(jd), C.int32(body), nil, C.SEFLG_SWIEPH, mode, &geo[0], 1013.25, 15, &result, &msg[0])
	if rc == -2 {
		return 0, ErrNoEvent
	}
	if rc < 0 {
		return 0, ErrUnavailable
	}
	return float64(result), nil
}
