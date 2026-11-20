package server

import "testing"

func TestFunctionChanges(t *testing.T) {
	jc, sc := "JC", "SC"
	got := functionChanges([]FunctionChange{{To: &jc}, {To: &sc}, {To: nil}})
	want := []struct{ from, to *string }{{nil, &jc}, {&jc, &sc}, {&sc, nil}}
	for i, w := range want {
		if got[i].From != w.from || got[i].To != w.to {
			t.Errorf("Eintrag %d: %v → %v, want %v → %v", i, got[i].From, got[i].To, w.from, w.to)
		}
	}
}
