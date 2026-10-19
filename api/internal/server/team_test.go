package server

import "testing"

func TestStaffEntry(t *testing.T) {
	salary := func(v float64) *float64 { return &v }
	signed := staffSnapshot{functionID: 1, salary: salary(20), signature: []byte("png")}

	tests := []struct {
		name       string
		prev       staffSnapshot
		hadPrev    bool
		functionID int
		salary     *float64
		want       float64
		keepsSig   bool
	}{
		{"neu eingeteilt", staffSnapshot{}, false, 1, salary(40), 40, false},
		{"gleiche Funktion behält Betrag und Unterschrift", signed, true, 1, salary(40), 20, true},
		{"andere Funktion erhält neuen Betrag", signed, true, 2, salary(80), 80, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := staffEntry(tt.prev, tt.hadPrev, tt.functionID, tt.salary)
			if got.salary == nil || *got.salary != tt.want {
				t.Errorf("Betrag = %v, want %v", got.salary, tt.want)
			}
			if (got.signature != nil) != tt.keepsSig {
				t.Errorf("Unterschrift erhalten = %v, want %v", got.signature != nil, tt.keepsSig)
			}
		})
	}

	t.Run("ohne Lohn über Zeiterfassung", func(t *testing.T) {
		if got := staffEntry(signed, true, 1, nil); got.salary != nil || got.signature != nil {
			t.Errorf("got %+v, want kein Betrag und keine Unterschrift", got)
		}
	})
}
