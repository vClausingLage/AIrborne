// Package il2 turns a MissionPlan into an IL-2 Korea mission file set
// (NAME.Mission + NAME.ger/.eng/... + NAME.list). The text format follows the
// verified reference mission described in docs/il2-korea-missions.md.
package il2

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"airborne/internal/gen"
	"airborne/internal/plan"
)

const (
	langTitle   = 0
	langDesc    = 1
	langAuthor  = 2
	defaultSky  = `summer\00_Clear_00\sky.ini`
	objCoalIcon = 559
	mapIconID   = 901
)

var cloudConfigRe = regexp.MustCompile(`^(summer|winter)\\[0-9]{2}_[A-Za-z0-9]+_[0-9]{2}\\sky\.ini$`)

type writer struct {
	b     strings.Builder
	next  int
	lang  []plan.Localized
	notes []string
}

func (w *writer) idx() int {
	w.next++
	return w.next
}

func (w *writer) text(l plan.Localized) int {
	w.lang = append(w.lang, l)
	return len(w.lang) - 1
}

func (w *writer) note(format string, args ...any) {
	w.notes = append(w.notes, fmt.Sprintf(format, args...))
}

func (w *writer) line(format string, args ...any) {
	fmt.Fprintf(&w.b, format, args...)
	w.b.WriteString("\r\n")
}

func ints(v []int) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = strconv.Itoa(n)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func asciiName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteRune('\'')
		case r > unicode.MaxASCII:
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// Generate writes the mission files into outDir and returns what was written.
func Generate(mp *plan.MissionPlan, outDir string) (*gen.Result, error) {
	if len(mp.PlayerGroups) == 0 || mp.PlayerGroups[0].Start == nil {
		return nil, fmt.Errorf("Plan hat keine Spielergruppe mit Startposition")
	}
	name := gen.MissionName(mp.Title.Pick("en"))
	w := &writer{}
	w.lang = make([]plan.Localized, 3)
	w.lang[langTitle] = mp.Title
	w.lang[langDesc] = mp.Briefing
	author := mp.Author
	if author == "" {
		author = "Airborne"
	}
	w.lang[langAuthor] = plan.Localized{De: author, En: author}

	w.build(mp)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("Missionsordner nicht erreichbar: %w", err)
	}
	base := filepath.Join(outDir, name)
	files := []string{}
	write := func(path string, data []byte) error {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		files = append(files, path)
		return nil
	}
	if err := write(base+".Mission", []byte(w.b.String())); err != nil {
		return nil, err
	}
	ger := utf16Lines(w.lang, "de")
	eng := utf16Lines(w.lang, "en")
	if err := write(base+".ger", ger); err != nil {
		return nil, err
	}
	for _, ext := range []string{".eng", ".rus", ".fra", ".spa", ".chs"} {
		if err := write(base+ext, eng); err != nil {
			return nil, err
		}
	}
	if err := write(base+".list", []byte{}); err != nil {
		return nil, err
	}
	// Stale binary mirror from a previous editor save would shadow the new text mission.
	if err := os.Remove(base + ".msnbin"); err == nil {
		w.note("Alte %s.msnbin entfernt (wird beim Speichern im Editor neu erzeugt)", name)
	}
	return &gen.Result{
		Game: "il2", Name: name, OutputDir: outDir, MainFile: base + ".Mission",
		Files: files, Notes: gen.NonNil(w.notes),
	}, nil
}

// utf16Lines renders "Index:Text" lines as UTF-16LE with BOM and CRLF.
func utf16Lines(lines []plan.Localized, lang string) []byte {
	var sb strings.Builder
	for i, l := range lines {
		t := strings.ReplaceAll(l.Pick(lang), "\r\n", " ")
		t = strings.ReplaceAll(t, "\n", " ")
		sb.WriteString(strconv.Itoa(i) + ":" + t + "\r\n")
	}
	out := []byte{0xFF, 0xFE}
	for _, r := range sb.String() {
		if r > 0xFFFF {
			r = '?'
		}
		out = append(out, byte(r), byte(r>>8))
	}
	return out
}

// ---------------------------------------------------------------------------
// asset paths

type asset struct {
	script string
	model  string
}

// resolve turns "vehicles/studebakerus6" (or a bare name with a default folder)
// into the Script/Model pair used by IL-2.
func resolve(ref, defaultFolder string) asset {
	s := strings.TrimSpace(ref)
	s = strings.ReplaceAll(s, "\\", "/")
	s = strings.TrimPrefix(s, "LuaScripts/WorldObjects/")
	s = strings.TrimSuffix(s, ".txt")
	folder := defaultFolder
	name := s
	if i := strings.LastIndex(s, "/"); i >= 0 {
		folder = strings.ToLower(s[:i])
		name = s[i+1:]
	}
	name = strings.ToLower(name)
	switch folder {
	case "planes", "plane", "aircraft":
		return asset{
			script: `LuaScripts\WorldObjects\Planes\` + name + `.txt`,
			model:  `graphics\planes\` + name + `\` + name + `.mgm`,
		}
	case "blocks", "block":
		cap := strings.ToUpper(name[:1]) + name[1:]
		return asset{
			script: `LuaScripts\WorldObjects\Blocks\` + cap + `.txt`,
			model:  `graphics\blocks\` + name + `.mgm`,
		}
	default: // vehicles, fixedobjects, ships, ...
		return asset{
			script: `LuaScripts\WorldObjects\` + folder + `\` + name + `.txt`,
			model:  `graphics\` + folder + `\` + name + `\` + name + `.mgm`,
		}
	}
}

// ---------------------------------------------------------------------------
// mission construction

type pos struct{ x, y, z float64 }

func (w *writer) build(mp *plan.MissionPlan) {
	player := mp.PlayerGroups[0]
	playerCountry := country(player.Country, 501)
	playerCoalition := coalition(playerCountry)
	start := player.Start
	startPos := pos{start.X, alt(start.Alt, 1000), start.Z}
	playerAsset := resolve(player.TypeName(), "planes")

	// ---- Options -----------------------------------------------------------
	countries := map[int]int{0: 0, playerCountry: playerCoalition}
	collect := func(c any) {
		if n, ok := plan.CountryInt(c); ok {
			countries[n] = coalition(n)
		}
	}
	for _, g := range mp.EnemyGroups {
		collect(g.Country)
	}
	for _, g := range mp.FriendlyGroups {
		collect(g.Country)
	}
	for _, f := range mp.Flak {
		collect(f.Country)
	}
	for _, s := range mp.Statics {
		collect(s.Country)
	}
	w.options(mp, playerAsset.script, countries)

	// ---- indices that objects must know before they are written ------------
	counterIdx := w.idx()
	targets := destroyableTargets(mp)
	planTargets := 0
	if len(mp.Objectives) > 0 {
		planTargets = mp.Objectives[0].Counter
	}
	counter := targets
	if planTargets > 0 && planTargets < targets {
		counter = planTargets
	}

	// ---- player planes -----------------------------------------------------
	var playerEntities []int
	var leaderEntity int
	startType := 0
	if t := strings.ToLower(start.Type); t == "runway" || t == "parking" || t == "ground" {
		startType = 1
		startPos.y = alt(start.Alt, 0)
		w.note("Bodenstart (StartType=1): Position/Hoehe im Editor am Flugplatz pruefen")
	}
	for i := 0; i < player.Units(); i++ {
		idx := w.idx()
		ent := w.idx()
		p := formationOffset(startPos, start.Heading, i, 120, 180)
		nm := "Leader"
		if i > 0 {
			nm = fmt.Sprintf("Wingman %d", i)
		}
		w.plane(planeSpec{
			name: nm, idx: idx, ent: ent, p: p, heading: start.Heading, a: playerAsset,
			country: playerCountry, ai: 4, coop: 1, formation: i, callsign: 1, callnum: i + 1, startType: startType,
		})
		w.entity(ent, "Plane entity", p, idx, 0)
		playerEntities = append(playerEntities, ent)
		if i == 0 {
			leaderEntity = ent
		}
	}
	route := player.Route
	if len(route) == 0 && start.Route != nil {
		route = start.Route
	}
	if len(route) == 0 {
		route = autoRoute(mp, startPos)
	}
	w.waypoints(route, startPos, []int{leaderEntity}, false, "WP")

	// ---- ground / air groups -----------------------------------------------
	// MCUs that must fire shortly after mission begin (AI routes / attack timers)
	var aiStarts []int
	for _, g := range mp.EnemyGroups {
		aiStarts = append(aiStarts, w.group(g, counterIdx, true)...)
	}
	for _, g := range mp.FriendlyGroups {
		aiStarts = append(aiStarts, w.group(g, 0, false)...)
	}
	for _, f := range mp.Flak {
		w.flak(f)
	}
	for _, s := range mp.Statics {
		w.statics(s)
	}

	// ---- logic -------------------------------------------------------------
	target := targetArea(mp, startPos)
	logicPos := pos{target.x, 5, target.z}

	beginIdx := w.idx()
	var beginTargets []int
	if len(aiStarts) > 0 {
		t := w.idx()
		beginTargets = append(beginTargets, t)
		w.timer(t, "T_AI_START", pos{startPos.x + 500, 5, startPos.z}, 2, aiStarts)
	}

	killTimer := w.idx()
	killTargets := []int{}
	objIdx := w.idx()
	if len(mp.Objectives) > 0 {
		killTargets = append(killTargets, objIdx)
	}

	for _, r := range mp.RadioQueue {
		switch strings.ToLower(strings.TrimSpace(r.Trigger)) {
		case "all_destroyed", "targets_destroyed", "success":
			sub := w.idx()
			w.subtitle(sub, "SUB_ERFOLG", logicPos, w.text(plan.Localized{De: r.Text("de"), En: r.Text("en")}))
			killTargets = append(killTargets, sub)
		case "check_zone", "zone", "enter_zone":
			zp := logicPos
			radius := 2500.0
			if r.Zone != nil {
				if r.Zone.X != 0 || r.Zone.Z != 0 {
					zp = pos{r.Zone.X, 5, r.Zone.Z}
				}
				if r.Zone.R > 0 {
					radius = r.Zone.R
				}
			}
			cz := w.idx()
			sub := w.idx()
			w.checkZone(cz, "CZ_KONTAKT", zp, radius, playerEntities, []int{sub})
			w.subtitle(sub, "SUB_KONTAKT", zp, w.text(plan.Localized{De: r.Text("de"), En: r.Text("en")}))
		default: // mission_begin_delay and anything time based
			delay := r.Delay
			if delay <= 0 {
				delay = 10
			}
			t := w.idx()
			sub := w.idx()
			tp := pos{startPos.x + 1000 + float64(len(beginTargets))*200, 5, startPos.z + 1000}
			w.timer(t, fmt.Sprintf("T_%dS", delay), tp, delay, []int{sub})
			w.subtitle(sub, "SUB_FUNK", tp, w.text(plan.Localized{De: r.Text("de"), En: r.Text("en")}))
			beginTargets = append(beginTargets, t)
		}
	}

	w.missionBegin(beginIdx, pos{startPos.x, 5, startPos.z}, beginTargets)

	if counter > 0 {
		w.counter(counterIdx, "CNT_ALLE_ZIELE", logicPos, counter, []int{killTimer})
		w.timer(killTimer, "T_4S_NACH_LETZTEM_KILL", logicPos, 4, killTargets)
	} else {
		w.note("Keine zerstoerbaren Ziele in enemyGroups - Missionsziel wird nie als erfuellt gemeldet")
	}
	if len(mp.Objectives) > 0 {
		o := mp.Objectives[0]
		title := o.Title
		if title.De == "" && title.En == "" {
			title = mp.Title
		}
		w.objective(objIdx, logicPos, w.text(title), w.text(o.Desc), playerCoalition)
	}
	for i, ic := range mp.Icons {
		from := w.idx()
		to := w.idx()
		label := ic.Label
		if label.De == "" && label.En == "" {
			label = plan.Localized{De: fmt.Sprintf("Ziel %d", i+1), En: fmt.Sprintf("Target %d", i+1)}
		}
		base := plan.Localized{De: "Einsatzbasis", En: "Home base"}
		w.icon(from, pos{ic.From.X, 10, ic.From.Z}, w.text(base), w.text(plan.Localized{De: "Ausgangspunkt", En: "Start point"}), []int{to}, playerCoalition)
		w.icon(to, pos{ic.To.X, 10, ic.To.Z}, w.text(label), w.text(ic.Desc), nil, playerCoalition)
	}
}

// destroyableTargets counts enemy vehicles/ships (planes only when there is
// nothing on the ground), matching the objective counter semantics.
func destroyableTargets(mp *plan.MissionPlan) int {
	ground, air := 0, 0
	for _, g := range mp.EnemyGroups {
		if g.IsAir() {
			air += g.Units()
		} else {
			ground += g.Units()
		}
	}
	if ground > 0 {
		return ground
	}
	return air
}

func targetArea(mp *plan.MissionPlan, fallback pos) pos {
	for _, g := range mp.EnemyGroups {
		if !g.IsAir() {
			a := g.Anchor()
			return pos{a.X, 5, a.Z}
		}
	}
	if len(mp.EnemyGroups) > 0 {
		a := mp.EnemyGroups[0].Anchor()
		return pos{a.X, 5, a.Z}
	}
	return fallback
}

func autoRoute(mp *plan.MissionPlan, start pos) []plan.Position {
	t := targetArea(mp, start)
	return []plan.Position{
		{X: t.x, Z: t.z, Alt: math.Max(500, start.y-300)},
		{X: start.x, Z: start.z, Alt: start.y},
	}
}

func alt(v, def float64) float64 {
	if v <= 0 {
		return def
	}
	return v
}

func country(v any, def int) int {
	if n, ok := plan.CountryInt(v); ok && n > 0 {
		return n
	}
	return def
}

// coalition: 5xx = red (1), 6xx = blue (2), else neutral.
func coalition(c int) int {
	switch {
	case c >= 500 && c < 600:
		return 1
	case c >= 600 && c < 700:
		return 2
	}
	return 0
}

// formationOffset places unit i behind/beside the leader relative to heading.
func formationOffset(p pos, heading float64, i int, back, side float64) pos {
	if i == 0 {
		return p
	}
	h := heading * math.Pi / 180
	fx, fz := math.Cos(h), math.Sin(h)
	rx, rz := -fz, fx
	sideSign := 1.0
	if i%2 == 0 {
		sideSign = -1
	}
	n := float64((i + 1) / 2)
	return pos{
		x: p.x - fx*back*float64(i) + rx*side*n*sideSign,
		y: p.y,
		z: p.z - fz*back*float64(i) + rz*side*n*sideSign,
	}
}

// lineOffset places unit i in a line (40 m spacing) along the heading.
func lineOffset(p pos, heading float64, i int, spacing float64) pos {
	h := heading * math.Pi / 180
	return pos{p.x + math.Cos(h)*spacing*float64(i), p.y, p.z + math.Sin(h)*spacing*float64(i)}
}

// ---------------------------------------------------------------------------
// block writers

func (w *writer) options(mp *plan.MissionPlan, mpPlaneScript string, countries map[int]int) {
	day, month, year := parseDate(mp.Date)
	h, m, s := parseTime(mp.Time)
	season := seasonFor(month)
	we := mp.Weather
	cloudLevel := we.CloudLevel
	if cloudLevel <= 0 {
		cloudLevel = 1500
	}
	cloudHeight := we.CloudHeight
	if cloudHeight <= 0 {
		cloudHeight = 6000
	}
	sky := strings.ReplaceAll(strings.TrimSpace(we.CloudConfig), "/", `\`)
	sky = strings.ReplaceAll(sky, `\\`, `\`)
	if !cloudConfigRe.MatchString(sky) {
		if sky != "" {
			w.note("cloudConfig %q unbekannt - Standard %s verwendet", we.CloudConfig, defaultSky)
		}
		sky = defaultSky
		if season == "wi" {
			sky = `winter\00_Clear_00\sky.ini`
		}
	}
	seaState := we.SeaState
	if seaState < 0 || seaState > 6 {
		seaState = 3
	}
	turb := we.Turbulence
	if turb < 0 || turb > 10 {
		turb = 5
	}
	wind := we.WindLayers
	if len(wind) == 0 {
		wind = [][3]int{{0, 150, 4}, {500, 160, 5}, {1000, 170, 6}, {2000, 180, 8}, {5000, 190, 10}}
	}

	w.line("# Mission File Version = 1.0;")
	w.line("")
	w.line("Options")
	w.line("{")
	w.line("  LCName = %d;", langTitle)
	w.line("  LCDesc = %d;", langDesc)
	w.line("  LCAuthor = %d;", langAuthor)
	w.line(`  PlayerConfig = "";`)
	w.line(`  MultiplayerPlaneConfig = "%s";`, mpPlaneScript)
	w.line("  Time = %d:%d:%d;", h, m, s)
	w.line("  Date = %d.%d.%d;", day, month, year)
	w.line(`  HMap = "graphics\LANDSCAPE_Korea_%s\height.hini";`, season)
	w.line(`  Textures = "graphics\LANDSCAPE_Korea_%s\textures.tini";`, season)
	w.line(`  Forests = "graphics\LANDSCAPE_Korea_%s\trees\woods.wds";`, season)
	w.line(`  Layers = "";`)
	w.line(`  GuiMap = "landscape_korea_su";`)
	w.line(`  SeasonPrefix = "%s";`, season)
	w.line("  MissionType = 1;")
	w.line("  AqmId = 0;")
	w.line("  CloudLevel = %d;", cloudLevel)
	w.line("  CloudHeight = %d;", cloudHeight)
	w.line("  PrecLevel = %d;", clampInt(we.PrecLevel, 0, 100))
	w.line("  PrecType = 0;")
	w.line(`  CloudConfig = "%s";`, sky)
	w.line("  SeaState = %d;", seaState)
	w.line("  Turbulence = %d;", turb)
	w.line("  TempPressLevel = 0;")
	w.line("  Temperature = %d;", temperatureFor(month))
	w.line("  Pressure = 760;")
	w.line("  Haze = 0.4;")
	w.line("  LayerFog = 0;")
	w.line("  CloudsShift = 0.2;")
	w.line("  WindLayers")
	w.line("  {")
	for _, l := range wind {
		w.line("    %d :     %d :     %d;", l[0], l[1], l[2])
	}
	w.line("  }")
	w.line("  Countries")
	w.line("  {")
	keys := make([]int, 0, len(countries))
	for k := range countries {
		keys = append(keys, k)
	}
	sortInts(keys)
	for _, k := range keys {
		w.line("    %d : %d;", k, countries[k])
	}
	w.line("  }")
	w.line("}")
	w.line("")
}

type planeSpec struct {
	name      string
	idx, ent  int
	p         pos
	heading   float64
	a         asset
	country   int
	ai        int
	coop      int
	formation int
	callsign  int
	callnum   int
	startType int
}

func (w *writer) plane(s planeSpec) {
	w.line("Plane")
	w.line("{")
	w.line(`  Name = "%s";`, asciiName(s.name))
	w.line("  Index = %d;", s.idx)
	w.line("  LinkTrId = %d;", s.ent)
	w.position(s.p, s.heading)
	w.line(`  Script = "%s";`, s.a.script)
	w.line(`  Model = "%s";`, s.a.model)
	w.line("  Country = %d;", s.country)
	w.line(`  Desc = "";`)
	w.line(`  Skin = "";`)
	w.line(`  BotSkin = "";`)
	w.line("  AILevel = %d;", s.ai)
	w.line("  CoopStart = %d;", s.coop)
	w.line("  NumberInFormation = %d;", s.formation)
	w.line("  Vulnerable = 1;")
	w.line("  Engageable = 1;")
	w.line("  LimitAmmo = 1;")
	w.line("  StartType = %d;", s.startType)
	w.line("  Callsign = %d;", s.callsign)
	w.line("  Callnum = %d;", s.callnum)
	w.line("  DamageReport = 50;")
	w.line("  DamageThreshold = 1;")
	w.line("  PayloadId = 0;")
	w.line("  ModMask = 1;")
	w.line("  AiRTBDecision = 0;")
	w.line("  DeleteAfterDeath = 1;")
	w.line("  DeleteAfterLand = 1;")
	w.line("  Spotter = -1;")
	w.line("  Fuel = 1;")
	w.line(`  TCode = "";`)
	w.line(`  TCodeColor = "";`)
	w.line("  GunLoad = [];")
	w.line("  GunBelt = [];")
	w.line("  VictoryCount = 0;")
	w.line("  Emblem = 0;")
	w.line("}")
	w.line("")
}

func (w *writer) position(p pos, heading float64) {
	w.line("  XPos = %.3f;", p.x)
	w.line("  YPos = %.3f;", p.y)
	w.line("  ZPos = %.3f;", p.z)
	w.line("  XOri = 0;")
	w.line("  YOri = %d;", int(math.Round(math.Mod(heading+360, 360))))
	w.line("  ZOri = 0;")
}

// entity writes the MCU_TR_Entity paired with an object. onDestroyed > 0 adds
// an OnEvent Type 13 (object destroyed) targeting that MCU.
func (w *writer) entity(idx int, name string, p pos, misObjID int, onDestroyed int) {
	w.line("MCU_TR_Entity")
	w.line("{")
	w.line("  Index = %d;", idx)
	w.line(`  Name = "%s";`, asciiName(name))
	w.line(`  Desc = "";`)
	w.line("  Targets = [];")
	w.line("  Objects = [];")
	w.position(pos{p.x, p.y + 0.2, p.z}, 0)
	w.line("  Enabled = 1;")
	w.line("  MisObjID = %d;", misObjID)
	if onDestroyed > 0 {
		w.line("  OnEvents")
		w.line("  {")
		w.line("    OnEvent")
		w.line("    {")
		w.line("      Type = 13;")
		w.line("      TarId = %d;", onDestroyed)
		w.line("    }")
		w.line("  }")
	}
	w.line("}")
	w.line("")
}

type vehicleSpec struct {
	name      string
	idx, ent  int
	p         pos
	heading   float64
	a         asset
	country   int
	formation int
	engage    bool
	ship      bool
}

func (w *writer) vehicle(s vehicleSpec) {
	kind := "Vehicle"
	if s.ship {
		kind = "Ship"
	}
	w.line("%s", kind)
	w.line("{")
	w.line(`  Name = "%s";`, asciiName(s.name))
	w.line("  Index = %d;", s.idx)
	w.line("  LinkTrId = %d;", s.ent)
	w.position(s.p, s.heading)
	w.line(`  Script = "%s";`, s.a.script)
	w.line(`  Model = "%s";`, s.a.model)
	w.line(`  Desc = "";`)
	w.line("  Country = %d;", s.country)
	w.line("  NumberInFormation = %d;", s.formation)
	w.line("  Vulnerable = 1;")
	w.line("  Engageable = %d;", boolInt(s.engage))
	w.line("  LimitAmmo = 1;")
	w.line("  AILevel = 2;")
	w.line("  DamageReport = 50;")
	w.line("  DamageThreshold = 1;")
	w.line("  DeleteAfterDeath = 1;")
	w.line("  CoopStart = 0;")
	w.line("  Spotter = -1;")
	w.line("  BeaconChannel = 0;")
	w.line("  Callsign = 0;")
	w.line("  PayloadId = 0;")
	w.line("  ModMask = 1;")
	w.line("  Fuel = 1;")
	w.line("  Callnum = 0;")
	w.line(`  Skin = "";`)
	w.line(`  BotSkin = "";`)
	w.line("  RepairTimeMultiplier = 0;")
	w.line("  RehealTimeMultiplier = 0;")
	w.line("  RearmTimeMultiplier = 0;")
	w.line("  RefuelTimeMultiplier = 0;")
	w.line("  MaintenanceRadius = 10;")
	w.line(`  TCode = "";`)
	w.line(`  TCodeColor = "";`)
	if !s.ship {
		w.line("  TrailerAtStart = 1;")
		w.line("  PinToTerrain = 1;")
	}
	w.line("}")
	w.line("")
}

func (w *writer) block(name string, idx int, p pos, heading float64, a asset, country int) {
	w.line("Block")
	w.line("{")
	w.line(`  Name = "%s";`, asciiName(name))
	w.line("  Index = %d;", idx)
	w.line("  LinkTrId = 0;")
	w.position(p, heading)
	w.line(`  Model = "%s";`, a.model)
	w.line(`  Script = "%s";`, a.script)
	w.line("  Country = %d;", country)
	w.line(`  Desc = "";`)
	w.line("  DamageReport = 50;")
	w.line("  DamageThreshold = 1;")
	w.line("  DeleteAfterDeath = 1;")
	w.line("  PinToTerrain = 1;")
	w.line("  Flags = 0;")
	w.line("}")
	w.line("")
}

func (w *writer) mcuHead(kind string, idx int, name string, p pos, targets, objects []int) {
	w.line("%s", kind)
	w.line("{")
	w.line("  Index = %d;", idx)
	if name != "" {
		w.line(`  Name = "%s";`, asciiName(name))
		w.line(`  Desc = "";`)
	}
	w.line("  Targets = %s;", ints(targets))
	w.line("  Objects = %s;", ints(objects))
	w.position(p, 0)
}

func (w *writer) waypoint(idx int, name string, p pos, area, speed int, targets, objects []int) {
	w.mcuHead("MCU_Waypoint", idx, name, p, targets, objects)
	w.line("  Area = %d;", area)
	w.line("  Speed = %d;", speed)
	w.line("  Priority = 1;")
	w.line("}")
	w.line("")
}

// waypoints writes a chain of waypoints for the given entities and returns the
// index of the first one. Ground routes loop back to the first waypoint.
func (w *writer) waypoints(route []plan.Position, ref pos, objects []int, ground bool, prefix string) int {
	if len(route) == 0 {
		return 0
	}
	idxs := make([]int, len(route))
	for i := range route {
		idxs[i] = w.idx()
	}
	for i, r := range route {
		var targets []int
		if i+1 < len(route) {
			targets = []int{idxs[i+1]}
		} else if ground && len(route) > 1 {
			targets = []int{idxs[0]}
		}
		area, speed := 2000, 320
		y := alt(r.Alt, ref.y)
		if ground {
			area, speed = 30, 30
			y = 100
		} else if i+1 < len(route) && i > 0 {
			area, speed = 1200, 260
		}
		w.waypoint(idxs[i], fmt.Sprintf("%s_%d", prefix, i+1), pos{r.X, y, r.Z}, area, speed, targets, objects)
	}
	return idxs[0]
}

func (w *writer) timer(idx int, name string, p pos, seconds int, targets []int) {
	w.mcuHead("MCU_Timer", idx, name, p, targets, nil)
	w.line("  Time = %d;", seconds)
	w.line("  Random = 100;")
	w.line("}")
	w.line("")
}

func (w *writer) counter(idx int, name string, p pos, n int, targets []int) {
	w.mcuHead("MCU_Counter", idx, name, p, targets, nil)
	w.line("  Counter = %d;", n)
	w.line("  Dropcount = 0;")
	w.line("}")
	w.line("")
}

func (w *writer) checkZone(idx int, name string, p pos, radius float64, objects, targets []int) {
	w.mcuHead("MCU_CheckZone", idx, name, p, targets, objects)
	w.line("  Zone = %d;", int(radius))
	w.line("  Cylinder = 1;")
	w.line("  Closer = 1;")
	w.line("}")
	w.line("")
}

func (w *writer) missionBegin(idx int, p pos, targets []int) {
	w.mcuHead("MCU_TR_MissionBegin", idx, "Mission Begin", p, targets, nil)
	w.line("  Enabled = 1;")
	w.line("}")
	w.line("")
}

func (w *writer) subtitle(idx int, name string, p pos, textIdx int) {
	w.mcuHead("MCU_TR_Subtitle", idx, name, p, nil, nil)
	w.line("  Enabled = 1;")
	w.line("  SubtitleInfo")
	w.line("  {")
	w.line("    Duration = 8;")
	w.line("    FontSize = 20;")
	w.line("    HAlign = 1;")
	w.line("    VAlign = 0;")
	w.line("    RColor = 255;")
	w.line("    GColor = 255;")
	w.line("    BColor = 255;")
	w.line("    LCText = %d;", textIdx)
	w.line("  }")
	w.line("  ")
	w.line("  Coalitions = [0, 1, 2];")
	w.line("}")
	w.line("")
}

func (w *writer) objective(idx int, p pos, nameIdx, descIdx, coalition int) {
	w.mcuHead("MCU_TR_MissionObjective", idx, "", p, nil, nil)
	w.line("  Enabled = 1;")
	w.line("  LCName = %d;", nameIdx)
	w.line("  LCDesc = %d;", descIdx)
	w.line("  TaskType = 0;")
	w.line("  Coalition = %d;", coalition)
	w.line("  Success = 1;")
	w.line("  IconType = %d;", objCoalIcon)
	w.line("}")
	w.line("")
}

func (w *writer) icon(idx int, p pos, nameIdx, descIdx int, targets []int, coalition int) {
	w.mcuHead("MCU_Icon", idx, "", p, targets, nil)
	w.line("  Enabled = 1;")
	w.line("  LCName = %d;", nameIdx)
	w.line("  LCDesc = %d;", descIdx)
	w.line("  IconId = %d;", mapIconID)
	w.line("  RColor = 255;")
	w.line("  GColor = 255;")
	w.line("  BColor = 255;")
	w.line("  LineType = 14;")
	w.line("  Coalitions = [%d];", coalition)
	w.line("}")
	w.line("")
}

func (w *writer) attackArea(idx int, p pos, objects []int, air bool) {
	w.mcuHead("MCU_CMD_AttackArea", idx, "CMD_ATTACK", p, nil, objects)
	w.line("  AttackGround = %d;", boolInt(!air))
	w.line("  AttackAir = %d;", boolInt(air))
	w.line("  AttackGTargets = %d;", boolInt(!air))
	w.line("  AttackArea = 5000;")
	w.line("  Time = 600;")
	w.line("  Priority = 1;")
	w.line("}")
	w.line("")
}

// group writes an enemy/friendly group. Vehicles get destroyed-events towards
// counterIdx (0 = none). Returns MCU indices that must fire at mission begin
// (first waypoints of moving groups, attack timers of AI planes).
func (w *writer) group(g plan.Group, counterIdx int, enemy bool) []int {
	anchor := g.Anchor()
	c := country(g.Country, 601)
	if g.IsAir() {
		return w.aiPlanes(g, anchor, c, enemy)
	}
	a := resolve(g.TypeName(), "vehicles")
	base := pos{anchor.X, alt(anchor.Alt, 100), anchor.Z}
	var leaderEnt int
	for i := 0; i < g.Units(); i++ {
		idx := w.idx()
		ent := w.idx()
		p := lineOffset(base, anchor.Head, i, 40)
		nm := g.Name
		if g.Units() > 1 {
			nm = fmt.Sprintf("%s %d", g.Name, i+1)
		}
		w.vehicle(vehicleSpec{name: nm, idx: idx, ent: ent, p: p, heading: anchor.Head, a: a, country: c, formation: i, engage: true, ship: g.IsShip()})
		w.entity(ent, nm+" entity", p, idx, counterIdx)
		if i == 0 {
			leaderEnt = ent
		}
	}
	if g.Moves() {
		first := w.waypoints(g.Route, base, []int{leaderEnt}, true, "GWP_"+asciiName(g.Name))
		return []int{first}
	}
	return nil
}

func (w *writer) aiPlanes(g plan.Group, anchor plan.Position, c int, enemy bool) []int {
	a := resolve(g.TypeName(), "planes")
	base := pos{anchor.X, alt(anchor.Alt, 1500), anchor.Z}
	var ents []int
	for i := 0; i < g.Units(); i++ {
		idx := w.idx()
		ent := w.idx()
		p := formationOffset(base, anchor.Head, i, 120, 180)
		nm := fmt.Sprintf("%s %d", g.Name, i+1)
		w.plane(planeSpec{name: nm, idx: idx, ent: ent, p: p, heading: anchor.Head, a: a, country: c, ai: 3, coop: 0, formation: i, callsign: 2, callnum: i + 1})
		w.entity(ent, nm+" entity", p, idx, 0)
		ents = append(ents, ent)
	}
	route := g.Route
	if len(route) == 0 {
		route = []plan.Position{{X: anchor.X, Z: anchor.Z, Alt: base.y}}
	}
	first := w.waypoints(route, base, ents[:1], false, "AWP_"+asciiName(g.Name))
	if enemy {
		// last waypoint -> attack command; we re-emit the chain end by adding a
		// command the final waypoint targets. Simplest: a timer chained from the
		// first waypoint's activation that issues the attack after 30 s.
		cmd := w.idx()
		t := w.idx()
		last := route[len(route)-1]
		w.timer(t, "T_ATTACK_"+asciiName(g.Name), pos{last.X, 5, last.Z}, 30, []int{cmd})
		w.attackArea(cmd, pos{last.X, 5, last.Z}, ents, true)
		return []int{first, t}
	}
	return []int{first}
}

func (w *writer) flak(f plan.Flak) {
	a := resolve(f.Script, "fixedobjects")
	c := country(f.Country, 601)
	base := pos{f.Position.X, alt(f.Position.Alt, 100), f.Position.Z}
	n := f.Count
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		idx := w.idx()
		ent := w.idx()
		p := lineOffset(base, f.Position.Head+90, i, 60)
		nm := fmt.Sprintf("Flak %s %d", filepath.Base(strings.ReplaceAll(f.Script, "\\", "/")), i+1)
		w.vehicle(vehicleSpec{name: nm, idx: idx, ent: ent, p: p, heading: f.Position.Head, a: a, country: c, engage: true})
		w.entity(ent, nm+" entity", p, idx, 0)
	}
}

func (w *writer) statics(s plan.StaticObject) {
	folder := "blocks"
	if k := strings.ToLower(s.Kind); k == "vehicle" || k == "fixedobject" || k == "ship" {
		folder = k + "s"
	}
	a := resolve(s.Script, folder)
	c := country(s.Country, 0)
	positions := s.Positions
	if len(positions) == 0 {
		w.note("statics %q ohne positions uebersprungen", s.Script)
		return
	}
	n := s.Count
	if n < len(positions) {
		n = len(positions)
	}
	for i := 0; i < n; i++ {
		ref := positions[i%len(positions)]
		p := lineOffset(pos{ref.X, alt(ref.Alt, 100), ref.Z}, ref.Head+90, i/len(positions), 30)
		nm := fmt.Sprintf("%s %d", filepath.Base(strings.ReplaceAll(s.Script, "\\", "/")), i+1)
		if strings.Contains(a.script, `\Blocks\`) {
			w.block(nm, w.idx(), p, ref.Head, a, c)
		} else {
			idx := w.idx()
			ent := w.idx()
			w.vehicle(vehicleSpec{name: nm, idx: idx, ent: ent, p: p, heading: ref.Head, a: a, country: c, engage: true})
			w.entity(ent, nm+" entity", p, idx, 0)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers

func parseDate(s string) (day, month, year int) {
	day, month, year = 14, 9, 1950
	s = strings.TrimSpace(s)
	if n, _ := fmt.Sscanf(s, "%d-%d-%d", &year, &month, &day); n == 3 {
		return
	}
	if n, _ := fmt.Sscanf(s, "%d.%d.%d", &day, &month, &year); n == 3 {
		return
	}
	return 14, 9, 1950
}

func parseTime(s string) (h, m, sec int) {
	s = strings.TrimSpace(s)
	if n, _ := fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec); n >= 2 {
		return
	}
	return 12, 0, 0
}

func seasonFor(month int) string {
	switch month {
	case 12, 1, 2:
		return "wi"
	case 6, 7, 8:
		return "su"
	}
	return "sp"
}

func temperatureFor(month int) int {
	t := []int{-4, -2, 4, 11, 17, 21, 24, 25, 20, 13, 6, -1}
	if month < 1 || month > 12 {
		return 15
	}
	return t[month-1]
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func sortInts(v []int) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
