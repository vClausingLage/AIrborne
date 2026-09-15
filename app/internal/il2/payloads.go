package il2

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"airborne/internal/gen"
	"airborne/internal/plan"
)

// ---------------------------------------------------------------------------
// IL-2 payloads
//
// IL-2 selects a loadout by PayloadId = index into the plane's payload list
// (the order of the "Payload" dropdown in the mission editor) plus ModMask
// for modifications. The lists live inside the encrypted .gtp archives, so
// they are maintained by hand in <root>/reference/il2-payloads.json:
//
//   {"planes": {"f80c10": {"name": "F-80C-10", "payloads": [
//       {"id": 0, "name": "6x .50 cal", "roles": ["CAP", "Escort"]},
//       {"id": 3, "name": "8x HVAR", "roles": ["CAS", "GroundAttack"], "modMask": "1"}
//   ]}}}
//
// Planes without entries keep PayloadId 0.

// Payload is one entry of a plane's payload list.
type Payload struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Roles   []string `json:"roles,omitempty"`
	ModMask string   `json:"modMask,omitempty"`
}

// PlaneEntry is one plane in the table.
type PlaneEntry struct {
	Name     string    `json:"name"`
	Payloads []Payload `json:"payloads"`
}

// PayloadTable maps the lowercase plane script name (f80c10) to its payloads.
type PayloadTable struct {
	Planes map[string]PlaneEntry `json:"planes"`
	Path   string                `json:"-"`
}

// LoadPayloadTable reads the JSON table; a missing file yields an empty
// table (not an error), so IL-2 export works without it.
func LoadPayloadTable(path string) (*PayloadTable, error) {
	t := &PayloadTable{Planes: map[string]PlaneEntry{}, Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return t, nil
		}
		return t, err
	}
	if err := json.Unmarshal(data, t); err != nil {
		return t, fmt.Errorf("%s: %w", path, err)
	}
	lower := map[string]PlaneEntry{}
	for k, v := range t.Planes {
		lower[strings.ToLower(k)] = v
	}
	t.Planes = lower
	return t, nil
}

// Known reports whether the table lists payloads for the plane.
func (t *PayloadTable) Known(plane string) bool {
	if t == nil {
		return false
	}
	e, ok := t.Planes[strings.ToLower(plane)]
	return ok && len(e.Payloads) > 0
}

// role families: a payload tagged with any role of the family matches a task
// of the same family, so "CAS" also accepts a payload tagged "GroundAttack".
var roleFamilies = map[string]string{
	"cap": "air", "patrol": "air", "intercept": "air", "fightersweep": "air", "sweep": "air", "escort": "air",
	"cas": "ground", "groundattack": "ground", "ground": "ground", "strike": "ground", "pinpointstrike": "ground",
	"bombing": "ground", "attack": "ground", "runwayattack": "ground", "sead": "ground", "antiship": "ground", "naval": "ground",
	"transport": "transport", "cargo": "transport", "recon": "recon", "reconnaissance": "recon",
}

func roleKey(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// Pick chooses the payload for a plane: entries tagged with the task's role
// first (exact role, then same family), best wish match wins, ties keep the
// list order. ok=false when the plane has no entries.
func (t *PayloadTable) Pick(plane, task, wish string) (Payload, bool) {
	if !t.Known(plane) {
		return Payload{}, false
	}
	all := t.Planes[strings.ToLower(plane)].Payloads
	want := roleKey(task)
	family := roleFamilies[want]
	var exact, related []Payload
	for _, p := range all {
		for _, r := range p.Roles {
			rk := roleKey(r)
			if rk == want {
				exact = append(exact, p)
				break
			}
			if family != "" && roleFamilies[rk] == family {
				related = append(related, p)
				break
			}
		}
	}
	candidates := all
	if len(exact) > 0 {
		candidates = exact
	} else if len(related) > 0 {
		candidates = related
	}
	best, bestScore := candidates[0], -1
	for _, p := range candidates {
		if s := gen.PayloadScore(wish, p.Name); s > bestScore {
			best, bestScore = p, s
		}
	}
	return best, true
}

// planeName extracts "f80c10" from a resolved script path.
func planeName(script string) string {
	s := strings.ReplaceAll(script, "\\", "/")
	s = s[strings.LastIndex(s, "/")+1:]
	return strings.ToLower(strings.TrimSuffix(s, ".txt"))
}

// PlaneNames lists the planes in the table (sorted), for diagnostics.
func (t *PayloadTable) PlaneNames() []string {
	var out []string
	for k := range t.Planes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// payloadFor picks PayloadId/ModMask for a group and records the choice.
func (w *writer) payloadFor(g plan.Group, a asset, isPlayer bool) (int, string) {
	plane := planeName(a.script)
	if w.payloads == nil || len(w.payloads.Planes) == 0 {
		if !w.payloadWarned {
			w.note("Keine IL-2 Payload-Tabelle (reference/il2-payloads.json) - alle Flugzeuge mit PayloadId 0")
			w.payloadWarned = true
		}
		return 0, ""
	}
	task := g.Task
	if strings.TrimSpace(task) == "" {
		task = "CAS"
		if !isPlayer {
			task = "CAP"
		}
	}
	p, ok := w.payloads.Pick(plane, task, g.Payload)
	if !ok {
		w.note("Gruppe %s: keine Payloads fuer %s in %s - PayloadId 0 (Bewaffnung im Editor setzen)", g.Name, plane, w.payloads.Path)
		return 0, ""
	}
	w.note("Bewaffnung %s (%s, %s): #%d %s", g.Name, plane, task, p.ID, p.Name)
	return p.ID, p.ModMask
}
