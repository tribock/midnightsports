package server

import (
	"testing"
	"time"
)

func TestStatsRange(t *testing.T) {
	tz, _ := time.LoadLocation("Europe/Zurich")

	tests := []struct {
		name     string
		from, to string
		ok       bool
	}{
		{"Zeitraum", "2026-03-01", "2026-11-14", true},
		{"ein Tag", "2026-11-14", "2026-11-14", true},
		{"Ende vor Beginn", "2026-11-14", "2026-03-01", false},
		{"fehlt", "", "2026-11-14", false},
		{"ungültiges Datum", "14.11.2026", "2026-11-14", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := statsRange(tz, tt.from, tt.to)
			if (err == nil) != tt.ok {
				t.Fatalf("err = %v, want ok = %v", err, tt.ok)
			}
			// Das Ende ist exklusiv: Mitternacht nach dem letzten Tag
			if tt.ok && !end.Equal(time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, tz)) || tt.ok && !start.Before(end) {
				t.Errorf("start %v, end %v", start, end)
			}
		})
	}
}
