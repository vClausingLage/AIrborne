package dcs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"airborne/internal/plan"
)

// Scripts holds the mission scripting libraries the generator may embed into
// a .miz. Behaviors that need MOOSE are skipped (with a note) when it is nil.
type Scripts struct {
	Moose     []byte
	MooseFile string // file name inside the .miz, e.g. "Moose_.lua"
}

// MooseRelPath is where the pinned MOOSE build lives below the repo root
// (see scripts/moose/VERSION for the release it was taken from).
var MooseRelPath = filepath.Join("scripts", "moose", "Moose_.lua")

// LoadScripts reads the pinned MOOSE build from <root>/scripts/moose. It
// returns nil when the file is missing, which disables MOOSE behaviors.
func LoadScripts(root string) *Scripts {
	data, err := os.ReadFile(filepath.Join(root, MooseRelPath))
	if err != nil || len(data) == 0 {
		return nil
	}
	return &Scripts{Moose: data, MooseFile: filepath.Base(MooseRelPath)}
}

// BehaviorScriptFile is the generated per-mission script inside the .miz.
const BehaviorScriptFile = "AIrborne_behaviors.lua"

// groupMod changes how a plan group is written so a behavior can drive it.
type groupMod struct {
	name string // name in the .miz (MOOSE finds groups by name)
	late bool   // late activation: the group is a template or waits for a trigger
}

// sideKey qualifies a plan group name with its side, matching Behavior.Side.
func (b *builder) sideKey(s *side, name string) string {
	if s == b.enemy {
		return "enemy:" + name
	}
	return "friendly:" + name
}

func behaviorSide(bh plan.Behavior) string {
	if bh.IsFriendly() {
		return "friendly"
	}
	return "enemy"
}

// prepareBehaviors runs before the groups are written: it decides which groups
// are renamed (MANTIS matches by name prefix) or become late-activated
// templates (SPAWN) / scramble groups (activated by a trigger).
func (b *builder) prepareBehaviors() {
	b.mods = map[string]groupMod{}
	for i, bh := range b.mp.Behaviors {
		side := behaviorSide(bh)
		switch strings.ToLower(strings.TrimSpace(bh.Type)) {
		case plan.BehaviorIADS:
			if !b.needMoose(i, bh) {
				continue
			}
			for _, n := range bh.Groups {
				b.mods[side+":"+n] = groupMod{name: iadsPrefix(i, "SAM") + " " + n}
			}
			for _, n := range bh.Sensors {
				b.mods[side+":"+n] = groupMod{name: iadsPrefix(i, "EWR") + " " + n}
			}
		case plan.BehaviorRespawn:
			if !b.needMoose(i, bh) {
				continue
			}
			for _, n := range bh.Groups {
				b.mods[side+":"+n] = groupMod{name: n, late: true}
			}
		case plan.BehaviorScramble:
			for _, n := range bh.Groups {
				b.mods[side+":"+n] = groupMod{name: n, late: true}
			}
		}
	}
}

func (b *builder) needMoose(i int, bh plan.Behavior) bool {
	if b.scripts != nil {
		return true
	}
	b.note("behaviors[%d] (%s) uebersprungen: MOOSE fehlt (%s)", "behaviors[%d] (%s) skipped: MOOSE missing (%s)", i, bh.Type, MooseRelPath)
	return false
}

// iadsPrefix is the MANTIS name prefix for network i. Only letters, digits and
// underscores: MOOSE matches prefixes as Lua patterns.
func iadsPrefix(i int, kind string) string {
	return fmt.Sprintf("AB_IADS%d_%s", i+1, kind)
}

// applyMod renames the group and returns whether it is late-activated.
func (b *builder) applyMod(s *side, g *plan.Group) groupMod {
	key := b.sideKey(s, g.Name)
	mod, ok := b.mods[key]
	if ok && mod.name != "" {
		g.Name = mod.name
	}
	return mod
}

// emitBehaviors writes the behaviors after all groups exist: native triggers
// for scrambles, one generated MOOSE script for the rest.
func (b *builder) emitBehaviors() {
	var lua []string
	for i, bh := range b.mp.Behaviors {
		side := behaviorSide(bh)
		coal := b.enemy.name
		if side == "friendly" {
			coal = b.player.name
		}
		switch strings.ToLower(strings.TrimSpace(bh.Type)) {
		case plan.BehaviorIADS:
			if b.scripts == nil {
				continue
			}
			lua = append(lua, fmt.Sprintf(
				"-- behaviors[%d]: iads (%d SAM, %d EWR)\nlocal iads = MANTIS:New(%s, %s, %s, nil, %s, false)\niads:Start()",
				i, len(bh.Groups), len(bh.Sensors), luaString(fmt.Sprintf("AB_IADS%d", i+1)),
				luaString(iadsPrefix(i, "SAM")), luaString(iadsPrefix(i, "EWR")), luaString(coal)))
			b.note("behaviors[%d]: Luftverteidigungsnetz (MANTIS) mit %d SAM- und %d Radargruppen", "behaviors[%d]: air defence network (MANTIS) with %d SAM and %d radar groups", i, len(bh.Groups), len(bh.Sensors))
		case plan.BehaviorRespawn:
			if b.scripts == nil {
				continue
			}
			for _, n := range bh.Groups {
				units := b.groupUnits[side+":"+n]
				if units == 0 {
					continue // group missing; Validate reports it
				}
				// InitLimit(units alive, groups in total): one copy alive at a
				// time, the first spawn plus Waves replacements.
				lua = append(lua, fmt.Sprintf(
					"-- behaviors[%d]: respawn %s\nSPAWN:New(%s):InitLimit(%d, %d):SpawnScheduled(%d, 0.3)",
					i, n, luaString(n), units, bh.RespawnWaves()+1, bh.RespawnInterval()))
			}
			b.note("behaviors[%d]: %d Gruppe(n) kehren bis zu %dx zurueck (Pruefung alle %d s)", "behaviors[%d]: %d group(s) return up to %dx (checked every %d s)", i, len(bh.Groups), bh.RespawnWaves(), bh.RespawnInterval())
		case plan.BehaviorScramble:
			b.scramble(i, bh, side)
		}
	}
	if len(lua) > 0 {
		b.loadScripts(lua)
	}
}

// scramble activates late-activated air groups once part of the player group
// enters the behavior's zone (native trigger, no MOOSE needed).
func (b *builder) scramble(i int, bh plan.Behavior, side string) {
	if bh.Zone == nil {
		return // Validate reports it
	}
	var gids []int
	for _, n := range bh.Groups {
		if gid, ok := b.groupIDs[side+":"+n]; ok {
			gids = append(gids, gid)
		}
	}
	if len(gids) == 0 {
		return
	}
	radius := bh.Zone.R
	if radius <= 0 {
		radius = 20000
	}
	zx, zy := b.xy(bh.Zone.Position())
	zid := b.addZone(zx, zy, radius)
	var calls []string
	actions := tbl()
	for _, gid := range gids {
		calls = append(calls, fmt.Sprintf("a_activate_group(%d)", gid))
		actions = append(actions, k(len(actions)+1, tbl(k("group", gid), k("predicate", "a_activate_group"))))
	}
	rules := arr(tbl(k("group", b.playerGroupID), k("predicate", "c_part_of_group_in_zone"), k("zone", zid)))
	b.addTrigger(fmt.Sprintf("AIRBORNE scramble behaviors[%d]", i), false, rules,
		fmt.Sprintf("c_part_of_group_in_zone(%d, %d)", b.playerGroupID, zid), calls, actions)
	b.note("behaviors[%d]: %d Luftgruppe(n) starten, sobald der Spieler die Zone (%.0f km) erreicht", "behaviors[%d]: %d air group(s) launch once the player reaches the zone (%.0f km)", i, len(gids), radius/1000)
}

// loadScripts embeds MOOSE plus the generated behavior script and loads both
// from a MISSION START trigger, MOOSE first.
func (b *builder) loadScripts(lua []string) {
	body := "-- Generated by AIrborne from the mission plan's behaviors. Requires MOOSE (loaded before this file).\n"
	for _, block := range lua {
		// Each behavior runs in its own protected block so one failure does
		// not take the others down; errors land in dcs.log.
		body += "do\n  local ok, err = pcall(function()\n" + indent(block, "    ") + "\n  end)\n" +
			"  if not ok then env.error(\"AIrborne behavior failed: \" .. tostring(err)) end\nend\n"
	}
	mooseKey := b.addResource(b.scripts.MooseFile, b.scripts.Moose)
	behKey := b.addResource(BehaviorScriptFile, []byte(body))
	var calls []string
	actions := tbl()
	for _, key := range []string{mooseKey, behKey} {
		calls = append(calls, fmt.Sprintf("a_do_script_file(getValueResourceByKey(%s))", luaString(key)))
		actions = append(actions, k(len(actions)+1, tbl(k("file", key), k("predicate", "a_do_script_file"))))
	}
	b.addTrigger("AIRBORNE MOOSE + behaviors", true, tbl(), "true", calls, actions)
}

// addResource stores a file in l10n/DEFAULT and registers it in mapResource.
func (b *builder) addResource(name string, data []byte) string {
	b.dictN++
	key := fmt.Sprintf("ResKey_Action_%d", b.dictN)
	b.extra["l10n/DEFAULT/"+name] = data
	b.resources = append(b.resources, k(key, name))
	return key
}

func indent(s, pad string) string {
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}
