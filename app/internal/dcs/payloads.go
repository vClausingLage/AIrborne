package dcs

import (
	"airborne/internal/gen"

	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// DCS mission-editor payload presets
//
// DCS ships one Lua file per aircraft type with the loadouts offered in the
// mission editor ("UnitPayloads"). Instead of inventing pylon CLSIDs we let
// the plan name a role (task) and a free-text wish and pick the stock preset
// that fits best. Sources, in order (later files add/override by type):
//   <DCS_ROOT>\MissionEditor\data\scripts\UnitPayloads\*.lua   (AI-only types)
//   <DCS_ROOT>\CoreMods\**\UnitPayloads\*.lua                  (flyable modules)
//   <DCS_ROOT>\Mods\aircraft\*\UnitPayloads\*.lua
//   <DCS_SAVED_GAMES>\MissionEditor\UnitPayloads\*.lua          (user presets)

// Preset is one mission-editor loadout.
type Preset struct {
	Name   string
	Pylons map[int]string // pylon number -> weapon CLSID
	Tasks  []int          // ME task ids this preset is offered for
}

// PayloadDB holds presets by DCS unit type name.
type PayloadDB struct {
	byType map[string][]Preset
	Files  int
}

// ME task ids (MissionEditor task roster; same numbering as pydcs).
const (
	taskIntercept      = 10
	taskCAP            = 11
	taskRefueling      = 13
	taskAWACS          = 14
	taskNothing        = 15
	taskAFAC           = 16
	taskReconnaissance = 17
	taskEscort         = 18
	taskFighterSweep   = 19
	taskSEAD           = 29
	taskAntishipStrike = 30
	taskCAS            = 31
	taskGroundAttack   = 32
	taskPinpointStrike = 33
	taskRunwayAttack   = 34
	taskTransport      = 35
)

// role describes how a plan task maps onto DCS.
type role struct {
	key     string   // canonical name
	dcsTask string   // group "task" string in the .miz
	chain   []int    // preferred ME task ids for preset selection, best first
	targets []string // EngageTargets target types for AI groups (nil = none)
}

var roles = []role{
	{"CAP", "CAP", []int{taskCAP, taskFighterSweep, taskIntercept, taskEscort}, []string{"Air"}},
	{"Intercept", "Intercept", []int{taskIntercept, taskCAP, taskFighterSweep}, []string{"Air"}},
	{"FighterSweep", "Fighter Sweep", []int{taskFighterSweep, taskCAP, taskIntercept}, []string{"Air"}},
	{"Escort", "Escort", []int{taskEscort, taskCAP, taskFighterSweep}, []string{"Air"}},
	{"CAS", "CAS", []int{taskCAS, taskGroundAttack, taskPinpointStrike, taskAFAC}, []string{"Helicopters", "Ground Units", "Light armed ships"}},
	{"GroundAttack", "Ground Attack", []int{taskGroundAttack, taskCAS, taskPinpointStrike}, []string{"Ground Units", "Light armed ships"}},
	{"Strike", "Pinpoint Strike", []int{taskPinpointStrike, taskGroundAttack, taskRunwayAttack, taskCAS}, nil},
	{"RunwayAttack", "Runway Attack", []int{taskRunwayAttack, taskPinpointStrike, taskGroundAttack}, nil},
	{"SEAD", "SEAD", []int{taskSEAD, taskPinpointStrike, taskGroundAttack}, []string{"Air Defence"}},
	{"AntiShip", "Antiship Strike", []int{taskAntishipStrike, taskPinpointStrike, taskGroundAttack}, []string{"Ships"}},
	{"AFAC", "AFAC", []int{taskAFAC, taskCAS}, nil},
	{"Recon", "Reconnaissance", []int{taskReconnaissance, taskNothing}, nil},
	{"Transport", "Transport", []int{taskTransport, taskNothing}, nil},
	{"AWACS", "AWACS", []int{taskAWACS}, nil},
	{"Refueling", "Refueling", []int{taskRefueling}, nil},
	{"Nothing", "Nothing", []int{taskNothing}, nil},
}

var roleAliases = map[string]string{
	"cap": "CAP", "patrol": "CAP", "combatairpatrol": "CAP",
	"intercept": "Intercept", "interception": "Intercept",
	"fightersweep": "FighterSweep", "sweep": "FighterSweep",
	"escort": "Escort",
	"cas":    "CAS", "closeairsupport": "CAS",
	"groundattack": "GroundAttack", "ground": "GroundAttack",
	"strike": "Strike", "pinpointstrike": "Strike", "pinpoint": "Strike", "bombing": "Strike", "attack": "Strike",
	"runwayattack": "RunwayAttack", "runway": "RunwayAttack",
	"sead": "SEAD", "dead": "SEAD",
	"antiship": "AntiShip", "antishipstrike": "AntiShip", "ship": "AntiShip", "naval": "AntiShip",
	"afac": "AFAC", "fac": "AFAC",
	"recon": "Recon", "reconnaissance": "Recon",
	"transport": "Transport", "cargo": "Transport",
	"awacs": "AWACS", "refueling": "Refueling", "tanker": "Refueling",
	"nothing": "Nothing", "none": "Nothing",
}

// lookupRole resolves a plan task string; ok=false when unknown/empty.
func lookupRole(s string) (role, bool) {
	key := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
	name, ok := roleAliases[key]
	if !ok {
		return role{}, false
	}
	for _, r := range roles {
		if r.key == name {
			return r, true
		}
	}
	return role{}, false
}

// LoadPayloadDB scans the DCS installation and Saved Games for preset files.
// Missing folders are skipped; an empty DB is returned when nothing is found.
func LoadPayloadDB(dcsRoot, savedGames string) *PayloadDB {
	db := &PayloadDB{byType: map[string][]Preset{}}
	var roots []string
	if dcsRoot != "" {
		roots = append(roots,
			filepath.Join(dcsRoot, "MissionEditor", "data", "scripts", "UnitPayloads"),
			filepath.Join(dcsRoot, "CoreMods"),
			filepath.Join(dcsRoot, "Mods", "aircraft"),
		)
	}
	if savedGames != "" {
		roots = append(roots, filepath.Join(savedGames, "MissionEditor", "UnitPayloads"))
	}
	for _, root := range roots {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(path), ".lua") || !strings.EqualFold(filepath.Base(filepath.Dir(path)), "UnitPayloads") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if typ, presets := parsePayloadFile(string(data)); typ != "" && len(presets) > 0 {
				db.byType[typ] = presets
				db.Files++
			}
			return nil
		})
	}
	return db
}

// Presets returns the presets of a unit type (case-insensitive).
func (db *PayloadDB) Presets(typ string) []Preset {
	if db == nil {
		return nil
	}
	if p, ok := db.byType[typ]; ok {
		return p
	}
	for k, p := range db.byType {
		if strings.EqualFold(k, typ) {
			return p
		}
	}
	return nil
}

// Pick chooses the preset for a unit type: presets offered for the role's
// task (walking the fallback chain), best match to the free-text wish first.
// ok=false when the type has no presets at all.
func (db *PayloadDB) Pick(typ string, r role, wish string) (Preset, bool) {
	all := db.Presets(typ)
	if len(all) == 0 {
		return Preset{}, false
	}
	candidates := all
	for _, task := range r.chain {
		var c []Preset
		for _, p := range all {
			for _, t := range p.Tasks {
				if t == task {
					c = append(c, p)
					break
				}
			}
		}
		if len(c) > 0 {
			candidates = c
			break
		}
	}
	best, bestScore := candidates[0], -1
	for _, p := range candidates {
		score := gen.PayloadScore(wish, p.Name)
		if score > bestScore {
			best, bestScore = p, score
		}
	}
	return best, true
}

// pylonTable renders a preset as the .miz "pylons" table.
func pylonTable(p Preset) T {
	nums := make([]int, 0, len(p.Pylons))
	for n := range p.Pylons {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	t := tbl()
	for _, n := range nums {
		t = append(t, k(n, tbl(k("CLSID", p.Pylons[n]))))
	}
	return t
}

// ---------------------------------------------------------------------------
// minimal reader for the UnitPayloads Lua files:
//   local unitPayloads = { ["name"] = "MiG-21Bis", ["payloads"] = { [1] = {...}, ... } }
//   return unitPayloads

func parsePayloadFile(src string) (string, []Preset) {
	i := strings.Index(src, "{")
	if i < 0 {
		return "", nil
	}
	lp := &presetParser{s: src, i: i}
	v := lp.value()
	root, ok := v.(presetTable)
	if !ok {
		return "", nil
	}
	typ, _ := root["name"].(string)
	var presets []Preset
	if pl, ok := root["payloads"].(presetTable); ok {
		for _, key := range sortedKeys(pl) {
			pt, ok := pl[key].(presetTable)
			if !ok {
				continue
			}
			p := Preset{Pylons: map[int]string{}}
			p.Name, _ = pt["name"].(string)
			if py, ok := pt["pylons"].(presetTable); ok {
				for _, pk := range sortedKeys(py) {
					e, ok := py[pk].(presetTable)
					if !ok {
						continue
					}
					clsid, _ := e["CLSID"].(string)
					num, _ := e["num"].(float64)
					if clsid != "" && num > 0 {
						p.Pylons[int(num)] = clsid
					}
				}
			}
			if ts, ok := pt["tasks"].(presetTable); ok {
				for _, tk := range sortedKeys(ts) {
					if n, ok := ts[tk].(float64); ok {
						p.Tasks = append(p.Tasks, int(n))
					}
				}
			}
			if p.Name != "" {
				presets = append(presets, p)
			}
		}
	}
	return typ, presets
}

type presetTable map[string]any

// sortedKeys orders numeric keys numerically, then strings.
func sortedKeys(t presetTable) []string {
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		na, ea := strconv.Atoi(keys[a])
		nb, eb := strconv.Atoi(keys[b])
		if ea == nil && eb == nil {
			return na < nb
		}
		if ea == nil || eb == nil {
			return ea == nil
		}
		return keys[a] < keys[b]
	})
	return keys
}

type presetParser struct {
	s string
	i int
}

func (p *presetParser) skip() {
	for p.i < len(p.s) {
		c := p.s[p.i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.i++
		case c == '-' && p.i+1 < len(p.s) && p.s[p.i+1] == '-':
			for p.i < len(p.s) && p.s[p.i] != '\n' {
				p.i++
			}
		default:
			return
		}
	}
}

func isIdent(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func (p *presetParser) value() any {
	p.skip()
	if p.i >= len(p.s) {
		return nil
	}
	switch c := p.s[p.i]; {
	case c == '{':
		return p.table()
	case c == '"':
		return p.str()
	case c == '-' || (c >= '0' && c <= '9'):
		j := p.i + 1
		for j < len(p.s) && (p.s[j] == '.' || p.s[j] == 'e' || p.s[j] == 'E' || p.s[j] == '-' || p.s[j] == '+' || (p.s[j] >= '0' && p.s[j] <= '9')) {
			j++
		}
		f, _ := strconv.ParseFloat(p.s[p.i:j], 64)
		p.i = j
		return f
	default:
		j := p.i
		for j < len(p.s) && isIdent(p.s[j]) {
			j++
		}
		word := p.s[p.i:j]
		if j == p.i {
			p.i++ // unknown character: skip so we always make progress
		} else {
			p.i = j
		}
		switch word {
		case "true":
			return true
		case "false":
			return false
		}
		return nil
	}
}

func (p *presetParser) str() string {
	p.i++ // opening quote
	var b strings.Builder
	for p.i < len(p.s) {
		c := p.s[p.i]
		if c == '\\' && p.i+1 < len(p.s) {
			p.i++
			switch p.s[p.i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(p.s[p.i])
			}
			p.i++
			continue
		}
		if c == '"' {
			p.i++
			break
		}
		b.WriteByte(c)
		p.i++
	}
	return b.String()
}

func (p *presetParser) table() presetTable {
	t := presetTable{}
	p.i++ // {
	auto := 1
	for {
		p.skip()
		if p.i >= len(p.s) {
			return t
		}
		if p.s[p.i] == '}' {
			p.i++
			return t
		}
		var key string
		if p.s[p.i] == '[' {
			p.i++
			switch kv := p.value().(type) {
			case string:
				key = kv
			case float64:
				key = strconv.Itoa(int(kv))
			}
			p.skip()
			if p.i < len(p.s) && p.s[p.i] == ']' {
				p.i++
			}
			p.skip()
			if p.i < len(p.s) && p.s[p.i] == '=' {
				p.i++
			}
		} else if isIdent(p.s[p.i]) && !(p.s[p.i] >= '0' && p.s[p.i] <= '9') {
			// bare identifier key (name = ...) or a bare value like true
			j := p.i
			for j < len(p.s) && isIdent(p.s[j]) {
				j++
			}
			save := p.i
			word := p.s[p.i:j]
			p.i = j
			p.skip()
			if p.i < len(p.s) && p.s[p.i] == '=' {
				p.i++
				key = word
			} else {
				p.i = save
				key = strconv.Itoa(auto)
				auto++
			}
		} else {
			key = strconv.Itoa(auto)
			auto++
		}
		t[key] = p.value()
		p.skip()
		if p.i < len(p.s) && (p.s[p.i] == ',' || p.s[p.i] == ';') {
			p.i++
		}
	}
}
