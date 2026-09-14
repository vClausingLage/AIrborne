// Package prefab holds reusable asset layouts ("prefabs") for DCS: a carrier
// group with parked aircraft, a FARP with tents and trucks, a SAM site, ...
// A prefab is described in local metres around an origin and is placed into a
// mission at a lat/lon with a heading; the DCS generator expands it into real
// ship/vehicle/static groups. Prefabs live as JSON files in <root>/prefabs.
package prefab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Prefab struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Game        string    `json:"game"`
	Country     string    `json:"country,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Elements    []Element `json:"elements"`
	Created     string    `json:"created,omitempty"`
}

// Element is one unit (or a row of identical units) inside a prefab.
// dx/dy are metres north/east of the prefab origin BEFORE the placement
// rotation; heading is relative to the prefab heading (0 = prefab forward).
type Element struct {
	Name      string  `json:"name"`
	Kind      string  `json:"kind"` // ship | vehicle | static | plane | helicopter
	Type      string  `json:"type"`
	Category  string  `json:"category,omitempty"`  // DCS static category
	ShapeName string  `json:"shapeName,omitempty"` // DCS static model (Fortifications/Warehouses/...)
	Livery    string  `json:"livery,omitempty"`
	Group     string  `json:"group,omitempty"` // ships/vehicles with the same group name form one DCS group
	Count     int     `json:"count,omitempty"`
	Spacing   float64 `json:"spacing,omitempty"`
	Layout    string  `json:"layout,omitempty"` // row (side by side, default) | column (one behind the other)
	DX        float64 `json:"dx"`
	DY        float64 `json:"dy"`
	Heading   float64 `json:"heading"`
	LinkTo    string  `json:"linkTo,omitempty"` // name of the ship element a deck static is attached to
}

func (e Element) Units() int {
	if e.Count < 1 {
		return 1
	}
	return e.Count
}

// IsStatic reports whether the element becomes a DCS static object.
// Aircraft inside a prefab are always parked statics.
func (e Element) IsStatic() bool {
	switch strings.ToLower(e.Kind) {
	case "static", "plane", "helicopter", "aircraft", "heli":
		return true
	}
	return false
}

func (e Element) IsShip() bool {
	k := strings.ToLower(e.Kind)
	return k == "ship" || k == "naval"
}

var validKinds = map[string]bool{"ship": true, "vehicle": true, "static": true, "plane": true, "helicopter": true}

// Normalize fills defaults from the catalog and returns validation issues.
func (p *Prefab) Normalize() []string {
	var issues []string
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		issues = append(issues, "name fehlt")
	}
	if p.Game == "" {
		p.Game = "dcs"
	}
	if p.ID == "" {
		p.ID = Slug(p.Name)
	}
	if len(p.Elements) == 0 {
		issues = append(issues, "elements ist leer")
	}
	names := map[string]int{}
	ships := map[string]bool{}
	for i := range p.Elements {
		e := &p.Elements[i]
		e.Kind = strings.ToLower(strings.TrimSpace(e.Kind))
		switch e.Kind {
		case "aircraft", "air":
			e.Kind = "plane"
		case "heli":
			e.Kind = "helicopter"
		case "naval", "boat":
			e.Kind = "ship"
		case "building", "object", "fortification":
			e.Kind = "static"
		case "ground", "unit", "truck", "tank":
			e.Kind = "vehicle"
		}
		if !validKinds[e.Kind] {
			issues = append(issues, fmt.Sprintf("elements[%d].kind %q unbekannt (ship|vehicle|static|plane|helicopter)", i, e.Kind))
		}
		e.Type = strings.TrimSpace(e.Type)
		if e.Type == "" {
			issues = append(issues, fmt.Sprintf("elements[%d].type fehlt", i))
		}
		if strings.TrimSpace(e.Name) == "" {
			e.Name = fmt.Sprintf("%s %d", e.Type, i+1)
		}
		if names[e.Name] > 0 {
			e.Name = fmt.Sprintf("%s (%d)", e.Name, names[e.Name]+1)
		}
		names[e.Name]++
		if e.IsShip() {
			ships[e.Name] = true
		}
		if e.Layout == "" {
			e.Layout = "row"
		}
		if e.Kind == "plane" || e.Kind == "helicopter" {
			if e.Category == "" {
				e.Category = "Planes"
				if e.Kind == "helicopter" {
					e.Category = "Helicopters"
				}
			}
		}
		if e.Kind == "static" {
			if info, ok := LookupStatic(e.Type); ok {
				e.Type = info.Type
				if e.Category == "" {
					e.Category = info.Category
				}
				if e.ShapeName == "" {
					e.ShapeName = info.Shape
				}
			}
			if e.Category == "" {
				issues = append(issues, fmt.Sprintf("elements[%d] (%s): Static-Typ nicht im Katalog und ohne category - wird als Fahrzeug platziert", i, e.Type))
			}
		}
	}
	for i, e := range p.Elements {
		if e.LinkTo != "" && !ships[e.LinkTo] {
			issues = append(issues, fmt.Sprintf("elements[%d].linkTo %q ist kein Schiff im Prefab", i, e.LinkTo))
		}
	}
	return issues
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func Slug(s string) string {
	r := strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")
	s = r.Replace(strings.ToLower(strings.TrimSpace(s)))
	s = strings.Trim(slugRe.ReplaceAllString(s, "-"), "-")
	if len(s) > 48 {
		s = strings.TrimRight(s[:48], "-")
	}
	if s == "" {
		s = "prefab"
	}
	return s
}

// Library is the on-disk prefab store (one JSON file per prefab).
type Library struct {
	Dir string
}

func NewLibrary(dir string) *Library { return &Library{Dir: dir} }

func (l *Library) List() ([]Prefab, error) {
	entries, err := os.ReadDir(l.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Prefab{}, nil
		}
		return nil, err
	}
	out := []Prefab{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(l.Dir, e.Name()))
		if err != nil {
			continue
		}
		var p Prefab
		if json.Unmarshal(data, &p) != nil || p.Name == "" {
			continue
		}
		if p.ID == "" {
			p.ID = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

// Find resolves a prefab by id or (case-insensitive) name.
func (l *Library) Find(nameOrID string) (*Prefab, bool) {
	key := strings.TrimSpace(nameOrID)
	if key == "" {
		return nil, false
	}
	list, err := l.List()
	if err != nil {
		return nil, false
	}
	for i := range list {
		if list[i].ID == key {
			return &list[i], true
		}
	}
	low := strings.ToLower(key)
	for i := range list {
		if strings.ToLower(list[i].Name) == low || list[i].ID == Slug(key) {
			return &list[i], true
		}
	}
	return nil, false
}

// Save normalizes and writes the prefab; an existing id is overwritten.
func (l *Library) Save(p Prefab) (Prefab, error) {
	if issues := p.Normalize(); len(issues) > 0 {
		for _, is := range issues {
			// only hard errors block saving
			if strings.Contains(is, "fehlt") || strings.Contains(is, "unbekannt") || strings.Contains(is, "leer") || strings.Contains(is, "linkTo") {
				return p, fmt.Errorf("Prefab ungueltig: %s", strings.Join(issues, "; "))
			}
		}
	}
	if p.Created == "" {
		p.Created = time.Now().Format(time.RFC3339)
	}
	if err := os.MkdirAll(l.Dir, 0o755); err != nil {
		return p, err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return p, err
	}
	return p, os.WriteFile(l.path(p.ID), data, 0o644)
}

func (l *Library) Delete(id string) error {
	id = Slug(id)
	if id == "" {
		return fmt.Errorf("Prefab-ID fehlt")
	}
	return os.Remove(l.path(id))
}

func (l *Library) path(id string) string {
	return filepath.Join(l.Dir, Slug(id)+".json")
}
