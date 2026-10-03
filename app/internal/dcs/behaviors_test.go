package dcs

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"

	"airborne/internal/plan"
)

// behaviorPlan extends samplePlan (player Syria = red, enemy Israel = blue)
// with an air defence network, a respawning reserve and a scramble.
func behaviorPlan() *plan.MissionPlan {
	mp := samplePlan()
	mp.EnemyGroups = append(mp.EnemyGroups,
		plan.Group{Name: "Hawk Site", Kind: "vehicle", Script: "Roland ADS", Count: 2, Country: "Israel",
			Position: &plan.Position{Lat: 33.0, Lon: 35.65}},
		plan.Group{Name: "EWR North", Kind: "vehicle", Script: "FPS-117", Count: 1, Country: "Israel",
			Position: &plan.Position{Lat: 33.2, Lon: 35.6}},
		plan.Group{Name: "Reserve", Kind: "vehicle", Script: "M-60", Count: 2, Country: "Israel",
			Position: &plan.Position{Lat: 33.05, Lon: 35.7}},
	)
	mp.Behaviors = []plan.Behavior{
		{Type: "iads", Groups: []string{"Hawk Site"}, Sensors: []string{"EWR North"}},
		{Type: "respawn", Groups: []string{"Reserve"}, Waves: 2, Interval: 240},
		{Type: "scramble", Groups: []string{"Kfir CAP"}, Zone: &plan.Zone{Lat: 33.1, Lon: 35.72, R: 30000}},
	}
	return mp
}

var stubScripts = &Scripts{Moose: []byte("-- MOOSE stub\n"), MooseFile: "Moose_.lua"}

// enemyGroups returns the blue (Israel) groups of a category keyed by name.
func enemyGroups(t *testing.T, m map[string]any, cat string) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	israel := get(t, m, "coalition", "blue", "country", "1").(map[string]any)
	for _, g := range get(t, israel, cat, "group").(map[string]any) {
		gm := g.(map[string]any)
		out[gm["name"].(string)] = gm
	}
	return out
}

func TestBehaviorsMiz(t *testing.T) {
	mp := behaviorPlan()
	if issues := mp.Validate(); len(issues) != 0 {
		t.Fatalf("plan should be valid: %v", issues)
	}
	res, err := GenerateOpts(mp, t.TempDir(), nil, nil, stubScripts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 {
		t.Fatalf("unexpected problems: %v", res.Problems)
	}
	files := readMiz(t, res.MainFile)
	m := parseLua(t, "mission", string(files["mission"]))
	resources := parseLua(t, "mapResource", string(files["l10n/DEFAULT/mapResource"]))

	// MANTIS finds its groups by name prefix, SPAWN/scramble groups wait.
	vehicles := enemyGroups(t, m, "vehicle")
	for _, name := range []string{"AB_IADS1_SAM Hawk Site", "AB_IADS1_EWR EWR North"} {
		if _, ok := vehicles[name]; !ok {
			t.Fatalf("group %q missing (have %v)", name, groupNames(vehicles))
		}
	}
	if vehicles["Reserve"]["lateActivation"] != true {
		t.Fatal("respawn template must be late-activated")
	}
	if vehicles["APC column"]["lateActivation"] != false {
		t.Fatal("ordinary groups must stay active")
	}
	kfir := enemyGroups(t, m, "plane")["Kfir CAP"]
	if kfir["lateActivation"] != true {
		t.Fatal("scramble group must be late-activated")
	}

	// "all destroyed" must not wait for the never-activated respawn template.
	reserveID := int(vehicles["Reserve"]["groupId"].(float64))
	conds := get(t, m, "trig", "conditions").(map[string]any)
	if c := conds["3"].(string); !strings.Contains(c, "c_group_dead(") || strings.Contains(c, "c_group_dead("+strconv.Itoa(reserveID)+")") {
		t.Fatalf("all_destroyed condition: %s", c)
	}

	// trigger 4: scramble on zone entry, trigger 5: MISSION START script loader
	rules := get(t, m, "trigrules").(map[string]any)
	if len(rules) != 5 {
		t.Fatalf("trigrules = %d, want 5", len(rules))
	}
	kfirID := int(kfir["groupId"].(float64))
	if a := get(t, m, "trig", "actions", "4").(string); a != "a_activate_group("+strconv.Itoa(kfirID)+"); mission.trig.func[4]=nil;" {
		t.Fatalf("scramble action: %s", a)
	}
	if c := conds["4"].(string); !strings.HasPrefix(c, "return(c_part_of_group_in_zone(1, ") {
		t.Fatalf("scramble condition: %s", c)
	}
	if get(t, rules, "4", "actions", "1", "predicate") != "a_activate_group" || get(t, rules, "4", "actions", "1", "group") != float64(kfirID) {
		t.Fatalf("scramble rule: %v", rules["4"])
	}

	start := get(t, rules, "5").(map[string]any)
	if start["predicate"] != "triggerStart" {
		t.Fatalf("loader predicate = %v", start["predicate"])
	}
	mooseKey := get(t, start, "actions", "1", "file").(string)
	behKey := get(t, start, "actions", "2", "file").(string)
	if resources[mooseKey] != "Moose_.lua" || resources[behKey] != BehaviorScriptFile {
		t.Fatalf("mapResource: %v (keys %s, %s)", resources, mooseKey, behKey)
	}
	want := `a_do_script_file(getValueResourceByKey("` + mooseKey + `"));a_do_script_file(getValueResourceByKey("` + behKey + `"));`
	if a := get(t, m, "trig", "actions", "5").(string); a != want {
		t.Fatalf("loader action:\n got %s\nwant %s", a, want)
	}
	if conds["5"] != "return(true)" || get(t, m, "trig", "funcStartup", "5") == nil {
		t.Fatal("loader must be a MISSION START trigger (funcStartup)")
	}
	if _, ok := get(t, m, "trig", "func").(map[string]any)["5"]; ok {
		t.Fatal("MISSION START trigger must not be in trig.func")
	}
	if string(files["l10n/DEFAULT/Moose_.lua"]) != "-- MOOSE stub\n" {
		t.Fatal("MOOSE not embedded")
	}

	// Run the generated script against MOOSE stubs and check the calls.
	calls, errs := runBehaviorScript(t, string(files["l10n/DEFAULT/"+BehaviorScriptFile]))
	if len(errs) != 0 {
		t.Fatalf("script errors: %v", errs)
	}
	wantCalls := []string{
		`MANTIS:New("AB_IADS1", "AB_IADS1_SAM", "AB_IADS1_EWR", nil, "blue", false)`,
		`MANTIS:Start()`,
		`SPAWN:New("Reserve")`,
		`SPAWN:InitLimit(2, 3)`,
		`SPAWN:SpawnScheduled(240, 0.3)`,
	}
	if strings.Join(calls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("MOOSE calls:\n%s\nwant:\n%s", strings.Join(calls, "\n"), strings.Join(wantCalls, "\n"))
	}
}

// Without MOOSE the script behaviors are skipped with a note (not a problem:
// the plan is fine), while the native scramble still works.
func TestBehaviorsWithoutMoose(t *testing.T) {
	res, err := GenerateOpts(behaviorPlan(), t.TempDir(), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 {
		t.Fatalf("missing MOOSE must not be a plan problem: %v", res.Problems)
	}
	if !strings.Contains(strings.Join(res.Notes, "\n"), "MOOSE") {
		t.Fatalf("expected a MOOSE note: %v", res.Notes)
	}
	files := readMiz(t, res.MainFile)
	if _, ok := files["l10n/DEFAULT/"+BehaviorScriptFile]; ok {
		t.Fatal("no behavior script without MOOSE")
	}
	m := parseLua(t, "mission", string(files["mission"]))
	vehicles := enemyGroups(t, m, "vehicle")
	if _, ok := vehicles["Hawk Site"]; !ok || vehicles["Reserve"]["lateActivation"] != false {
		t.Fatalf("groups should be written unchanged: %v", groupNames(vehicles))
	}
	if len(get(t, m, "trigrules").(map[string]any)) != 4 { // 3 radios + scramble
		t.Fatal("scramble trigger missing")
	}
}

// If every target respawns, "all destroyed" cannot work; that is a plan
// problem the repair round can fix.
func TestRespawnAllTargetsIsProblem(t *testing.T) {
	mp := samplePlan()
	mp.EnemyGroups = mp.EnemyGroups[:2] // without the F-5 pair, which would count as a target
	mp.Behaviors = []plan.Behavior{{Type: "respawn", Groups: []string{"APC column", "Tanks"}}}
	res, err := GenerateOpts(mp, t.TempDir(), nil, nil, stubScripts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(res.Problems, "\n"), "respawn") {
		t.Fatalf("expected respawn problem, got %v", res.Problems)
	}
}

// The pinned MOOSE build must at least compile as Lua 5.1 (catches a broken
// download or a file that is not MOOSE at all).
func TestPinnedMooseCompiles(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	sc := LoadScripts(root)
	if sc == nil {
		t.Skip("scripts/moose/Moose_.lua not present")
	}
	if !strings.Contains(string(sc.Moose[:200]), "MOOSE") {
		t.Fatal("Moose_.lua does not look like a MOOSE build")
	}
	L := lua.NewState()
	defer L.Close()
	if _, err := L.LoadString(string(sc.Moose)); err != nil {
		t.Fatalf("Moose_.lua does not compile: %v", err)
	}
	// MANTIS:Start is an FSM transition added in MANTIS:New.
	for _, fn := range []string{"function MANTIS:New(", "AddTransition(\"Stopped\",\"Start\",\"Running\")", "function SPAWN:New(", "function SPAWN:InitLimit(", "function SPAWN:SpawnScheduled("} {
		if !strings.Contains(string(sc.Moose), fn) {
			t.Fatalf("MOOSE build lacks %s", fn)
		}
	}
}

// runBehaviorScript executes the generated script with recording stubs for
// the MOOSE classes it uses and returns the calls ("CLASS:Method(args)").
func runBehaviorScript(t *testing.T, script string) (calls, errs []string) {
	t.Helper()
	L := lua.NewState()
	defer L.Close()
	stubs := `
calls, errors = {}, {}
env = { error = function(m) table.insert(errors, m) end, info = function() end }
local function fmt(v)
  if type(v) == "string" then return string.format("%q", v) end
  return tostring(v)
end
local function class(name, methods)
  local c = {}
  c.__index = c
  for _, m in ipairs(methods) do
    c[m] = function(self, ...)
      local n, args = select("#", ...), {}
      for i = 1, n do args[i] = fmt((select(i, ...))) end
      table.insert(calls, name .. ":" .. m .. "(" .. table.concat(args, ", ") .. ")")
      if m == "New" then return setmetatable({}, c) end
      return self
    end
  end
  return c
end
MANTIS = class("MANTIS", {"New", "Start"})
SPAWN = class("SPAWN", {"New", "InitLimit", "SpawnScheduled"})
`
	if err := L.DoString(stubs); err != nil {
		t.Fatal(err)
	}
	if err := L.DoString(script); err != nil {
		t.Fatalf("behavior script fails: %v\n%s", err, script)
	}
	read := func(name string) []string {
		var out []string
		L.GetGlobal(name).(*lua.LTable).ForEach(func(_, v lua.LValue) { out = append(out, v.String()) })
		return out
	}
	return read("calls"), read("errors")
}

func groupNames(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	return out
}
