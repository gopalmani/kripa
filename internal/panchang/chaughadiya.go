package panchang

import "time"

// ChaughadiyaPeriod is one eighth of the day (sunrise to sunset) or night
// (sunset to next sunrise). Nature is the traditional classification only.
type ChaughadiyaPeriod struct {
	Name   string    `json:"name"`
	Nature string    `json:"nature"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
}
type Chaughadiya struct {
	Day   []ChaughadiyaPeriod `json:"day"`
	Night []ChaughadiyaPeriod `json:"night"`
}

// Day periods follow Udveg, Char, Labh, Amrit, Kaal, Shubh, Rog from the
// weekday lord (Sunday Udveg, Monday Amrit, ...); night periods follow Shubh,
// Amrit, Char, Rog, Kaal, Labh, Udveg (Sunday night Shubh, Monday night Char, ...).
var (
	dayCycle   = []string{"Udveg", "Char", "Labh", "Amrit", "Kaal", "Shubh", "Rog"}
	nightCycle = []string{"Shubh", "Amrit", "Char", "Rog", "Kaal", "Labh", "Udveg"}
	natures    = map[string]string{"Amrit": "good", "Shubh": "good", "Labh": "good", "Char": "neutral", "Udveg": "inauspicious", "Kaal": "inauspicious", "Rog": "inauspicious"}
)

func chaughadiya(rise, set, nextRise time.Time, weekday int) Chaughadiya {
	return Chaughadiya{
		Day:   eighths(rise, set, dayCycle, weekday*3%7),
		Night: eighths(set, nextRise, nightCycle, weekday*2%7),
	}
}

func eighths(from, to time.Time, cycle []string, first int) []ChaughadiyaPeriod {
	d := to.Sub(from) / 8
	out := make([]ChaughadiyaPeriod, 8)
	for i := range out {
		name := cycle[(first+i)%7]
		end := from.Add(time.Duration(i+1) * d).Round(time.Second)
		if i == 7 {
			end = to
		}
		out[i] = ChaughadiyaPeriod{name, natures[name], from.Add(time.Duration(i) * d).Round(time.Second), end}
	}
	return out
}
