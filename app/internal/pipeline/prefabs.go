package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"airborne/internal/ai"
	"airborne/internal/logging"
	"airborne/internal/plan"
	"airborne/internal/prefab"
)

// gameContext is GameContext plus, for DCS, the list of saved prefabs the
// mission LLM may place via plan.prefabs.
func (p *Pipeline) gameContext(game string) string {
	ctx := GameContext(game)
	if game != "dcs" {
		return ctx
	}
	list, _ := p.Lib.List()
	if len(list) == 0 {
		return ctx
	}
	var b strings.Builder
	b.WriteString(ctx)
	b.WriteString("\n\n## Verfuegbare Prefabs (fertige Asset-Gruppen, DCS)\n")
	b.WriteString("Diese Prefabs koennen im Plan ueber `prefabs` platziert werden: `{\"prefab\":\"<id>\",\"side\":\"player|friendly|enemy\",\"name\":\"...\",\"position\":{\"lat\":..,\"lon\":..,\"heading\":..}}`. Die enthaltenen Einheiten NICHT zusaetzlich in enemyGroups/friendlyGroups/statics aufzaehlen. Nutze ein Prefab nur, wenn es zur Story passt (Epoche, Fraktion, Wasser/Land).\n")
	for _, pf := range list {
		kinds := map[string]int{}
		for _, e := range pf.Elements {
			kinds[e.Kind] += e.Units()
		}
		var parts []string
		for _, k := range []string{"ship", "vehicle", "plane", "helicopter", "static"} {
			if kinds[k] > 0 {
				parts = append(parts, fmt.Sprintf("%d %s", kinds[k], k))
			}
		}
		fmt.Fprintf(&b, "- id \"%s\": %s (%s; Country %s) - %s\n", pf.ID, pf.Name, strings.Join(parts, ", "), orDash(pf.Country), pf.Description)
	}
	return b.String()
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// GeneratePrefab asks the LLM for a prefab layout. The result is NOT saved;
// the caller reviews it and calls SavePrefab.
func (p *Pipeline) GeneratePrefab(ctx context.Context, input string) (prefab.Prefab, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return prefab.Prefab{}, fmt.Errorf("Bitte beschreiben, was das Prefab enthalten soll")
	}
	p.mu.Lock()
	prompt := fill(p.tmpl[prefabFile], map[string]string{
		"PREFAB_INPUT": input,
		"CATALOG":      prefab.CatalogText(),
	})
	p.mu.Unlock()

	logging.Infof("prefab gestartet")
	logging.Block("PROMPT prefab", prompt)
	cfg := ai.LoadConfig()
	out, err := cfg.Chat(ctx, []ai.Message{{Role: "user", Content: prompt}})
	if err != nil {
		logging.Errorf("prefab fehlgeschlagen: %v", err)
		return prefab.Prefab{}, err
	}
	logging.Block("RESULT prefab RAW", out)
	pf, err := decodePrefab(out)
	if err != nil {
		logging.Errorf("prefab JSON-Fehler: %v", err)
		return prefab.Prefab{}, err
	}
	for _, is := range pf.Normalize() {
		logging.Infof("  Prefab-Hinweis: %s", is)
	}
	logging.Infof("Prefab erfolgreich: %s (%d Elemente)", pf.Name, len(pf.Elements))
	return pf, nil
}

func decodePrefab(out string) (prefab.Prefab, error) {
	raw, err := ai.ExtractJSON(out)
	if err != nil {
		return prefab.Prefab{}, fmt.Errorf("Antwort enthaelt kein JSON: %w", err)
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return prefab.Prefab{}, fmt.Errorf("JSON ungueltig: %w", err)
	}
	data, err := json.Marshal(normalizeJSON(generic))
	if err != nil {
		return prefab.Prefab{}, err
	}
	var pf prefab.Prefab
	if err := json.Unmarshal(data, &pf); err != nil {
		return prefab.Prefab{}, fmt.Errorf("Prefab-JSON ungueltig: %w", err)
	}
	return pf, nil
}

// SavePrefab stores user-reviewed prefab JSON in the library.
func (p *Pipeline) SavePrefab(text string) (prefab.Prefab, error) {
	pf, err := decodePrefab(text)
	if err != nil {
		return prefab.Prefab{}, err
	}
	saved, err := p.Lib.Save(pf)
	if err != nil {
		return prefab.Prefab{}, err
	}
	logging.Infof("Prefab gespeichert: %s (%s)", saved.Name, saved.ID)
	return saved, nil
}

func (p *Pipeline) DeletePrefab(id string) error {
	return p.Lib.Delete(id)
}

// PlacePrefab appends a prefab placement to the current plan.
func (p *Pipeline) PlacePrefab(id, side string, lat, lon, heading float64, country string) error {
	pf, ok := p.Lib.Find(id)
	if !ok {
		return fmt.Errorf("Prefab %q nicht gefunden", id)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proj.Plan == nil {
		return fmt.Errorf("Kein Missionsplan vorhanden - erst Blitz oder Merge ausfuehren, dann Prefab einfuegen")
	}
	if p.proj.Plan.Game != "dcs" {
		return fmt.Errorf("Prefabs werden nur fuer DCS unterstuetzt")
	}
	pp := plan.PrefabPlacement{Prefab: pf.ID, Name: pf.Name, Side: side,
		Position: plan.Position{Lat: lat, Lon: lon, Head: heading}}
	if strings.TrimSpace(country) != "" {
		pp.Country = strings.TrimSpace(country)
	}
	p.proj.Plan.Prefabs = append(p.proj.Plan.Prefabs, pp)
	p.proj.Output = nil
	return nil
}
