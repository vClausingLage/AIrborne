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
	for _, f := range []string{systemFile, mergeFile, quickFile, prefabFile, schemaFile} {
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
