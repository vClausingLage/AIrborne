package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type MissionPlan struct {
	Game           string            `json:"game"`
	Title          Localized         `json:"title"`
	Author         string            `json:"author"`
	Map            string            `json:"map"`
	Date           string            `json:"date"`
	Time           string            `json:"time"`
	Weather        Weather           `json:"weather"`
	PlayerGroups   []Group           `json:"playerGroups"`
	EnemyGroups    []Group           `json:"enemyGroups"`
	FriendlyGroups []Group           `json:"friendlyGroups,omitempty"`
	Flak           []Flak            `json:"flak"`
	Statics        []StaticObject    `json:"statics"`
	Objectives     []Objective       `json:"objectives"`
	RadioQueue     []Radio           `json:"radioQueue"`
	Briefing       Localized         `json:"briefing"`
	Icons          []Icon            `json:"icons"`
	Prefabs        []PrefabPlacement `json:"prefabs,omitempty"`
	// Media is user-provided content that never passes through the LLM
	// (briefing picture, kneeboard pages). Preserved when a plan is regenerated.
	Media *Media `json:"media,omitempty"`
	// Challenge marks a "Herausforderung": the generators hide every hint about
	// the opposition (no target icons, DCS groups hidden on the map, F10 view
	// restricted) and Validate warns when briefing/radio texts leak enemy types.
	Challenge bool `json:"challenge,omitempty"`
}

// Media references files on disk that are copied into the mission as-is.
type Media struct {
	// BriefingImage is a PNG/JPG shown on the mission's briefing screen
	// (IL-2: <Mission>.png next to the .Mission; DCS: l10n/DEFAULT + pictureFileName).
	BriefingImage string `json:"briefingImage,omitempty"`
	// Kneeboards are PNG pages copied to KNEEBOARD/IMAGES in the .miz (DCS only).
	Kneeboards []string `json:"kneeboards,omitempty"`
	// KneeboardBriefing renders the briefing text as the first kneeboard page (DCS only).
	KneeboardBriefing bool `json:"kneeboardBriefing,omitempty"`
}

// IsEmpty reports whether no media is attached.
func (m *Media) IsEmpty() bool {
	return m == nil || (m.BriefingImage == "" && len(m.Kneeboards) == 0 && !m.KneeboardBriefing)
}

// PrefabPlacement drops a saved prefab (see internal/prefab) into the mission
// at a position/heading. DCS only. Side decides the coalition.
type PrefabPlacement struct {
	Prefab   string   `json:"prefab"` // prefab id or name
	Name     string   `json:"name,omitempty"`
	Side     string   `json:"side"` // player | friendly | enemy
	Country  any      `json:"country,omitempty"`
	Position Position `json:"position"`
}

// IsEnemy reports whether the placement belongs to the opposing coalition.
func (pp PrefabPlacement) IsEnemy() bool {
	switch strings.ToLower(strings.TrimSpace(pp.Side)) {
	case "enemy", "red", "gegner", "opfor", "hostile":
		return true
	}
	return false
}

type Localized struct {
	De string `json:"de"`
	En string `json:"en"`
}

// Pick returns the text for the given language, falling back to the other one.
func (l Localized) Pick(lang string) string {
	if lang == "de" {
		if l.De != "" {
			return l.De
		}
		return l.En
	}
	if l.En != "" {
		return l.En
	}
	return l.De
}

type Weather struct {
	CloudLevel  int      `json:"cloudLevel"`
	CloudHeight int      `json:"cloudHeight"`
	PrecLevel   int      `json:"precLevel"`
	CloudConfig string   `json:"cloudConfig"`
	SeaState    int      `json:"seaState"`
	Turbulence  int      `json:"turbulence"`
	WindLayers  [][3]int `json:"windLayers"`
}

// Position: IL-2 uses x/z (meters), DCS uses lat/lon (degrees). Alt in meters.
type Position struct {
	X    float64 `json:"x,omitempty"`
	Z    float64 `json:"z,omitempty"`
	Alt  float64 `json:"alt,omitempty"`
	Lat  float64 `json:"lat,omitempty"`
	Lon  float64 `json:"lon,omitempty"`
	Head float64 `json:"heading,omitempty"`
}

type Start struct {
	Type    string     `json:"type"`
	X       float64    `json:"x,omitempty"`
	Z       float64    `json:"z,omitempty"`
	Lat     float64    `json:"lat,omitempty"`
	Lon     float64    `json:"lon,omitempty"`
	Alt     float64    `json:"alt"`
	Heading float64    `json:"heading"`
	Route   []Position `json:"route,omitempty"`
}

func (s Start) Position() Position {
	return Position{X: s.X, Z: s.Z, Lat: s.Lat, Lon: s.Lon, Alt: s.Alt, Head: s.Heading}
}

type Movement struct {
	Type string `json:"type"`
}

type Group struct {
	Name        string     `json:"name"`
	Kind        string     `json:"kind,omitempty"`
	Aircraft    string     `json:"aircraft,omitempty"`
	Script      string     `json:"script,omitempty"`
	Count       int        `json:"count,omitempty"`
	Country     any        `json:"country,omitempty"`
	CountryName string     `json:"countryName,omitempty"`
	Start       *Start     `json:"start,omitempty"`
	Position    *Position  `json:"position,omitempty"`
	Formation   []int      `json:"formation,omitempty"`
	Callsign    []int      `json:"callsign,omitempty"`
	Movement    *Movement  `json:"movement,omitempty"`
	Route       []Position `json:"route,omitempty"`
	Payload     string     `json:"payload,omitempty"`
	Task        string     `json:"task,omitempty"` // DCS role: CAP, CAS, SEAD, Strike, ...
	Notes       string     `json:"notes,omitempty"`
}

// Units returns the number of units in the group (at least 1).
func (g Group) Units() int {
	if g.Count < 1 {
		return 1
	}
	return g.Count
}

// TypeName returns the unit type: aircraft for air groups, script otherwise.
func (g Group) TypeName() string {
	if g.Aircraft != "" {
		return g.Aircraft
	}
	return g.Script
}

// IsAir reports whether the group is an aircraft/helicopter group.
func (g Group) IsAir() bool {
	switch strings.ToLower(g.Kind) {
	case "plane", "aircraft", "helicopter", "heli", "air":
		return true
	}
	return g.Aircraft != "" && g.Script == ""
}

func (g Group) IsShip() bool {
	k := strings.ToLower(g.Kind)
	return k == "ship" || k == "naval" || strings.HasPrefix(strings.ToLower(g.Script), "ships/")
}

// Moves reports whether the group follows its route.
func (g Group) Moves() bool {
	if len(g.Route) == 0 {
		return false
	}
	if g.Movement == nil {
		return true
	}
	t := strings.ToLower(g.Movement.Type)
	return t != "static" && t != ""
}

// Anchor is the group's reference position (start for air groups).
func (g Group) Anchor() Position {
	if g.Start != nil {
		return g.Start.Position()
	}
	if g.Position != nil {
		return *g.Position
	}
	return Position{}
}

type Flak struct {
	Script      string   `json:"script"`
	Count       int      `json:"count"`
	Position    Position `json:"position"`
	Country     any      `json:"country,omitempty"`
	CountryName string   `json:"countryName,omitempty"`
	Engageable  bool     `json:"engageable"`
	Radius      float64  `json:"radius,omitempty"`
}

type StaticObject struct {
	Kind        string     `json:"kind"`
	Script      string     `json:"script"`
	Count       int        `json:"count"`
	Country     any        `json:"country,omitempty"`
	CountryName string     `json:"countryName,omitempty"`
	Positions   []Position `json:"positions"`
}

type Objective struct {
	Title    Localized `json:"title"`
	Desc     Localized `json:"desc"`
	TaskType int       `json:"taskType,omitempty"`
	Success  int       `json:"success,omitempty"`
	Counter  int       `json:"counter"`
}

type Zone struct {
	X   float64 `json:"x,omitempty"`
	Z   float64 `json:"z,omitempty"`
	Lat float64 `json:"lat,omitempty"`
	Lon float64 `json:"lon,omitempty"`
	R   float64 `json:"r"`
}

func (z Zone) Position() Position {
	return Position{X: z.X, Z: z.Z, Lat: z.Lat, Lon: z.Lon}
}

type Radio struct {
	Trigger string `json:"trigger"`
	Delay   int    `json:"delay,omitempty"`
	Zone    *Zone  `json:"zone,omitempty"`
	Counter int    `json:"counter,omitempty"`
	Speaker string `json:"speaker"`
	TextDe  string `json:"textDe"`
	TextEn  string `json:"textEn"`
}

// Text returns the radio text with speaker prefix for the given language.
func (r Radio) Text(lang string) string {
	t := Localized{De: r.TextDe, En: r.TextEn}.Pick(lang)
	sp := strings.TrimSpace(r.Speaker)
	if sp != "" && !strings.HasPrefix(strings.ToLower(t), strings.ToLower(sp)) {
		return sp + ": " + t
	}
	return t
}

type Icon struct {
	From  Position  `json:"from"`
	To    Position  `json:"to"`
	Label Localized `json:"label"`
	Desc  Localized `json:"desc"`
}

// CountryInt converts a plan country value (number or numeric string) to an int.
func CountryInt(v any) (int, bool) {
	switch t := v.(type) {
	case float64:
		return int(t), true
	case int:
		return t, true
	case int64:
		return int(t), true
	case string:
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &n); err == nil {
			return n, true
		}
	}
	return 0, false
}

// CountryString converts a plan country value to a country name string.
func CountryString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return fmt.Sprintf("%d", int(t))
	}
	return ""
}

func (p *MissionPlan) Validate() []string {
	var issues []string
	if p.Game != "il2" && p.Game != "dcs" {
		issues = append(issues, "game muss \"il2\" oder \"dcs\" sein")
	}
	if p.Title.De == "" && p.Title.En == "" {
		issues = append(issues, "title (de/en) fehlt")
	}
	if p.Date == "" {
		issues = append(issues, "date fehlt")
	}
	if p.Time == "" {
		issues = append(issues, "time fehlt")
	}
	if p.Map == "" {
		issues = append(issues, "map fehlt")
	}
	if len(p.PlayerGroups) == 0 {
		issues = append(issues, "playerGroups ist leer")
	}
	if p.Briefing.De == "" && p.Briefing.En == "" {
		issues = append(issues, "briefing (de/en) fehlt")
	}
	if len(p.Objectives) == 0 {
		issues = append(issues, "objectives ist leer")
	}
	for i, g := range p.PlayerGroups {
		if g.Start == nil {
			issues = append(issues, fmt.Sprintf("playerGroups[%d].start fehlt (Startposition)", i))
			continue
		}
		if p.Game == "dcs" && g.Start.Lat == 0 && g.Start.Lon == 0 && g.Start.X == 0 && g.Start.Z == 0 {
			issues = append(issues, fmt.Sprintf("playerGroups[%d].start ohne lat/lon", i))
		}
		if g.TypeName() == "" {
			issues = append(issues, fmt.Sprintf("playerGroups[%d].aircraft fehlt", i))
		}
	}
	for i, g := range p.EnemyGroups {
		if g.TypeName() == "" {
			issues = append(issues, fmt.Sprintf("enemyGroups[%d].script fehlt", i))
		}
		if g.Position == nil && g.Start == nil {
			issues = append(issues, fmt.Sprintf("enemyGroups[%d].position fehlt", i))
		}
	}
	for i, pp := range p.Prefabs {
		if strings.TrimSpace(pp.Prefab) == "" {
			issues = append(issues, fmt.Sprintf("prefabs[%d].prefab fehlt", i))
		}
		switch strings.ToLower(strings.TrimSpace(pp.Side)) {
		case "player", "friendly", "enemy", "red", "blue", "gegner", "opfor", "hostile", "":
		default:
			issues = append(issues, fmt.Sprintf("prefabs[%d].side %q unbekannt (player|friendly|enemy)", i, pp.Side))
		}
		if p.Game == "dcs" && pp.Position.Lat == 0 && pp.Position.Lon == 0 {
			issues = append(issues, fmt.Sprintf("prefabs[%d].position ohne lat/lon", i))
		}
		if p.Game == "il2" {
			issues = append(issues, fmt.Sprintf("prefabs[%d]: Prefabs werden nur fuer DCS unterstuetzt", i))
		}
	}
	if p.Media != nil {
		if p.Media.BriefingImage != "" && !isImageFile(p.Media.BriefingImage) {
			issues = append(issues, "media.briefingImage: Datei fehlt oder ist kein PNG/JPG: "+p.Media.BriefingImage)
		}
		for i, kb := range p.Media.Kneeboards {
			if !isImageFile(kb) {
				issues = append(issues, fmt.Sprintf("media.kneeboards[%d]: Datei fehlt oder ist kein PNG/JPG: %s", i, kb))
			}
		}
		if p.Game == "il2" && (len(p.Media.Kneeboards) > 0 || p.Media.KneeboardBriefing) {
			issues = append(issues, "media.kneeboards: Kneeboards gibt es nur in DCS (werden ignoriert)")
		}
	}
	if p.Challenge {
		issues = append(issues, p.challengeLeaks()...)
	}
	return issues
}

func isImageFile(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg":
		return true
	}
	return false
}

// EnemyTerms lists the unit type names of the opposition (used to detect
// briefings that give the enemy away in challenge mode).
func (p *MissionPlan) EnemyTerms() []string {
	seen := map[string]bool{}
	var terms []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if i := strings.LastIndexAny(s, "/\\"); i >= 0 {
			s = s[i+1:]
		}
		key := strings.ToLower(s)
		if len(key) < 3 || seen[key] {
			return
		}
		seen[key] = true
		terms = append(terms, s)
	}
	for _, g := range p.EnemyGroups {
		add(g.Aircraft)
		add(g.Script)
	}
	for _, f := range p.Flak {
		add(f.Script)
	}
	return terms
}

// PlayerTexts returns every text the player gets to read or hear.
func (p *MissionPlan) PlayerTexts() map[string]string {
	t := map[string]string{
		"briefing.de": p.Briefing.De, "briefing.en": p.Briefing.En,
	}
	for i, o := range p.Objectives {
		t[fmt.Sprintf("objectives[%d]", i)] = o.Title.De + " " + o.Title.En + " " + o.Desc.De + " " + o.Desc.En
	}
	for i, r := range p.RadioQueue {
		t[fmt.Sprintf("radioQueue[%d]", i)] = r.TextDe + " " + r.TextEn
	}
	for i, ic := range p.Icons {
		t[fmt.Sprintf("icons[%d]", i)] = ic.Label.De + " " + ic.Label.En + " " + ic.Desc.De + " " + ic.Desc.En
	}
	return t
}

// challengeLeaks reports player-facing texts that name an enemy unit type.
func (p *MissionPlan) challengeLeaks() []string {
	var issues []string
	terms := p.EnemyTerms()
	fields := p.PlayerTexts()
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		text := strings.ToLower(fields[n])
		for _, term := range terms {
			if strings.Contains(text, strings.ToLower(term)) {
				issues = append(issues, fmt.Sprintf("Herausforderung: %s verraet den Gegner (%q)", n, term))
			}
		}
	}
	if len(p.Icons) > 0 {
		issues = append(issues, "Herausforderung: icons werden nicht exportiert (Zielgebiet bleibt verborgen)")
	}
	return issues
}

