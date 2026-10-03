package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"airborne/internal/i18n"
	"airborne/internal/plan"
)

// The token limit is stored in <root>/settings.json, overrides AI_MAX_TOKENS
// while set, and 0 falls back to the .env default.
func TestMaxTokensSetting(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	src, _ := filepath.Abs(filepath.Join("..", "..", "..", "prompts"))
	for _, f := range []string{systemFile, mergeFile, quickFile, prefabFile, challengeFile, repairFile, schemaFile} {
		data, err := os.ReadFile(filepath.Join(src, f))
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(root, "prompts", f), data, 0o644)
	}
	t.Setenv("AI_MAX_TOKENS", "12345")

	p, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.aiConfig().MaxTokens; got != 12345 {
		t.Fatalf("default should come from .env: %d", got)
	}
	if err := p.SetMaxTokens(500); err == nil {
		t.Fatal("limits below the minimum must be rejected")
	}
	if err := p.SetMaxTokens(60000); err != nil {
		t.Fatal(err)
	}
	st := p.State()
	if st.Config.MaxTokens != 60000 || st.Config.MaxTokensEnv != 12345 {
		t.Fatalf("state: max=%d env=%d", st.Config.MaxTokens, st.Config.MaxTokensEnv)
	}

	// the setting survives a restart
	p2, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := p2.aiConfig().MaxTokens; got != 60000 {
		t.Fatalf("setting not persisted: %d", got)
	}
	if err := p2.SetMaxTokens(0); err != nil {
		t.Fatal(err)
	}
	if got := p2.aiConfig().MaxTokens; got != 12345 {
		t.Fatalf("0 should fall back to .env: %d", got)
	}
}

// The language is stored in settings.json, restored on startup and switches
// backend texts (roles, errors, validation issues).
func TestLanguageSetting(t *testing.T) {
	t.Cleanup(func() { i18n.SetLang(i18n.Default) })
	p := testPipeline(t)
	if st := p.State(); st.Config.Language != "de" || st.Roles[2].Title != "3 · Gegner" {
		t.Fatalf("default should be German: %q / %q", st.Config.Language, st.Roles[2].Title)
	}
	if err := p.SetLanguage("fr"); err == nil {
		t.Fatal("unknown languages must be rejected")
	}
	if err := p.SetLanguage("en"); err != nil {
		t.Fatal(err)
	}
	st := p.State()
	if st.Config.Language != "en" || st.Roles[2].Title != "3 · Enemy" {
		t.Fatalf("state not English: %q / %q", st.Config.Language, st.Roles[2].Title)
	}
	if _, err := p.Generate(); err == nil || !strings.HasPrefix(err.Error(), "No mission plan yet") {
		t.Fatalf("error not English: %v", err)
	}
	if issues := (&plan.MissionPlan{Game: "dcs"}).Validate(); len(issues) == 0 || issues[0] != "title (de/en) missing" {
		t.Fatalf("validation not English: %v", issues)
	}

	// survives a restart
	i18n.SetLang(i18n.Default)
	p2, err := New(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	if got := p2.State().Config.Language; got != "en" {
		t.Fatalf("language not persisted: %q", got)
	}
}
