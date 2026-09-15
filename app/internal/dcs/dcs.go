// Package dcs turns a MissionPlan into a DCS World .miz file. The archive
// layout and Lua structures follow docs/dcs-miz-workflow.md (verified against
// a reference mission); the mission is built from scratch, no template needed.
package dcs

import (
	"archive/zip"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"airborne/internal/gen"
	"airborne/internal/plan"
)

type side struct {
	name      string // "blue" | "red"
	countries []string
	groups    map[string]map[string][]T // country -> category -> groups
}

func newSide(name string) *side {
	return &side{name: name, groups: map[string]map[string][]T{}}
}

func (s *side) add(country, category string, g T) {
	if _, ok := s.groups[country]; !ok {
		s.countries = append(s.countries, country)
		s.groups[country] = map[string][]T{}
	}
	s.groups[country][category] = append(s.groups[country][category], g)
}

type builder struct {
	mp            *plan.MissionPlan
	terr          terrain
	notes         []string
	groupID       int
	unitID        int
	zoneID        int
	dictN         int
	dict          T
	actions       T
	conds         T
	funcs         T
	flags         T
	rules         T
	zones         T
	player        *side
	enemy         *side
	assigned      map[string]string // country -> side name
	llWarned      bool
	prefabs       PrefabSource
	payloads      *PayloadDB
	payloadWarned bool

	playerGroupID int
	enemyGroupIDs []int
	center        [2]float64
}

func (b *builder) note(format string, args ...any) {
	b.notes = append(b.notes, fmt.Sprintf(format, args...))
}

func (b *builder) nextGroup() int { b.groupID++; return b.groupID }
func (b *builder) nextUnit() int  { b.unitID++; return b.unitID }
func (b *builder) nextZone() int  { b.zoneID++; return b.zoneID }

// Generate writes NAME.miz into outDir (without prefab support).
func Generate(mp *plan.MissionPlan, outDir string) (*gen.Result, error) {
	return GenerateWith(mp, outDir, nil)
}

// GenerateWith writes NAME.miz into outDir; prefabs resolves plan.Prefabs.
func GenerateWith(mp *plan.MissionPlan, outDir string, prefabs PrefabSource) (*gen.Result, error) {
	return GenerateOpts(mp, outDir, prefabs, nil)
}

// GenerateOpts is GenerateWith plus the mission-editor payload presets used
// to arm aircraft (nil = empty pylons).
func GenerateOpts(mp *plan.MissionPlan, outDir string, prefabs PrefabSource, payloads *PayloadDB) (*gen.Result, error) {
	if len(mp.PlayerGroups) == 0 || mp.PlayerGroups[0].Start == nil {
		return nil, fmt.Errorf("Plan hat keine Spielergruppe mit Startposition")
	}
	terr, ok := lookupTerrain(mp.Map)
	if !ok {
		return nil, fmt.Errorf("DCS-Karte %q unbekannt (bekannt: Caucasus, Syria, PersianGulf, Nevada, Normandy, TheChannel, MarianaIslands, Falklands, Sinai, Kola)", mp.Map)
	}
	b := &builder{mp: mp, terr: terr, zoneID: 100, assigned: map[string]string{}, prefabs: prefabs, payloads: payloads}
	mission := b.build()

	name := gen.MissionName(mp.Title.Pick("en"))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("Missionsordner nicht erreichbar: %w", err)
	}
	out := filepath.Join(outDir, name+".miz")
	files := map[string]string{
		"mission":                  Serialize("mission", mission),
		"options":                  Serialize("options", optionsTable()),
		"warehouses":               Serialize("warehouses", tbl(k("airports", tbl()), k("warehouses", tbl()))),
		"theatre":                  terr.theatre,
		"l10n/DEFAULT/dictionary":  Serialize("dictionary", b.dict),
		"l10n/DEFAULT/mapResource": Serialize("mapResource", tbl()),
	}
	if err := writeZip(out, files); err != nil {
		return nil, err
	}
	return &gen.Result{
		Game: "dcs", Name: name, OutputDir: outDir, MainFile: out,
		Files: []string{out}, Notes: gen.NonNil(b.notes),
	}, nil
}

func writeZip(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: n, Method: zip.Deflate})
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(files[n])); err != nil {
			return err
		}
	}
	return zw.Close()
}

// ---------------------------------------------------------------------------
// coordinates

func (b *builder) xy(p plan.Position) (float64, float64) {
	if p.Lat != 0 || p.Lon != 0 {
		return b.terr.toXY(p.Lat, p.Lon)
	}
	// Tolerate lat/lon that ended up in x/z.
	if (p.X != 0 || p.Z != 0) && math.Abs(p.X) <= 90 && math.Abs(p.Z) <= 180 {
		if !b.llWarned {
			b.note("Positionen ohne lat/lon: x/z als Breite/Laenge interpretiert")
			b.llWarned = true
		}
		return b.terr.toXY(p.X, p.Z)
	}
	return p.X, p.Z
}

func rad(deg float64) float64 { return deg * math.Pi / 180 }

// ---------------------------------------------------------------------------
// mission

func (b *builder) build() T {
	mp := b.mp
	player := mp.PlayerGroups[0]
	playerCountry := b.country(player.Country, "USA")
	playerRed := redCountries[playerCountry]
	if playerRed {
		b.player, b.enemy = newSide("red"), newSide("blue")
	} else {
		b.player, b.enemy = newSide("blue"), newSide("red")
	}

	sx, sy := b.xy(player.Start.Position())
	b.center = [2]float64{sx, sy}

	b.playerGroupID = b.airGroup(player, b.player, true, playerCountry)
	for _, g := range mp.EnemyGroups {
		c := b.sideCountry(g.Country, b.enemy)
		if g.IsAir() {
			b.airGroup(g, b.enemy, false, c)
		} else {
			id := b.groundGroup(g, b.enemy, c)
			b.enemyGroupIDs = append(b.enemyGroupIDs, id)
		}
	}
	for _, g := range mp.FriendlyGroups {
		c := b.sideCountry(g.Country, b.player)
		if g.IsAir() {
			b.airGroup(g, b.player, false, c)
		} else {
			b.groundGroup(g, b.player, c)
		}
	}
	for i, f := range mp.Flak {
		c, sd := b.resolveSide(f.Country, b.enemy)
		g := plan.Group{Name: fmt.Sprintf("Flak %d", i+1), Kind: "vehicle", Script: f.Script, Count: f.Count,
			Position: &f.Position}
		b.groundGroup(g, sd, c)
	}
	for i, st := range mp.Statics {
		if len(st.Positions) == 0 || st.Script == "" {
			continue
		}
		c, sd := b.resolveSide(st.Country, b.enemy)
		for j, p := range st.Positions {
			pp := p
			g := plan.Group{Name: fmt.Sprintf("Static %d-%d", i+1, j+1), Kind: "vehicle", Script: st.Script, Count: 1, Position: &pp}
			b.groundGroup(g, sd, c)
		}
	}
	b.placePrefabs()

	// ---- texts -------------------------------------------------------------
	briefing := mp.Briefing.Pick("de")
	if mp.Briefing.En != "" && mp.Briefing.De != "" {
		briefing = mp.Briefing.De + "\n\n--- EN ---\n" + mp.Briefing.En
	}
	task := ""
	if len(mp.Objectives) > 0 {
		o := mp.Objectives[0]
		task = strings.TrimSpace(o.Title.Pick("de") + "\n" + o.Desc.Pick("de"))
	}
	descKey := b.dictKey("descriptionText", briefing)
	blueTask, redTask := task, ""
	if playerRed {
		blueTask, redTask = "", task
	}
	blueKey := b.dictKey("descriptionBlueTask", blueTask)
	redKey := b.dictKey("descriptionRedTask", redTask)
	neutralKey := b.dictKey("descriptionNeutralsTask", "")
	sortieKey := b.dictKey("sortie", mp.Title.Pick("de"))

	// ---- triggers ----------------------------------------------------------
	for _, r := range mp.RadioQueue {
		b.radio(r)
	}

	// ---- assemble ----------------------------------------------------------
	day, month, year := parseDate(mp.Date)
	h, m, s := parseTime(mp.Time)

	blueSide, redSide := b.player, b.enemy
	if playerRed {
		blueSide, redSide = b.enemy, b.player
	}
	used := map[int]bool{}
	blueIDs, redIDs := arr(), arr()
	for _, c := range blueSide.countries {
		blueIDs = append(blueIDs, k(len(blueIDs)+1, countryIDs[c]))
		used[countryIDs[c]] = true
	}
	for _, c := range redSide.countries {
		redIDs = append(redIDs, k(len(redIDs)+1, countryIDs[c]))
		used[countryIDs[c]] = true
	}
	neutralIDs := arr()
	ids := make([]int, 0, len(countryIDs))
	for _, id := range countryIDs {
		if !used[id] {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		neutralIDs = append(neutralIDs, k(len(neutralIDs)+1, id))
	}

	tx, ty := b.center[0], b.center[1]
	if len(b.enemyGroupIDs) > 0 {
		a := mp.EnemyGroups[0].Anchor()
		tx, ty = b.xy(a)
	}

	return tbl(
		k("groundControl", groundControl()),
		k("requiredModules", tbl()),
		k("date", tbl(k("Day", day), k("Year", year), k("Month", month))),
		k("trig", tbl(
			k("actions", b.actions),
			k("events", tbl()),
			k("custom", tbl()),
			k("func", b.funcs),
			k("flag", b.flags),
			k("conditions", b.conds),
			k("customStartup", tbl()),
			k("funcStartup", tbl()),
		)),
		k("maxDictId", b.dictN),
		k("result", tbl(
			k("offline", tbl(k("conditions", tbl()), k("actions", tbl()), k("func", tbl()))),
			k("total", 0),
			k("blue", tbl(k("conditions", tbl()), k("actions", tbl()), k("func", tbl()))),
			k("red", tbl(k("conditions", tbl()), k("actions", tbl()), k("func", tbl()))),
		)),
		k("pictureFileNameN", tbl()),
		k("descriptionNeutralsTask", neutralKey),
		k("pictureFileNameServer", tbl()),
		k("weather", b.weather(month)),
		k("theatre", b.terr.theatre),
		k("triggers", tbl(k("zones", b.zones))),
		k("map", tbl(k("centerY", b.center[1]), k("zoom", 100000), k("centerX", b.center[0]))),
		k("coalitions", tbl(k("neutrals", neutralIDs), k("blue", blueIDs), k("red", redIDs))),
		k("descriptionText", descKey),
		k("pictureFileNameR", tbl()),
		k("descriptionBlueTask", blueKey),
		k("goals", tbl()),
		k("descriptionRedTask", redKey),
		k("pictureFileNameB", tbl()),
		k("coalition", tbl(
			k("neutrals", tbl(k("bullseye", point(0, 0)), k("nav_points", tbl()), k("name", "neutrals"), k("country", tbl()))),
			k("blue", b.coalition(blueSide, tx, ty)),
			k("red", b.coalition(redSide, tx, ty)),
		)),
		k("sortie", sortieKey),
		k("version", 23),
		k("trigrules", b.rules),
		k("currentKey", 1000+b.unitID),
		k("failures", tbl()),
		k("forcedOptions", tbl()),
		k("start_time", h*3600+m*60+s),
	)
}

func (b *builder) coalition(s *side, bx, by float64) T {
	countries := tbl()
	for i, c := range s.countries {
		entry := tbl(k("id", countryIDs[c]), k("name", c))
		cats := []string{"plane", "helicopter", "vehicle", "ship", "static"}
		for _, cat := range cats {
			groups := s.groups[c][cat]
			if len(groups) == 0 {
				continue
			}
			list := tbl()
			for j, g := range groups {
				list = append(list, k(j+1, g))
			}
			entry = append(entry, k(cat, tbl(k("group", list))))
		}
		countries = append(countries, k(i+1, entry))
	}
	return tbl(
		k("bullseye", point(bx, by)),
		k("nav_points", tbl()),
		k("name", s.name),
		k("country", countries),
	)
}

// country resolves a plan country value to a canonical DCS country name.
func (b *builder) country(v any, def string) string {
	s := plan.CountryString(v)
	if c, ok := canonicalCountry(s); ok {
		return c
	}
	if s != "" {
		b.note("Country %q unbekannt - %s verwendet", s, def)
	}
	return def
}

// sideCountry resolves a country for the given side; a country already used
// by the other side is replaced by that side's CJTF placeholder.
func (b *builder) sideCountry(v any, s *side) string {
	def := "USA"
	if s.name == "red" {
		def = "Russia"
	}
	c := b.country(v, def)
	if owner, ok := b.assigned[c]; ok && owner != s.name {
		repl := "Combined Joint Task Forces Blue"
		if s.name == "red" {
			repl = "Combined Joint Task Forces Red"
		}
		b.note("Country %s ist bereits auf Seite %s - %s verwendet", c, owner, repl)
		c = repl
	}
	b.assigned[c] = s.name
	return c
}

// resolveSide is the lenient variant for flak/statics: a country that is
// already in use keeps its side, an unknown one lands on the preferred side.
func (b *builder) resolveSide(v any, preferred *side) (string, *side) {
	def := "USA"
	if preferred.name == "red" {
		def = "Russia"
	}
	c := b.country(v, def)
	if owner, ok := b.assigned[c]; ok {
		if owner == b.player.name {
			return c, b.player
		}
		return c, b.enemy
	}
	b.assigned[c] = preferred.name
	return c, preferred
}

// ---------------------------------------------------------------------------
// groups

func (b *builder) airGroup(g plan.Group, s *side, isPlayer bool, country string) int {
	b.assigned[country] = s.name
	typ := g.TypeName()
	heli := strings.ToLower(g.Kind) == "helicopter" || isHelicopterType(typ)
	cat := "plane"
	if heli {
		cat = "helicopter"
	}
	anchor := g.Anchor()
	x, y := b.xy(anchor)
	altM := anchor.Alt
	if altM <= 0 {
		altM = 2000
		if heli {
			altM = 300
		}
	}
	speed := 200.0
	if heli {
		speed = 50
	}
	if isPlayer && g.Start != nil {
		if t := strings.ToLower(g.Start.Type); t != "" && t != "air" {
			b.note("Startart %q nicht unterstuetzt - Luftstart verwendet (Flugplatz im Editor zuweisen)", g.Start.Type)
		}
	}
	gid := b.nextGroup()
	numeric := numericCallsignCountries[country]
	r := b.roleFor(g, s, isPlayer)
	pylons := b.pylonsFor(g, typ, r)
	units := tbl()
	for i := 0; i < g.Units(); i++ {
		ux, uy := offset(x, y, anchor.Head, i, 100, 150)
		skill := "High"
		if isPlayer {
			skill = "Client"
		}
		var callsign any
		if numeric {
			callsign = 100 + gid*10 + i + 1
		} else {
			callsign = tbl(k(1, 1), k(2, 1), k(3, i+1), k("name", fmt.Sprintf("Enfield1%d", i+1)))
		}
		units = append(units, k(i+1, tbl(
			k("alt", altM),
			k("alt_type", "BARO"),
			k("skill", skill),
			k("speed", speed),
			k("type", typ),
			k("unitId", b.nextUnit()),
			k("psi", -rad(anchor.Head)),
			k("x", ux),
			k("name", fmt.Sprintf("%s-%d", g.Name, i+1)),
			k("payload", tbl(
				k("pylons", pylons),
				k("fuel", fuelFor(typ)),
				k("flare", 60),
				k("chaff", 60),
				k("gun", 100),
			)),
			k("y", uy),
			k("heading", rad(anchor.Head)),
			k("callsign", callsign),
		)))
	}

	task := r.dcsTask
	wpTasks := tbl()
	if !isPlayer && len(r.targets) > 0 {
		targets := make([]any, len(r.targets))
		for i, t := range r.targets {
			targets[i] = t
		}
		wpTasks = arr(tbl(
			k("enabled", true), k("key", r.key), k("id", "EngageTargets"), k("number", 1), k("auto", true),
			k("params", tbl(k("targetTypes", arr(targets...)), k("priority", 0))),
		))
	}
	points := arr(airPoint(x, y, altM, speed, wpTasks, true))
	route := g.Route
	if len(route) == 0 && g.Start != nil {
		route = g.Start.Route
	}
	for i, r := range route {
		rx, ry := b.xy(r)
		ra := r.Alt
		if ra <= 0 {
			ra = altM
		}
		points = append(points, k(i+2, airPoint(rx, ry, ra, speed, tbl(), false)))
	}
	freq := 251.0
	if numeric {
		freq = 124
	}
	if heli {
		freq = 127.5
	}
	grp := tbl(
		k("modulation", 0),
		k("tasks", tbl()),
		k("radioSet", false),
		k("task", task),
		k("uncontrolled", false),
		k("route", tbl(k("points", points))),
		k("groupId", gid),
		k("hidden", false),
		k("units", units),
		k("y", y),
		k("x", x),
		k("name", g.Name),
		k("communication", true),
		k("start_time", 0),
		k("frequency", freq),
	)
	s.add(country, cat, grp)
	if len(b.enemyGroupIDs) == 0 && s == b.enemy {
		// air-only enemies still count as targets for "all destroyed"
		b.enemyGroupIDs = append(b.enemyGroupIDs, gid)
	}
	return gid
}

func airPoint(x, y, alt, speed float64, tasks T, first bool) T {
	return tbl(
		k("alt", alt),
		k("action", "Turning Point"),
		k("alt_type", "BARO"),
		k("speed", speed),
		k("task", tbl(k("id", "ComboTask"), k("params", tbl(k("tasks", tasks))))),
		k("type", "Turning Point"),
		k("ETA", 0),
		k("ETA_locked", first),
		k("y", y),
		k("x", x),
		k("speed_locked", true),
		k("formation_template", ""),
	)
}

func (b *builder) groundGroup(g plan.Group, s *side, country string) int {
	b.assigned[country] = s.name
	typ := g.TypeName()
	anchor := g.Anchor()
	x, y := b.xy(anchor)
	ship := g.IsShip()
	cat := "vehicle"
	if ship {
		cat = "ship"
	}
	gid := b.nextGroup()
	units := tbl()
	for i := 0; i < g.Units(); i++ {
		ux, uy := offset(x, y, anchor.Head, i, 40, 0)
		units = append(units, k(i+1, groundUnit(typ, fmt.Sprintf("%s-%d", g.Name, i+1), b.nextUnit(), ux, uy, anchor.Head, ship)))
	}
	speed := 0.0
	if g.Moves() {
		speed = 5.5
		if ship {
			speed = 8
		}
	}
	action := "Off Road"
	if ship {
		action = "Turning Point"
	}
	points := arr(groundPoint(x, y, speed, action))
	if g.Moves() {
		for i, r := range g.Route {
			rx, ry := b.xy(r)
			points = append(points, k(i+2, groundPoint(rx, ry, speed, action)))
		}
	}
	s.add(country, cat, groundGroupTable(g.Name, gid, units, x, y, points))
	return gid
}

// groundUnit builds one vehicle or ship unit table.
func groundUnit(typ, name string, uid int, x, y, headingDeg float64, ship bool) T {
	u := tbl(
		k("skill", "Average"),
		k("type", typ),
		k("unitId", uid),
		k("y", y),
		k("x", x),
		k("name", name),
		k("heading", rad(math.Mod(headingDeg+360, 360))),
	)
	if ship {
		u = append(u, k("transportable", tbl(k("randomTransportable", false))), k("modulation", 0), k("frequency", 127500))
	} else {
		u = append(u, k("coldAtStart", false), k("playerCanDrive", false))
	}
	return u
}

// groundGroupTable wraps units into a vehicle/ship group.
func groundGroupTable(name string, gid int, units T, x, y float64, points T) T {
	return tbl(
		k("visible", false),
		k("lateActivation", false),
		k("tasks", tbl()),
		k("uncontrollable", false),
		k("task", "Ground Nothing"),
		k("taskSelected", true),
		k("route", tbl(k("spans", tbl()), k("points", points))),
		k("groupId", gid),
		k("hidden", false),
		k("units", units),
		k("y", y),
		k("x", x),
		k("name", name),
		k("start_time", 0),
	)
}

func groundPoint(x, y, speed float64, action string) T {
	return tbl(
		k("alt", 0),
		k("type", "Turning Point"),
		k("ETA", 0),
		k("alt_type", "BARO"),
		k("formation_template", ""),
		k("y", y),
		k("x", x),
		k("ETA_locked", true),
		k("speed", speed),
		k("action", action),
		k("task", tbl(k("id", "ComboTask"), k("params", tbl(k("tasks", tbl()))))),
		k("speed_locked", true),
	)
}

// offset places unit i behind (back) and beside (side) the anchor, heading-relative.
func offset(x, y, headingDeg float64, i int, back, side float64) (float64, float64) {
	if i == 0 {
		return x, y
	}
	h := rad(headingDeg)
	fx, fy := math.Cos(h), math.Sin(h) // x north, y east
	rx, ry := fy, -fx
	sign := 1.0
	if i%2 == 0 {
		sign = -1
	}
	n := float64((i + 1) / 2)
	return x - fx*back*float64(i) + rx*side*n*sign, y - fy*back*float64(i) + ry*side*n*sign
}

// ---------------------------------------------------------------------------
// triggers & dictionary

func (b *builder) dictKey(kind, text string) string {
	b.dictN++
	key := fmt.Sprintf("DictKey_%s_%d", kind, b.dictN)
	b.dict = append(b.dict, k(key, text))
	return key
}

func (b *builder) addRule(comment string, rules T, expr string, textKey string, seconds int) {
	n := len(b.rules) + 1
	// Quotes are escaped by luaString on output, matching the mission editor.
	lua := fmt.Sprintf(`a_out_text_delay(getValueDictByKey("%s"), %d, false, 0); mission.trig.func[%d]=nil;`, textKey, seconds, n)
	b.actions = append(b.actions, k(n, lua))
	b.conds = append(b.conds, k(n, "return("+expr+" )"))
	b.funcs = append(b.funcs, k(n, fmt.Sprintf("if mission.trig.conditions[%d]() then mission.trig.actions[%d]() end", n, n)))
	b.flags = append(b.flags, k(n, true))
	b.rules = append(b.rules, k(n, tbl(
		k("rules", rules),
		k("comment", comment),
		k("eventlist", ""),
		k("actions", arr(tbl(
			k("seconds", seconds),
			k("start_delay", 0),
			k("KeyDict_text", textKey),
			k("text", textKey),
			k("predicate", "a_out_text_delay"),
			k("clearview", false),
		))),
		k("predicate", "triggerOnce"),
		k("colorItem", "0x000000ff"),
	)))
}

func (b *builder) radio(r plan.Radio) {
	text := r.Text("de")
	if r.TextEn != "" && r.TextDe != "" {
		text = r.Text("de") + "\n" + r.Text("en")
	}
	key := b.dictKey("ActionText", text)
	switch strings.ToLower(strings.TrimSpace(r.Trigger)) {
	case "all_destroyed", "targets_destroyed", "success":
		if len(b.enemyGroupIDs) == 0 {
			b.note("all_destroyed-Funkspruch ohne Gegnergruppen uebersprungen")
			return
		}
		rules := tbl()
		var parts []string
		for i, gid := range b.enemyGroupIDs {
			if i > 0 {
				rules = append(rules, k(len(rules)+1, tbl(k("predicate", "and"))))
			}
			rules = append(rules, k(len(rules)+1, tbl(k("group", gid), k("predicate", "c_group_dead"))))
			parts = append(parts, fmt.Sprintf("c_group_dead(%d)", gid))
		}
		b.addRule("AIRBORNE alle Ziele vernichtet", rules, strings.Join(parts, " and "), key, 20)
	case "check_zone", "zone", "enter_zone":
		zx, zy := b.center[0], b.center[1]
		radius := 5000.0
		if r.Zone != nil {
			if p := r.Zone.Position(); p.Lat != 0 || p.Lon != 0 || p.X != 0 || p.Z != 0 {
				zx, zy = b.xy(p)
			}
			if r.Zone.R > 0 {
				radius = r.Zone.R
			}
		}
		zid := b.nextZone()
		b.zones = append(b.zones, k(len(b.zones)+1, tbl(
			k("radius", radius),
			k("zoneId", zid),
			k("color", arr(1, 1, 1, 0.15)),
			k("properties", tbl()),
			k("hidden", false),
			k("y", zy),
			k("x", zx),
			k("name", fmt.Sprintf("AB_ZONE_%d", zid)),
			k("type", 0),
			k("heading", 0),
		)))
		rules := arr(tbl(k("group", b.playerGroupID), k("predicate", "c_part_of_group_in_zone"), k("zone", zid)))
		b.addRule("AIRBORNE Zone erreicht", rules, fmt.Sprintf("c_part_of_group_in_zone(%d, %d)", b.playerGroupID, zid), key, 20)
	default:
		delay := r.Delay
		if delay <= 0 {
			delay = 10
		}
		rules := arr(tbl(k("predicate", "c_time_after"), k("seconds", delay)))
		b.addRule(fmt.Sprintf("AIRBORNE Funk nach %d s", delay), rules, fmt.Sprintf("c_time_after(%d)", delay), key, 20)
	}
}

// ---------------------------------------------------------------------------
// static tables

func (b *builder) weather(month int) T {
	we := b.mp.Weather
	base := float64(we.CloudLevel)
	if base < 300 {
		base = 1500
	}
	if base > 5000 {
		base = 5000
	}
	thick := float64(we.CloudHeight - we.CloudLevel)
	if thick < 200 {
		thick = 200
	}
	if thick > 2000 {
		thick = 2000
	}
	density := 0.0
	cfg := strings.ToLower(we.CloudConfig)
	switch {
	case strings.Contains(cfg, "overcast") || strings.Contains(cfg, "heavy"):
		density = 9
	case strings.Contains(cfg, "medium"):
		density = 6
	case strings.Contains(cfg, "light") || strings.Contains(cfg, "few"):
		density = 3
	}
	prec := 0
	if we.PrecLevel > 0 {
		prec = 1
		if density < 6 {
			density = 7
		}
	}
	layers := we.WindLayers
	wind := func(altM int) T {
		bestDir, bestSpd, bestDiff := 180, 3, math.MaxInt
		for _, l := range layers {
			d := l[0] - altM
			if d < 0 {
				d = -d
			}
			if d < bestDiff {
				bestDiff, bestDir, bestSpd = d, l[1], l[2]
			}
		}
		return tbl(k("speed", float64(bestSpd)), k("dir", bestDir))
	}
	turb := we.Turbulence * 6
	if turb < 0 {
		turb = 0
	}
	temps := []int{4, 6, 10, 16, 21, 26, 30, 30, 26, 20, 12, 6}
	temp := 20
	if month >= 1 && month <= 12 {
		temp = temps[month-1]
	}
	return tbl(
		k("wind", tbl(k("at8000", wind(8000)), k("atGround", wind(0)), k("at2000", wind(2000)))),
		k("enable_fog", false),
		k("season", tbl(k("temperature", temp))),
		k("qnh", 760),
		k("cyclones", tbl()),
		k("dust_density", 0),
		k("enable_dust", false),
		k("clouds", tbl(k("thickness", thick), k("density", density), k("base", base), k("iprecptns", prec))),
		k("atmosphere_type", 0),
		k("groundTurbulence", turb),
		k("halo", tbl(k("preset", "auto"))),
		k("type_weather", 0),
		k("modifiedTime", false),
		k("name", "AIrborne"),
		k("fog", tbl(k("visibility", 0), k("thickness", 0))),
		k("visibility", tbl(k("distance", 80000))),
	)
}

func groundControl() T {
	role := func() T { return tbl(k("neutrals", 0), k("blue", 0), k("red", 0)) }
	return tbl(
		k("passwords", tbl(
			k("artillery_commander", tbl()), k("instructor", tbl()), k("observer", tbl()), k("forward_observer", tbl()),
		)),
		k("roles", tbl(
			k("artillery_commander", role()), k("instructor", role()), k("observer", role()), k("forward_observer", role()),
		)),
		k("isPilotControlVehicles", false),
	)
}

func optionsTable() T {
	return tbl(
		k("miscellaneous", tbl(
			k("chat_window_at_start", true), k("TrackIR_external_views", true), k("f11_free_camera", true),
			k("f10_awacs", true), k("Coordinate_Display", "Lat Long Decimal"), k("accidental_failures", false),
			k("autologin", true), k("show_pilot_body", false), k("collect_stat", true), k("synchronize_controls", false),
			k("backup", false), k("headmove", false), k("f5_nearest_ac", true), k("F2_view_effects", 1),
			k("launcher", true), k("allow_server_screenshots", true), k("mgrs_grid_visible", false),
			k("backupTime", 30), k("geo_grid_visible", false), k("force_feedback_enabled", false),
		)),
		k("difficulty", tbl(
			k("fuel", false), k("spottingDot", 3), k("miniHUD", false), k("optionsView", "optview_all"),
			k("setGlobal", true), k("avionicsLanguage", "english"), k("cockpitVisualRM", false), k("map", true),
			k("gWarmUp", false), k("spectatorExternalViews", true), k("userSnapView", true), k("iconsTheme", "nato"),
			k("weapons", false), k("padlock", true), k("birds", 0), k("permitCrash", true), k("immortal", false),
			k("easyCommunication", true), k("wakeTurbulence", true), k("easyFlight", false), k("hideStick", false),
			k("radio", false), k("geffect", "realistic"), k("unrestrictedSATNAV", false), k("reports", true),
			k("tips", true), k("cockpitStatusBarAllowed", false), k("autoTrimmer", false), k("externalViews", true),
			k("RBDAI", true), k("controlsIndicator", true), k("units", "metric"), k("userMarks", true), k("labels", 0),
		)),
	)
}

// ---------------------------------------------------------------------------
// helpers

func parseDate(s string) (day, month, year int) {
	s = strings.TrimSpace(s)
	if n, _ := fmt.Sscanf(s, "%d-%d-%d", &year, &month, &day); n == 3 {
		return
	}
	if n, _ := fmt.Sscanf(s, "%d.%d.%d", &day, &month, &year); n == 3 {
		return
	}
	return 1, 6, 2011
}

func parseTime(s string) (h, m, sec int) {
	s = strings.TrimSpace(s)
	if n, _ := fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec); n >= 2 {
		return
	}
	return 12, 0, 0
}

// roleFor resolves the plan's task for an air group. Without a task the old
// defaults apply: enemy aircraft fly CAP, everything else CAS.
func (b *builder) roleFor(g plan.Group, s *side, isPlayer bool) role {
	if r, ok := lookupRole(g.Task); ok {
		return r
	}
	if strings.TrimSpace(g.Task) != "" {
		b.note("Gruppe %s: Rolle %q unbekannt - Standardrolle verwendet", g.Name, g.Task)
	}
	def := "CAS"
	if !isPlayer && s == b.enemy {
		def = "CAP"
	}
	r, _ := lookupRole(def)
	return r
}

// pylonsFor picks the mission-editor preset for the group and reports the
// choice; empty pylons when no presets are available for the type.
func (b *builder) pylonsFor(g plan.Group, typ string, r role) T {
	if b.payloads == nil || b.payloads.Files == 0 {
		if !b.payloadWarned {
			b.note("Keine DCS-Bewaffnungs-Presets gefunden (DCS_ROOT in .env pruefen) - Pylons bleiben leer")
			b.payloadWarned = true
		}
		return tbl()
	}
	p, ok := b.payloads.Pick(typ, r, g.Payload)
	if !ok {
		b.note("Gruppe %s: keine Presets fuer Typ %q - Bewaffnung im Editor setzen", g.Name, typ)
		return tbl()
	}
	b.note("Bewaffnung %s (%s, %s): %s", g.Name, typ, r.key, p.Name)
	return pylonTable(p)
}
