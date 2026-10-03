package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"airborne/internal/ai"
	"airborne/internal/i18n"
	"airborne/internal/logging"
)

// Settings are the user-editable options that live next to .env in
// <root>/settings.json. Unlike .env they are written by the app itself.
type Settings struct {
	// MaxTokens caps the output tokens of one plan request (0 = use the
	// AI_MAX_TOKENS default from .env). A plan is a single LLM call, so this
	// is also the limit per generated mission.
	MaxTokens int `json:"maxTokens"`
	// Language is the UI and message language ("de" or "en"; empty = de).
	Language string `json:"language,omitempty"`
}

const (
	settingsFile = "settings.json"
	// minMaxTokens guards against limits so small that no plan JSON can fit.
	minMaxTokens = 1000
	// maxMaxTokens is the sanity ceiling; no current model emits more per request.
	maxMaxTokens = 1_000_000
)

func (p *Pipeline) settingsPath() string {
	return filepath.Join(p.Root, settingsFile)
}

func (p *Pipeline) loadSettings() {
	data, err := os.ReadFile(p.settingsPath())
	if err != nil {
		return
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		logging.Errorf("settings.json ungueltig, Standardwerte verwendet: %v", err)
		return
	}
	p.settings = s
	i18n.SetLang(s.Language)
}

func (p *Pipeline) saveSettings() error {
	data, err := json.MarshalIndent(p.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.settingsPath(), data, 0o644)
}

// SetMaxTokens stores the per-plan output token limit; 0 falls back to .env.
func (p *Pipeline) SetMaxTokens(n int) error {
	if n != 0 && (n < minMaxTokens || n > maxMaxTokens) {
		return i18n.Errorf("Token-Limit muss zwischen %d und %d liegen (0 = Standard aus .env)", "Token limit must be between %d and %d (0 = default from .env)", minMaxTokens, maxMaxTokens)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.settings.MaxTokens = n
	if err := p.saveSettings(); err != nil {
		return i18n.Errorf("Einstellungen konnten nicht gespeichert werden: %w", "Settings could not be saved: %w", err)
	}
	logging.Infof("Token-Limit pro Plan: %d (0 = .env)", n)
	return nil
}

// SetLanguage switches the UI and message language and stores it.
func (p *Pipeline) SetLanguage(lang string) error {
	if !i18n.Valid(lang) {
		return i18n.Errorf("Unbekannte Sprache %q (de|en)", "Unknown language %q (de|en)", lang)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.settings.Language = lang
	i18n.SetLang(lang)
	if err := p.saveSettings(); err != nil {
		return i18n.Errorf("Einstellungen konnten nicht gespeichert werden: %w", "Settings could not be saved: %w", err)
	}
	logging.Infof("Sprache: %s", lang)
	return nil
}

// aiConfig is the .env LLM config with the user settings applied.
func (p *Pipeline) aiConfig() ai.Config {
	cfg := ai.LoadConfig()
	p.mu.Lock()
	if p.settings.MaxTokens > 0 {
		cfg.MaxTokens = p.settings.MaxTokens
	}
	p.mu.Unlock()
	return cfg
}

// chat runs one LLM request with the effective config and remembers the
// token usage for the UI. Tests replace the transport via chatFn.
func (p *Pipeline) chat(ctx context.Context, messages []ai.Message) (string, error) {
	send := p.aiConfig().ChatUsage
	if p.chatFn != nil {
		send = p.chatFn
	}
	out, usage, err := send(ctx, messages)
	if usage.TotalTokens > 0 {
		p.mu.Lock()
		u := usage
		p.lastUsage = &u
		p.mu.Unlock()
	}
	return out, err
}
