package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"airborne/internal/ai"
	"airborne/internal/dcs"
	"airborne/internal/gen"
	"airborne/internal/il2"
	"airborne/internal/logging"
	"airborne/internal/plan"
	"airborne/internal/prefab"
)

// RoleDef describes one of the five input sections of the step-by-step mode.
// The user writes free text per section; the sections are merged into ONE
// structured prompt (prompts/06-merge.md) - there are no per-role LLM calls.
type RoleDef struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Placeholder string `json:"placeholder"`
	Hint        string `json:"hint"`
}

var Roles = []RoleDef{
	{Key: "story", Title: "1 · Story / Lage",
		Hint:        "Krieg, Jahr/Monat, Region, taktische Lage, warum dieser Einsatz, Wetter, Tageszeit",
		Placeholder: "z. B. September 1950, Vorabend der Landung bei Inchon. Nordkoreanische Il-10 sollen UN-Aufklärungstrupps im Hügelland nordwestlich der Stadt ausschalten…"},
	{Key: "player", Title: "2 · Player",
		Hint:        "Flugzeugtyp, Anzahl Spielerplätze, Fraktion, Callsign, Startart/Ort, Bewaffnung",
		Placeholder: "z. B. 2x Il-10, KPAF, Luftstart über der Küste westlich von Inchon, Bomben + Raketen…"},
	{Key: "enemy", Title: "3 · Gegner",
		Hint:        "Feindliche Einheiten Luft/Boden, Typen, Anzahl, Positionen, Verhalten (statisch/Patrouille/Kolonne)",
		Placeholder: "z. B. Lastwagen und Jeeps unter Tarnnetzen im Hügelland, eine Jeep-Patrouille Richtung Kimpo, evtl. ein Paar F-51D als Jagdschutz…"},
	{Key: "resistance", Title: "4 · Widerstand",
		Hint:        "Flak/Luftabwehr, Bodenwiderstand, Verbündete, statische Objekte – realistisch dosiert",
		Placeholder: "z. B. zwei Bofors-Geschütze und ein M16-Flakpanzer beim Hauptversteck, Tarnnetze über den Fahrzeugen…"},
	{Key: "briefing", Title: "5 · Briefing / Funk",
		Hint:        "Ton und Sprache des Briefings, Callsigns, Funksprüche (Intro, Kontakt, Erfolg, Zeitlimit)",
		Placeholder: "z. B. propagandistischer KPA-Ton, Callsign „Sowol“, Meldungen von „Basis“: Einleitung nach 10 s, Feindkontakt im Zielgebiet, Erfolgsmeldung, Zeitlimit 25 min…"},
}

const (
	mergeFile  = "06-merge.md"
	quickFile  = "07-quick.md"
	prefabFile = "08-prefab.md"
	schemaFile = "schema.md"
)

type Project struct {
	Game   string            `json:"game"`
	Inputs map[string]string `json:"inputs"`
	Plan   *plan.MissionPlan `json:"plan,omitempty"`
	Output *gen.Result       `json:"output,omitempty"`
}

func NewProject() *Project {
	return &Project{Game: "il2", Inputs: map[string]string{}}
}

type RoleState struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Hint        string `json:"hint"`
	Placeholder string `json:"placeholder"`
	Input       string `json:"input"`
}

type ConfigState struct {
	Base           string `json:"base"`
	Model          string `json:"model"`
	HasKey         bool   `json:"hasKey"`
	PromptsDir     string `json:"promptsDir"`
	LogPath        string `json:"logPath"`
	IL2MissionsDir string `json:"il2MissionsDir"`
	IL2Editor      string `json:"il2Editor"`
	DCSMissionsDir string `json:"dcsMissionsDir"`
}

type State struct {
	Roles       []RoleState       `json:"roles"`
	Config      ConfigState       `json:"config"`
	Game        string            `json:"game"`
	Plan        *plan.MissionPlan `json:"plan"`
	PlanJSON    string            `json:"planJson"`
	ProjectPath string            `json:"projectPath"`
	Root        string            `json:"root"`
	Issues      []string          `json:"issues"`
	Output      *gen.Result       `json:"output"`
	Prefabs     []prefab.Prefab   `json:"prefabs"`
}

type Pipeline struct {
	mu         sync.Mutex
	Root       string
	PromptsDir string
	tmpl       map[string]string
	proj       *Project
	projPath   string
	Lib        *prefab.Library
}

func New(root string) (*Pipeline, error) {
	p := &Pipeline{Root: root}
	p.PromptsDir = filepath.Join(root, "prompts")
	if err := p.loadTemplates(); err != nil {
		return nil, err
	}
	p.Lib = prefab.NewLibrary(filepath.Join(root, "prefabs"))
	p.proj = NewProject()
	p.projPath = filepath.Join(root, "projects", "current.json")
	if data, err := os.ReadFile(p.projPath); err == nil {
		proj := NewProject()
		if json.Unmarshal(data, proj) == nil {
			if proj.Inputs == nil {
				proj.Inputs = map[string]string{}
			}
			p.proj = proj
		}
	}
	return p, nil
}

func (p *Pipeline) loadTemplates() error {
	p.tmpl = map[string]string{}
	for _, f := range []string{mergeFile, quickFile, prefabFile, schemaFile} {
		data, err := os.ReadFile(filepath.Join(p.PromptsDir, f))
		if err != nil {
			return fmt.Errorf("Prompt-Template fehlt: %s (%w)", f, err)
		}
		p.tmpl[f] = string(data)
	}
	return nil
}

func FindRoot(starts ...string) string {
	for _, s := range starts {
		dir := s
		for i := 0; i < 6 && dir != ""; i++ {
			if fileExists(filepath.Join(dir, "prompts", mergeFile)) {
				return dir
			}
			dir = filepath.Dir(dir)
		}
	}
	return ""
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// Paths holds the game folders read from .env.
type Paths struct {
	IL2MissionsDir string
	IL2Editor      string
	IL2Root        string
	DCSSavedGames  string
	DCSMissionsDir string
}

func LoadPaths() Paths {
	clean := func(k string) string {
		v := strings.TrimSpace(os.Getenv(k))
		v = strings.Trim(v, "\"'")
		return strings.TrimRight(v, "\\/")
	}
	ps := Paths{
		IL2MissionsDir: clean("IL2_MISSIONS_DIR"),
		IL2Editor:      clean("IL2_EDITOR"),
		IL2Root:        clean("IL2_ROOT"),
		DCSSavedGames:  clean("DCS_SAVED_GAMES"),
		DCSMissionsDir: clean("DCS_MISSIONS_DIR"),
	}
	if ps.DCSMissionsDir == "" && ps.DCSSavedGames != "" {
		ps.DCSMissionsDir = filepath.Join(ps.DCSSavedGames, "Missions")
	}
	return ps
}

func userSection(text string) string {
	if strings.TrimSpace(text) == "" {
		return "(keine Vorgabe - frei nach Story entscheiden)"
	}
	return strings.TrimSpace(text)
}

func fill(tmpl string, ph map[string]string) string {
	out := tmpl
	for k, v := range ph {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

func (p *Pipeline) State() State {
	cfg := ai.LoadConfig()
	paths := LoadPaths()
	st := State{
		Roles:  []RoleState{},
		Issues: []string{},
		Config: ConfigState{
			Base: cfg.Base, Model: cfg.Model, HasKey: cfg.Key != "",
			PromptsDir: p.PromptsDir, LogPath: logging.Path(),
			IL2MissionsDir: paths.IL2MissionsDir, IL2Editor: paths.IL2Editor, DCSMissionsDir: paths.DCSMissionsDir,
		},
		Game:        p.proj.Game,
		Plan:        p.proj.Plan,
		ProjectPath: p.projPath,
		Root:        p.Root,
		Output:      p.proj.Output,
	}
	if p.proj.Plan != nil {
		if data, err := json.MarshalIndent(p.proj.Plan, "", "  "); err == nil {
			st.PlanJSON = string(data)
		}
		st.Issues = p.proj.Plan.Validate()
	}
	for _, r := range Roles {
		st.Roles = append(st.Roles, RoleState{
			Key: r.Key, Title: r.Title, Hint: r.Hint, Placeholder: r.Placeholder,
			Input: p.proj.Inputs[r.Key],
		})
	}
	st.Prefabs, _ = p.Lib.List()
	if st.Prefabs == nil {
		st.Prefabs = []prefab.Prefab{}
	}
	return st
}

func (p *Pipeline) SetInput(role, text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.proj.Inputs[role] = text
}

func (p *Pipeline) SetGame(game string) {
	if game != "il2" && game != "dcs" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.proj.Game = game
	if p.proj.Plan != nil {
		p.proj.Plan.Game = game
	}
}

// SetPlanJSON replaces the plan with user-edited JSON.
func (p *Pipeline) SetPlanJSON(text string) error {
	mp, err := decodePlan(text)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if mp.Game == "" {
		mp.Game = p.proj.Game
	}
	p.proj.Plan = &mp
	p.proj.Output = nil
	return nil
}

func (p *Pipeline) runPlanPrompt(ctx context.Context, label, prompt string) (plan.MissionPlan, error) {
	logging.Infof("%s gestartet", label)
	logging.Block("PROMPT "+label, prompt)
	cfg := ai.LoadConfig()
	out, err := cfg.Chat(ctx, []ai.Message{{Role: "user", Content: prompt}})
	if err != nil {
		logging.Errorf("%s fehlgeschlagen: %v", label, err)
		return plan.MissionPlan{}, err
	}
	logging.Block("RESULT "+label+" RAW", out)
	mp, err := decodePlan(out)
	if err != nil {
		logging.Errorf("%s JSON-Fehler: %v", label, err)
		return plan.MissionPlan{}, err
	}
	return mp, nil
}

func (p *Pipeline) storePlan(mp plan.MissionPlan, game string) {
	if mp.Game == "" {
		mp.Game = game
	}
	p.mu.Lock()
	p.proj.Plan = &mp
	p.proj.Output = nil
	p.mu.Unlock()
}

func (p *Pipeline) QuickMission(ctx context.Context, input string) (plan.MissionPlan, error) {
	p.mu.Lock()
	game := p.proj.Game
	prompt := fill(p.tmpl[quickFile], map[string]string{
		"GAME_CONTEXT": p.gameContext(game),
		"QUICK_INPUT":  strings.TrimSpace(input),
		"SCHEMA":       p.tmpl[schemaFile],
	})
	p.mu.Unlock()

	mp, err := p.runPlanPrompt(ctx, "quick", prompt)
	if err != nil {
		return plan.MissionPlan{}, err
	}
	p.storePlan(mp, game)
	logging.Infof("QuickMission erfolgreich: %s", mp.Title.De)
	return mp, nil
}

// Merge builds ONE structured prompt from the five user sections and asks the
// LLM for the mission plan.
func (p *Pipeline) Merge(ctx context.Context) (plan.MissionPlan, error) {
	p.mu.Lock()
	game := p.proj.Game
	filled := 0
	for _, r := range Roles {
		if strings.TrimSpace(p.proj.Inputs[r.Key]) != "" {
			filled++
		}
	}
	if filled == 0 {
		p.mu.Unlock()
		return plan.MissionPlan{}, fmt.Errorf("Bitte mindestens einen Abschnitt (z. B. Story) ausfuellen")
	}
	prompt := fill(p.tmpl[mergeFile], map[string]string{
		"GAME_CONTEXT":    p.gameContext(game),
		"USER_STORY":      userSection(p.proj.Inputs["story"]),
		"USER_PLAYER":     userSection(p.proj.Inputs["player"]),
		"USER_ENEMY":      userSection(p.proj.Inputs["enemy"]),
		"USER_RESISTANCE": userSection(p.proj.Inputs["resistance"]),
		"USER_BRIEFING":   userSection(p.proj.Inputs["briefing"]),
		"SCHEMA":          p.tmpl[schemaFile],
	})
	p.mu.Unlock()

	mp, err := p.runPlanPrompt(ctx, "merge", prompt)
	if err != nil {
		return plan.MissionPlan{}, err
	}
	p.storePlan(mp, game)
	logging.Infof("Merge erfolgreich: %s", mp.Title.De)
	return mp, nil
}

// Generate writes the mission files for the plan's game into the configured
// game folder and remembers the result in the project.
func (p *Pipeline) Generate() (*gen.Result, error) {
	p.mu.Lock()
	mp := p.proj.Plan
	p.mu.Unlock()
	if mp == nil {
		return nil, fmt.Errorf("Kein Missionsplan vorhanden - erst Blitz oder Merge ausfuehren")
	}
	paths := LoadPaths()
	var (
		res *gen.Result
		err error
	)
	switch mp.Game {
	case "il2":
		if paths.IL2MissionsDir == "" {
			return nil, fmt.Errorf("IL2_MISSIONS_DIR ist nicht gesetzt (.env)")
		}
		res, err = il2.Generate(mp, paths.IL2MissionsDir)
	case "dcs":
		if paths.DCSMissionsDir == "" {
			return nil, fmt.Errorf("DCS_SAVED_GAMES ist nicht gesetzt (.env)")
		}
		res, err = dcs.GenerateWith(mp, paths.DCSMissionsDir, p.Lib)
	default:
		return nil, fmt.Errorf("unbekanntes Spiel im Plan: %q", mp.Game)
	}
	if err != nil {
		logging.Errorf("Generate (%s) fehlgeschlagen: %v", mp.Game, err)
		return nil, err
	}
	logging.Infof("Generate (%s) erfolgreich: %s", mp.Game, res.MainFile)
	for _, n := range res.Notes {
		logging.Infof("  Hinweis: %s", n)
	}
	p.mu.Lock()
	p.proj.Output = res
	p.mu.Unlock()
	return res, nil
}

var intFields = map[string]bool{
	"country": true, "count": true, "taskType": true, "success": true,
	"counter": true, "delay": true,
	"cloudLevel": true, "cloudHeight": true, "precLevel": true, "seaState": true, "turbulence": true,
}

var floatFields = map[string]bool{
	"x": true, "z": true, "alt": true, "lat": true, "lon": true, "heading": true, "r": true, "radius": true,
	"dx": true, "dy": true, "spacing": true,
}

var boolFields = map[string]bool{"engageable": true}

func normalizeJSON(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if s, ok := val.(string); ok {
				trimmed := strings.TrimSpace(s)
				if intFields[k] {
					if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
						t[k] = n
						continue
					}
				}
				if floatFields[k] {
					if fl, err := strconv.ParseFloat(trimmed, 64); err == nil {
						t[k] = fl
						continue
					}
				}
				if boolFields[k] && (trimmed == "true" || trimmed == "false") {
					t[k] = trimmed == "true"
					continue
				}
			}
			// LLMs sometimes emit {"targets": 7} for the objective counter.
			if k == "counter" {
				if m, ok := val.(map[string]any); ok {
					for _, inner := range m {
						if fl, ok := inner.(float64); ok {
							t[k] = fl
							break
						}
					}
					continue
				}
			}
			t[k] = normalizeJSON(val)
		}
		return t
	case []any:
		for i, val := range t {
			if s, ok := val.(string); ok {
				if fl, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
					t[i] = fl
					continue
				}
			}
			t[i] = normalizeJSON(val)
		}
		return t
	default:
		return v
	}
}

func decodePlan(out string) (plan.MissionPlan, error) {
	raw, err := ai.ExtractJSON(out)
	if err != nil {
		return plan.MissionPlan{}, fmt.Errorf("Antwort enthaelt kein JSON: %w", err)
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return plan.MissionPlan{}, fmt.Errorf("JSON ungueltig: %w", err)
	}
	normalized := normalizeJSON(generic)
	data, err := json.Marshal(normalized)
	if err != nil {
		return plan.MissionPlan{}, err
	}
	logging.Block("PLAN JSON (normalisiert)", string(data))
	var mp plan.MissionPlan
	if err := json.Unmarshal(data, &mp); err != nil {
		return plan.MissionPlan{}, fmt.Errorf("Missionsplan-JSON ungueltig: %w (Details im Log airborne.log)", err)
	}
	return mp, nil
}

func (p *Pipeline) SaveProject(path string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if path == "" {
		path = p.projPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(p.proj, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	p.projPath = path
	return path, nil
}

func (p *Pipeline) LoadProject(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if path == "" {
		path = p.projPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	proj := NewProject()
	if err := json.Unmarshal(data, proj); err != nil {
		return fmt.Errorf("Projektdatei ungueltig: %w", err)
	}
	if proj.Inputs == nil {
		proj.Inputs = map[string]string{}
	}
	p.proj = proj
	p.projPath = path
	return nil
}
