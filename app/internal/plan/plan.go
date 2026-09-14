package plan

import (
	"fmt"
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
	return issues
}
