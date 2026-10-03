package pipeline

import (
	"path/filepath"
	"strings"
	"testing"
)

func realPlaces(t *testing.T) []Place {
	t.Helper()
	root, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	places := loadPlaces(root)
	if len(places) < 100 {
		t.Fatalf("expected the reference place list, got %d entries", len(places))
	}
	return places
}

// Airfields and major cities are always listed; villages only when named,
// matched as whole words and via modern/German spellings.
func TestPlacesContext(t *testing.T) {
	places := realPlaces(t)

	base := placesContext(places, "")
	for _, want := range []string{"K-14 Kimpo (105928, 269363)", "Seoul (", "Pyongyang (", "DPRK military camp 01 (", "Staudaemme: "} {
		if !strings.Contains(base, want) {
			t.Errorf("base context lacks %q", want)
		}
	}
	if strings.Contains(base, "Anak (") || strings.Contains(base, "In der Beschreibung genannte Orte") {
		t.Error("villages must only be listed when the text names them")
	}
	if n := len(base); n > 20000 {
		t.Errorf("base context too large for the prompt: %d bytes", n)
	}

	named := placesContext(places, "Angriff auf die Bruecke bei Anak, Rueckflug nach Pjöngjang.")
	if !strings.Contains(named, "In der Beschreibung genannte Orte: Anak (") {
		t.Errorf("named village not listed:\n%s", named)
	}
	if strings.Count(named, "Pyongyang (") != strings.Count(base, "Pyongyang (") {
		t.Error("alias must not duplicate an already listed city")
	}
	if strings.Contains(placesContext(places, "Anakonda"), "Anak (") {
		t.Error("partial word must not match")
	}
	if placesContext(nil, "Seoul") != "" {
		t.Error("no places -> no section")
	}
}

// The IL-2 prompt carries the place list, the DCS prompt does not.
func TestGameContextPlaces(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	p, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.challengePrompt("il2", "f86a5", "Anak"); !strings.Contains(got, "K-14 Kimpo (") || !strings.Contains(got, "Anak (") {
		t.Error("il2 prompt lacks the place list")
	}
	if got := p.challengePrompt("dcs", "MiG-21Bis", ""); strings.Contains(got, "K-14 Kimpo (") {
		t.Error("dcs prompt must not carry IL-2 places")
	}
}
