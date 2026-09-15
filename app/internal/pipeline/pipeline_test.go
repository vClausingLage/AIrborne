package pipeline

import (
	"path/filepath"
	"strings"
	"testing"
)

// New must load every prompt template from <root>/prompts, and the system
// prompt must name the target game.
func TestSystemPrompt(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	p, err := New(root)
	if err != nil {
		t.Fatalf("New(%s): %v", root, err)
	}
	for _, f := range []string{systemFile, mergeFile, quickFile, prefabFile, challengeFile, schemaFile} {
		if strings.TrimSpace(p.tmpl[f]) == "" {
			t.Errorf("template %s is empty", f)
		}
	}
	for game, want := range map[string]string{"il2": "IL-2 Sturmovik: Korea", "dcs": "DCS World"} {
		got := p.systemPrompt(game)
		if !strings.Contains(got, "**"+want+"**") {
			t.Errorf("systemPrompt(%s) does not name the game %q:\n%s", game, want, got)
		}
		if strings.Contains(got, "{{") {
			t.Errorf("systemPrompt(%s) has unfilled placeholder", game)
		}
	}
}

// The challenge prompt names the aircraft, the map hint and carries the
// no-enemy-information rule plus the schema.
func TestChallengePrompt(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	p, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	got := p.challengePrompt("dcs", "MiG-21Bis", "")
	for _, want := range []string{"MiG-21bis (MiG-21Bis)", "frei waehlbar", "keine Gegnerinformationen", "\"playerGroups\""} {
		if !strings.Contains(got, want) {
			t.Errorf("challenge prompt lacks %q", want)
		}
	}
	if strings.Contains(got, "{{") {
		t.Error("unfilled placeholder in challenge prompt")
	}
	if got := p.challengePrompt("il2", "f86a5", ""); !strings.Contains(got, "**Karte:** Korea") {
		t.Error("il2 challenge prompt must pin the map to Korea")
	}
	if got := p.challengePrompt("dcs", "Eigenbau-Typ", "Syria"); !strings.Contains(got, "**Spielerflugzeug:** Eigenbau-Typ") || !strings.Contains(got, "**Karte:** Syria") {
		t.Error("free-text aircraft / map hint not passed through")
	}
}
