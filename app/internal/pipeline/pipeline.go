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
	"airborne/internal/i18n"
	"airborne/internal/il2"
	"airborne/internal/logging"
	"airborne/internal/plan"
	"airborne/internal/prefab"
)

// RoleDef describes one of the five input sections of the step-by-step mode.
// The user writes free text per section; the sections are merged into ONE
// structured prompt (prompts/06-merge.md) - there are no per-role LLM calls.
type RoleDef struct {
	Key         string
	Title       plan.Localized
	Placeholder plan.Localized
	Hint        plan.Localized
}

var Roles = []RoleDef{
	{Key: "story", Title: plan.Localized{De: "1 · Story / Lage", En: "1 · Story / situation"},
		Hint: plan.Localized{
			De: "Krieg, Jahr/Monat, Region, taktische Lage, warum dieser Einsatz, Wetter, Tageszeit",
			En: "War, year/month, region, tactical situation, why this sortie, weather, time of day"},
		Placeholder: plan.Localized{
			De: "z. B. September 1950, Vorabend der Landung bei Inchon. Nordkoreanische Il-10 sollen UN-Aufklärungstrupps im Hügelland nordwestlich der Stadt ausschalten…",
			En: "e.g. September 1950, eve of the Inchon landing. North Korean Il-10s are to take out UN recon teams in the hills northwest of the city…"}},
	{Key: "player", Title: plan.Localized{De: "2 · Player", En: "2 · Player"},
		Hint: plan.Localized{
			De: "Flugzeugtyp, Anzahl Spielerplätze, Fraktion, Callsign, Startart/Ort, Bewaffnung",
			En: "Aircraft type, number of player slots, faction, callsign, start type/location, loadout"},
		Placeholder: plan.Localized{
			De: "z. B. 2x Il-10, KPAF, Luftstart über der Küste westlich von Inchon, Bomben + Raketen…",
			En: "e.g. 2x Il-10, KPAF, air start over the coast west of Inchon, bombs + rockets…"}},
	{Key: "enemy", Title: plan.Localized{De: "3 · Gegner", En: "3 · Enemy"},
		Hint: plan.Localized{
			De: "Feindliche Einheiten Luft/Boden, Typen, Anzahl, Positionen, Verhalten (statisch/Patrouille/Kolonne)",
			En: "Enemy air/ground units, types, numbers, positions, behaviour (static/patrol/column)"},
		Placeholder: plan.Localized{
			De: "z. B. Lastwagen und Jeeps unter Tarnnetzen im Hügelland, eine Jeep-Patrouille Richtung Kimpo, evtl. ein Paar F-51D als Jagdschutz…",
			En: "e.g. trucks and jeeps under camouflage nets in the hills, a jeep patrol heading for Kimpo, maybe a pair of F-51Ds as fighter cover…"}},
	{Key: "resistance", Title: plan.Localized{De: "4 · Widerstand", En: "4 · Resistance"},
		Hint: plan.Localized{
			De: "Flak/Luftabwehr, Bodenwiderstand, Verbündete, statische Objekte – realistisch dosiert",
			En: "Flak/air defence, ground resistance, allies, static objects – in realistic doses"},
		Placeholder: plan.Localized{
			De: "z. B. zwei Bofors-Geschütze und ein M16-Flakpanzer beim Hauptversteck, Tarnnetze über den Fahrzeugen…",
			En: "e.g. two Bofors guns and an M16 AA half-track at the main hideout, camouflage nets over the vehicles…"}},
	{Key: "briefing", Title: plan.Localized{De: "5 · Briefing / Funk", En: "5 · Briefing / radio"},
		Hint: plan.Localized{
			De: "Ton und Sprache des Briefings, Callsigns, Funksprüche (Intro, Kontakt, Erfolg, Zeitlimit)",
			En: "Tone and language of the briefing, callsigns, radio messages (intro, contact, success, time limit)"},
		Placeholder: plan.Localized{
			De: "z. B. propagandistischer KPA-Ton, Callsign „Sowol“, Meldungen von „Basis“: Einleitung nach 10 s, Feindkontakt im Zielgebiet, Erfolgsmeldung, Zeitlimit 25 min…",
			En: "e.g. propagandistic KPA tone, callsign \"Sowol\", messages from \"Base\": intro after 10 s, enemy contact in the target area, success report, time limit 25 min…"}},
}

const (
	systemFile    = "00-system.md"
	mergeFile     = "06-merge.md"
	quickFile     = "07-quick.md"
	prefabFile    = "08-prefab.md"
	challengeFile = "09-challenge.md"
	repairFile    = "10-repair.md"
	schemaFile    = "schema.md"
)

type Project struct {
	Game   string            `json:"game"`
	Inputs map[string]string `json:"inputs"`
	Plan   *plan.MissionPlan `json:"plan,omitempty"`
	Output *gen.Result       `json:"output,omitempty"`
	// Repair is the outcome of the repair round for Plan (nil = no problems
	// were found, or the plan was edited by hand).
	Repair *RepairInfo `json:"repair,omitempty"`
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
	// MaxTokens is the effective per-plan output limit, MaxTokensEnv the
	// .env default it falls back to when the user setting is 0.
	MaxTokens    int    `json:"maxTokens"`
	MaxTokensEnv int    `json:"maxTokensEnv"`
	SettingsPath string `json:"settingsPath"`
	// Language is the UI and message language ("de" or "en").
	Language string `json:"language"`
	// LastUsage is the token accounting of the most recent LLM call.
	LastUsage *ai.Usage `json:"lastUsage"`
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
	Repair      *RepairInfo       `json:"repair"`
	Output      *gen.Result       `json:"output"`
	Prefabs     []prefab.Prefab   `json:"prefabs"`
	// Aircraft lists the flyable types offered in the challenge mode (per game).
	Aircraft map[string][]AircraftOption `json:"aircraft"`
	// Maps lists the DCS theatres the challenge mode can be pinned to.
	Maps []string `json:"maps"`
}

type Pipeline struct {
	mu         sync.Mutex
	Root       string
	PromptsDir string
	tmpl       map[string]string
	proj       *Project
	projPath   string
	Lib        *prefab.Library
	places     []Place
	settings   Settings
	lastUsage  *ai.Usage
	// chatFn overrides the LLM transport (tests); nil = ai.Config.ChatUsage.
	chatFn func(context.Context, []ai.Message) (string, ai.Usage, error)
}

func New(root string) (*Pipeline, error) {
	p := &Pipeline{Root: root}
	p.PromptsDir = filepath.Join(root, "prompts")
	if err := p.loadTemplates(); err != nil {
		return nil, err
	}
	p.Lib = prefab.NewLibrary(filepath.Join(root, "prefabs"))
	p.places = loadPlaces(root)
	p.loadSettings()
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
	for _, f := range []string{systemFile, mergeFile, quickFile, prefabFile, challengeFile, repairFile, schemaFile} {
		data, err := os.ReadFile(filepath.Join(p.PromptsDir, f))
		if err != nil {
			return i18n.Errorf("Prompt-Template fehlt: %s (%w)", "Prompt template missing: %s (%w)", f, err)
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
	DCSRoot        string
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
		DCSRoot:        clean("DCS_ROOT"),
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
	env := ai.LoadConfig()
	cfg := p.aiConfig()
	paths := LoadPaths()
	p.mu.Lock()
	lastUsage := p.lastUsage
	p.mu.Unlock()
	st := State{
		Roles:  []RoleState{},
		Issues: []string{},
		Config: ConfigState{
			Base: cfg.Base, Model: cfg.Model, HasKey: cfg.Key != "",
			PromptsDir: p.PromptsDir, LogPath: logging.Path(),
			IL2MissionsDir: paths.IL2MissionsDir, IL2Editor: paths.IL2Editor, DCSMissionsDir: paths.DCSMissionsDir,
			MaxTokens: cfg.MaxTokens, MaxTokensEnv: env.MaxTokens, SettingsPath: p.settingsPath(), Language: i18n.Lang(),
			LastUsage: lastUsage,
		},
		Game:        p.proj.Game,
		Plan:        p.proj.Plan,
		Repair:      p.proj.Repair,
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
	lang := i18n.Lang()
	for _, r := range Roles {
		st.Roles = append(st.Roles, RoleState{
			Key: r.Key, Title: r.Title.Pick(lang), Hint: r.Hint.Pick(lang), Placeholder: r.Placeholder.Pick(lang),
			Input: p.proj.Inputs[r.Key],
		})
	}
	st.Prefabs, _ = p.Lib.List()
	if st.Prefabs == nil {
		st.Prefabs = []prefab.Prefab{}
	}
	st.Aircraft = map[string][]AircraftOption{"il2": IL2Aircraft, "dcs": DCSAircraft}
	st.Maps = DCSMaps
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
	p.proj.Repair = nil
	return nil
}

func (p *Pipeline) runPlanPrompt(ctx context.Context, label, game, prompt string) (plan.MissionPlan, error) {
	logging.Infof("%s gestartet", label)
	system := p.systemPrompt(game)
	logging.Block("SYSTEM "+label, system)
	logging.Block("PROMPT "+label, prompt)
	out, err := p.chat(ctx, []ai.Message{{Role: "system", Content: system}, {Role: "user", Content: prompt}})
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

// storePlan makes mp the project plan; repair is the outcome of the repair
// round (nil when the plan had no problems).
func (p *Pipeline) storePlan(mp plan.MissionPlan, game string, repair *RepairInfo) {
	if mp.Game == "" {
		mp.Game = game
	}
	p.mu.Lock()
	// User-provided media never comes from the LLM: carry it over.
	if mp.Media == nil && p.proj.Plan != nil && !p.proj.Plan.Media.IsEmpty() {
		mp.Media = p.proj.Plan.Media
	}
	p.proj.Plan = &mp
	p.proj.Output = nil
	p.proj.Repair = repair
	p.mu.Unlock()
}

// SetMedia attaches briefing picture / kneeboards to the current plan.
func (p *Pipeline) SetMedia(m plan.Media) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proj.Plan == nil {
		return i18n.Errorf("Kein Missionsplan vorhanden - erst Blitz, Herausforderung oder Merge ausfuehren", "No mission plan yet - run Blitz, Challenge or Merge first")
	}
	if m.IsEmpty() {
		p.proj.Plan.Media = nil
	} else {
		mm := m
		p.proj.Plan.Media = &mm
	}
	p.proj.Output = nil
	return nil
}

// challengePrompt renders prompts/09-challenge.md for the chosen aircraft.
func (p *Pipeline) challengePrompt(game, aircraft, mapHint string) string {
	hint := strings.TrimSpace(mapHint)
	if hint == "" {
		hint = "frei waehlbar (passend zu Flugzeug und Epoche)"
		if game == "il2" {
			hint = "Korea"
		}
	}
	label := aircraft
	for _, opt := range append(append([]AircraftOption{}, IL2Aircraft...), DCSAircraft...) {
		if opt.Type == aircraft {
			label = fmt.Sprintf("%s (%s)", opt.Label, opt.Type)
			break
		}
	}
	return fill(p.tmpl[challengeFile], map[string]string{
		"GAME_CONTEXT": p.gameContext(game, mapHint),
		"AIRCRAFT":     label,
		"MAP_HINT":     hint,
		"SCHEMA":       p.tmpl[schemaFile],
	})
}

// challengeTrim applies the challenge-mode invariants the LLM must not
// override: flagged as challenge, no map icons, a single player slot.
func challengeTrim(mp *plan.MissionPlan) {
	mp.Challenge = true
	mp.Icons = nil
	if len(mp.PlayerGroups) > 0 {
		mp.PlayerGroups[0].Count = 1
	}
}

// Challenge builds a blind skirmish for the chosen aircraft: the LLM designs
// the whole scenario and is told to keep every enemy detail out of the
// player-facing texts; the plan is flagged so the generators hide the rest.
func (p *Pipeline) Challenge(ctx context.Context, aircraft, mapHint string) (plan.MissionPlan, error) {
	aircraft = strings.TrimSpace(aircraft)
	if aircraft == "" {
		return plan.MissionPlan{}, i18n.Errorf("Bitte ein Flugzeug waehlen", "Please choose an aircraft")
	}
	p.mu.Lock()
	game := p.proj.Game
	prompt := p.challengePrompt(game, aircraft, mapHint)
	p.mu.Unlock()

	mp, err := p.runPlanPrompt(ctx, "challenge", game, prompt)
	if err != nil {
		return plan.MissionPlan{}, err
	}
	// The challenge flag must be set before the check so leaked enemy details
	// count as problems; the LLM may re-add icons, so the trim runs twice.
	challengeTrim(&mp)
	mp, repair := p.repair(ctx, "challenge", game, mp)
	challengeTrim(&mp)
	p.storePlan(mp, game, repair)
	logging.Infof("Challenge erfolgreich: %s (%s)", mp.Title.De, aircraft)
	return mp, nil
}

func (p *Pipeline) QuickMission(ctx context.Context, input string) (plan.MissionPlan, error) {
	p.mu.Lock()
	game := p.proj.Game
	prompt := fill(p.tmpl[quickFile], map[string]string{
		"GAME_CONTEXT": p.gameContext(game, input),
		"QUICK_INPUT":  strings.TrimSpace(input),
		"SCHEMA":       p.tmpl[schemaFile],
	})
	p.mu.Unlock()

	mp, err := p.runPlanPrompt(ctx, "quick", game, prompt)
	if err != nil {
		return plan.MissionPlan{}, err
	}
	mp, repair := p.repair(ctx, "quick", game, mp)
	p.storePlan(mp, game, repair)
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
		return plan.MissionPlan{}, i18n.Errorf("Bitte mindestens einen Abschnitt (z. B. Story) ausfuellen", "Please fill in at least one section (e.g. Story)")
	}
	prompt := fill(p.tmpl[mergeFile], map[string]string{
		"GAME_CONTEXT":    p.gameContext(game, userText(p.proj.Inputs)),
		"USER_STORY":      userSection(p.proj.Inputs["story"]),
		"USER_PLAYER":     userSection(p.proj.Inputs["player"]),
		"USER_ENEMY":      userSection(p.proj.Inputs["enemy"]),
		"USER_RESISTANCE": userSection(p.proj.Inputs["resistance"]),
		"USER_BRIEFING":   userSection(p.proj.Inputs["briefing"]),
		"SCHEMA":          p.tmpl[schemaFile],
	})
	p.mu.Unlock()

	mp, err := p.runPlanPrompt(ctx, "merge", game, prompt)
	if err != nil {
		return plan.MissionPlan{}, err
	}
	mp, repair := p.repair(ctx, "merge", game, mp)
	p.storePlan(mp, game, repair)
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
		return nil, i18n.Errorf("Kein Missionsplan vorhanden - erst Blitz oder Merge ausfuehren", "No mission plan yet - run Blitz or Merge first")
	}
	paths := LoadPaths()
	outDir := paths.IL2MissionsDir
	if mp.Game == "dcs" {
		outDir = paths.DCSMissionsDir
	}
	if outDir == "" {
		if mp.Game == "dcs" {
			return nil, i18n.Errorf("DCS_SAVED_GAMES ist nicht gesetzt (.env)", "DCS_SAVED_GAMES is not set (.env)")
		}
		return nil, i18n.Errorf("IL2_MISSIONS_DIR ist nicht gesetzt (.env)", "IL2_MISSIONS_DIR is not set (.env)")
	}
	res, err := p.generateTo(mp, outDir)
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

// generateTo runs the plan's generator into outDir with the payload data the
// export uses (IL-2 table, DCS presets, prefab library). The repair loop
// calls it with a scratch folder to learn what the generator complains about.
func (p *Pipeline) generateTo(mp *plan.MissionPlan, outDir string) (*gen.Result, error) {
	paths := LoadPaths()
	switch mp.Game {
	case "il2":
		table, terr := il2.LoadPayloadTable(filepath.Join(p.Root, "reference", "il2-payloads.json"))
		if terr != nil {
			logging.Errorf("IL-2 Payload-Tabelle: %v", terr)
		}
		logging.Infof("IL-2 Payload-Tabelle: %d Flugzeuge (%s)", len(table.Planes), table.Path)
		return il2.GenerateOpts(mp, outDir, table)
	case "dcs":
		payloads := dcs.LoadPayloadDB(paths.DCSRoot, paths.DCSSavedGames)
		logging.Infof("DCS-Bewaffnungs-Presets: %d Typen (DCS_ROOT=%s)", payloads.Files, paths.DCSRoot)
		scripts := dcs.LoadScripts(p.Root)
		if scripts == nil && len(mp.Behaviors) > 0 {
			logging.Errorf("MOOSE nicht gefunden: %s", filepath.Join(p.Root, dcs.MooseRelPath))
		}
		return dcs.GenerateOpts(mp, outDir, p.Lib, payloads, scripts)
	default:
		return nil, i18n.Errorf("unbekanntes Spiel im Plan: %q", "unknown game in plan: %q", mp.Game)
	}
}

var intFields = map[string]bool{
	"country": true, "count": true, "taskType": true, "success": true,
	"counter": true, "delay": true, "interval": true, "waves": true,
	"cloudLevel": true, "cloudHeight": true, "precLevel": true, "seaState": true, "turbulence": true,
}

var floatFields = map[string]bool{
	"x": true, "z": true, "alt": true, "lat": true, "lon": true, "heading": true, "r": true, "radius": true, "speed": true,
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
		return plan.MissionPlan{}, i18n.Errorf("Antwort enthaelt kein JSON: %w", "Response contains no JSON: %w", err)
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return plan.MissionPlan{}, i18n.Errorf("JSON ungueltig: %w", "Invalid JSON: %w", err)
	}
	normalized := normalizeJSON(generic)
	data, err := json.Marshal(normalized)
	if err != nil {
		return plan.MissionPlan{}, err
	}
	logging.Block("PLAN JSON (normalisiert)", string(data))
	var mp plan.MissionPlan
	if err := json.Unmarshal(data, &mp); err != nil {
		return plan.MissionPlan{}, i18n.Errorf("Missionsplan-JSON ungueltig: %w (Details im Log airborne.log)", "Invalid mission plan JSON: %w (details in airborne.log)", err)
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
		return i18n.Errorf("Projektdatei ungueltig: %w", "Invalid project file: %w", err)
	}
	if proj.Inputs == nil {
		proj.Inputs = map[string]string{}
	}
	p.proj = proj
	p.projPath = path
	return nil
}

// NewProject discards inputs, plan and output and starts an empty project in
// the default autosave slot (projects/current.json). The selected game is kept.
func (p *Pipeline) NewProject() {
	p.mu.Lock()
	defer p.mu.Unlock()
	game := p.proj.Game
	p.proj = NewProject()
	p.proj.Game = game
	p.projPath = filepath.Join(p.Root, "projects", "current.json")
}

// GameName is the human-readable game name used in prompts.
func GameName(game string) string {
	if game == "dcs" {
		return "DCS World"
	}
	return "IL-2 Sturmovik: Korea"
}

// systemPrompt renders prompts/00-system.md (role, game, design principles);
// it is sent as the system message before every plan prompt.
func (p *Pipeline) systemPrompt(game string) string {
	return fill(p.tmpl[systemFile], map[string]string{"GAME_NAME": GameName(game)})
}
