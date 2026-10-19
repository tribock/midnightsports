package server

import (
	"testing"
	"time"
	_ "time/tzdata"
)

func TestEventTimes(t *testing.T) {
	tz, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, date, start, end string
		wantHours              float64
		wantEndDay             int
	}{
		{"gleicher Tag", "2026-11-07", "19:00", "23:00", 4, 7},
		{"über Mitternacht", "2026-11-07", "21:00", "01:00", 4, 8},
		{"bis Mitternacht", "2026-11-07", "21:00", "00:00", 3, 8},
		// Am 25.10.2026 endet die Sommerzeit, die Nacht hat eine Stunde mehr
		{"Zeitumstellung", "2026-10-24", "22:00", "04:00", 7, 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := eventTimes(tz, tt.date, tt.start, tt.end)
			if err != nil {
				t.Fatal(err)
			}
			if got := end.Sub(start).Hours(); got != tt.wantHours {
				t.Errorf("Dauer = %v, want %v", got, tt.wantHours)
			}
			if got := end.Day(); got != tt.wantEndDay {
				t.Errorf("Endtag = %d, want %d", got, tt.wantEndDay)
			}
		})
	}
}

func TestEventTimesInvalid(t *testing.T) {
	tz := time.UTC
	for _, in := range [][3]string{
		{"", "21:00", "23:00"},
		{"2026-13-01", "21:00", "23:00"},
		{"2026-11-07", "25:00", "23:00"},
		{"2026-11-07", "21:00", ""},
	} {
		if _, _, err := eventTimes(tz, in[0], in[1], in[2]); err == nil {
			t.Errorf("%v: Fehler erwartet", in)
		}
	}
}
