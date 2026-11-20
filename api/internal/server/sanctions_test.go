package server

import (
	"testing"
	"time"
)

func TestEffectiveSanction(t *testing.T) {
	n := func(v int) *int { return &v }
	since := time.Date(2026, 10, 1, 20, 0, 0, 0, time.UTC)
	day := func(d int) time.Time { return since.Add(days(d)) }
	at := func(d int) *time.Time { t := day(d); return &t }
	tests := []struct {
		name      string
		level     string
		rules     sanctionRules
		now       time.Time
		want      string
		wantUntil *time.Time
	}{
		{"Sperre ohne Frist", LevelBan, sanctionRules{BanAction: BanActionUnban}, day(400), LevelBan, nil},
		{"Sperre läuft noch", LevelBan, sanctionRules{BanDays: n(30), BanAction: BanActionUnban}, day(29), LevelBan, at(30)},
		{"Sperre abgelaufen, entsperrt", LevelBan, sanctionRules{BanDays: n(30), BanAction: BanActionUnban}, day(30), "", nil},
		{"Sperre abgelaufen, wird Verwarnung ohne Frist", LevelBan,
			sanctionRules{BanDays: n(30), BanAction: BanActionWarn}, day(100), LevelWarning, nil},
		{"Verwarnung nach Sperre zählt ab Ende der Sperre", LevelBan,
			sanctionRules{BanDays: n(30), BanAction: BanActionWarn, WarningDays: n(60)}, day(89), LevelWarning, at(90)},
		{"Verwarnung nach Sperre abgelaufen", LevelBan,
			sanctionRules{BanDays: n(30), BanAction: BanActionWarn, WarningDays: n(60)}, day(90), "", nil},
		{"Verwarnung ohne Frist", LevelWarning, sanctionRules{BanAction: BanActionUnban}, day(1000), LevelWarning, nil},
		{"Verwarnung läuft noch", LevelWarning, sanctionRules{WarningDays: n(14)}, day(13), LevelWarning, at(14)},
		{"Verwarnung gelöscht", LevelWarning, sanctionRules{WarningDays: n(14)}, day(14), "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, until := effectiveSanction(tt.level, since, tt.rules, tt.now)
			if got != tt.want {
				t.Errorf("Stufe = %q, want %q", got, tt.want)
			}
			if (until == nil) != (tt.wantUntil == nil) || (until != nil && !until.Equal(*tt.wantUntil)) {
				t.Errorf("bis = %v, want %v", until, tt.wantUntil)
			}
		})
	}
}

func TestSanctionRulesValidate(t *testing.T) {
	n := func(v int) *int { return &v }
	tests := []struct {
		name  string
		rules sanctionRules
		ok    bool
	}{
		{"ohne Fristen", sanctionRules{BanAction: BanActionUnban}, true},
		{"mit Fristen", sanctionRules{BanDays: n(30), BanAction: BanActionWarn, WarningDays: n(90)}, true},
		{"unbekannte Aktion", sanctionRules{BanAction: "loeschen"}, false},
		{"Frist 0 Tage", sanctionRules{BanDays: n(0), BanAction: BanActionUnban}, false},
		{"Frist zu lang", sanctionRules{WarningDays: n(4000), BanAction: BanActionUnban}, false},
	}
	for _, tt := range tests {
		if err := tt.rules.validate(); (err == nil) != tt.ok {
			t.Errorf("%s: err = %v, gültig erwartet: %v", tt.name, err, tt.ok)
		}
	}
}

func TestBanned(t *testing.T) {
	if banned(nil) || banned([]Sanction{{Level: LevelWarning}}) {
		t.Error("Verwarnung gilt nicht als Sperre")
	}
	if !banned([]Sanction{{Level: LevelBan}, {Level: LevelWarning}}) {
		t.Error("Sperre nicht erkannt")
	}
}
