package dcs

import (
	"fmt"
	"math"
	"strings"

	"airborne/internal/plan"
	"airborne/internal/prefab"
)

// PrefabSource resolves prefab ids/names from the plan to their definition.
type PrefabSource interface {
	Find(nameOrID string) (*prefab.Prefab, bool)
}

// placedUnit is one unit of a prefab after rotation into map coordinates.
type placedUnit struct {
	name    string
	typ     string
	x, y    float64
	heading float64 // degrees, absolute
	elem    int     // index into prefab.Elements
	idx     int     // unit index inside the element
}

// asStatic reports whether an element is emitted as a DCS static object;
// statics without a known category fall back to a (stationary) vehicle group.
func asStatic(e prefab.Element) bool {
	return e.IsStatic() && e.Category != ""
}

// placePrefabs expands every plan.Prefabs entry into DCS groups.
func (b *builder) placePrefabs() {
	for i, pp := range b.mp.Prefabs {
		if b.prefabs == nil {
			b.note("Prefab %q: keine Prefab-Bibliothek verfuegbar", pp.Prefab)
			continue
		}
		pf, ok := b.prefabs.Find(pp.Prefab)
		if !ok {
			b.note("Prefab %q nicht gefunden - uebersprungen", pp.Prefab)
			continue
		}
		pfc := *pf
		for _, is := range pfc.Normalize() {
			b.note("Prefab %s: %s", pfc.Name, is)
		}
		b.placePrefab(pfc, pp, i)
	}
}

func (b *builder) placePrefab(pf prefab.Prefab, pp plan.PrefabPlacement, idx int) {
	sd := b.player
	if pp.IsEnemy() {
		sd = b.enemy
	}
	countryVal := pp.Country
	if countryVal == nil || plan.CountryString(countryVal) == "" {
		countryVal = pf.Country
	}
	country := b.sideCountry(countryVal, sd)
	label := strings.TrimSpace(pp.Name)
	if label == "" {
		label = pf.Name
	}
	if idx > 0 {
		label = fmt.Sprintf("%s %d", label, idx+1)
	}

	ox, oy := b.xy(pp.Position)
	h := rad(pp.Position.Head)
	cosH, sinH := math.Cos(h), math.Sin(h)
	world := func(dx, dy float64) (float64, float64) {
		return ox + dx*cosH - dy*sinH, oy + dx*sinH + dy*cosH
	}

	// local positions of every unit (before rotation), keyed by element index
	type local struct{ dx, dy, hdg float64 }
	locals := make([][]local, len(pf.Elements))
	for i, e := range pf.Elements {
		eh := rad(e.Heading)
		fx, fy := math.Cos(eh), math.Sin(eh) // forward in local frame
		rx, ry := -math.Sin(eh), math.Cos(eh)
		spacing := e.Spacing
		if spacing <= 0 {
			spacing = 20
			if e.IsShip() {
				spacing = 400
			}
		}
		for u := 0; u < e.Units(); u++ {
			d := float64(u) * spacing
			dx, dy := e.DX, e.DY
			if strings.ToLower(e.Layout) == "column" {
				dx, dy = dx-fx*d, dy-fy*d
			} else {
				dx, dy = dx+rx*d, dy+ry*d
			}
			locals[i] = append(locals[i], local{dx, dy, e.Heading})
		}
	}

	// pass 1: ships and vehicles (grouped by element.group), remember unit ids
	// so deck statics can be linked to their carrier.
	type groupKey struct{ cat, name string }
	order := []groupKey{}
	groups := map[groupKey][]placedUnit{}
	unitIDs := map[string]int{} // element name -> first unitId
	elemIdx := map[string]int{} // element name -> index
	for i, e := range pf.Elements {
		elemIdx[e.Name] = i
		if asStatic(e) {
			continue
		}
		cat := "vehicle"
		if e.IsShip() {
			cat = "ship"
		}
		gname := e.Group
		if gname == "" {
			gname = e.Name
		}
		key := groupKey{cat, label + " " + gname}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		for u, l := range locals[i] {
			x, y := world(l.dx, l.dy)
			pu := placedUnit{name: fmt.Sprintf("%s %s-%d", label, e.Name, u+1), typ: e.Type, x: x, y: y,
				heading: pp.Position.Head + l.hdg, elem: i, idx: u}
			groups[key] = append(groups[key], pu)
		}
	}
	for _, key := range order {
		units := groups[key]
		gid := b.nextGroup()
		ut := tbl()
		ship := key.cat == "ship"
		for i, u := range units {
			uid := b.nextUnit()
			if u.idx == 0 {
				unitIDs[pf.Elements[u.elem].Name] = uid // link target for deck statics
			}
			ut = append(ut, k(i+1, groundUnit(u.typ, u.name, uid, u.x, u.y, u.heading, ship)))
		}
		action := "Off Road"
		if ship {
			action = "Turning Point"
		}
		points := arr(groundPoint(units[0].x, units[0].y, 0, action))
		sd.add(country, key.cat, groundGroupTable(key.name, gid, ut, units[0].x, units[0].y, points))
	}

	// pass 2: statics (one DCS static group per unit, like the mission editor)
	for i, e := range pf.Elements {
		if !asStatic(e) {
			continue
		}
		var link *linkInfo
		if e.LinkTo != "" {
			if uid, ok := unitIDs[e.LinkTo]; ok {
				si := elemIdx[e.LinkTo]
				sl := locals[si][0]
				link = &linkInfo{unitID: uid, shipDX: sl.dx, shipDY: sl.dy, shipHdg: sl.hdg}
			} else {
				b.note("Prefab %s: linkTo %q nicht aufloesbar - Static frei platziert", pf.Name, e.LinkTo)
			}
		}
		for u, l := range locals[i] {
			x, y := world(l.dx, l.dy)
			name := fmt.Sprintf("%s %s-%d", label, e.Name, u+1)
			var lk *linkInfo
			if link != nil {
				lk = link.rel(l.dx, l.dy, l.hdg)
			}
			b.staticGroup(sd, country, name, e, x, y, pp.Position.Head+l.hdg, lk)
		}
	}
}

// linkInfo attaches a static to a ship unit (parked aircraft on a carrier).
type linkInfo struct {
	unitID                  int
	shipDX, shipDY, shipHdg float64
	offX, offY, angle       float64
}

// rel computes the offsets of a static in the ship's own frame.
func (l linkInfo) rel(dx, dy, hdg float64) *linkInfo {
	sh := rad(l.shipHdg)
	vx, vy := dx-l.shipDX, dy-l.shipDY
	out := l
	out.offX = vx*math.Cos(sh) + vy*math.Sin(sh)  // along the ship's axis (bow positive)
	out.offY = -vx*math.Sin(sh) + vy*math.Cos(sh) // starboard positive
	out.angle = rad(hdg - l.shipHdg)
	return &out
}

func (b *builder) staticGroup(s *side, country, name string, e prefab.Element, x, y, headingDeg float64, link *linkInfo) {
	gid := b.nextGroup()
	uid := b.nextUnit()
	hdg := rad(math.Mod(headingDeg+360, 360))
	unit := tbl(
		k("category", e.Category),
		k("type", e.Type),
		k("unitId", uid),
		k("rate", 100),
		k("y", y),
		k("x", x),
		k("name", name),
		k("heading", hdg),
		k("dead", false),
	)
	if e.ShapeName != "" {
		unit = append(unit, k("shape_name", e.ShapeName))
	}
	if e.Livery != "" {
		unit = append(unit, k("livery_id", e.Livery))
	}
	if e.Category == "Cargos" {
		unit = append(unit, k("canCargo", true), k("mass", 1000))
	}
	if link != nil {
		unit = append(unit,
			k("linkUnit", link.unitID),
			k("linkOffset", true),
			k("offsets", tbl(k("angle", link.angle), k("x", link.offX), k("y", link.offY))),
		)
	}
	grp := tbl(
		k("heading", hdg),
		k("route", tbl(k("points", arr(tbl(
			k("alt", 0), k("type", ""), k("name", ""), k("y", y), k("speed", 0), k("x", x),
			k("formation_template", ""), k("action", ""),
		))))),
		k("groupId", gid),
		k("units", tbl(k(1, unit))),
		k("y", y),
		k("x", x),
		k("name", name),
		k("dead", false),
	)
	s.add(country, "static", grp)
}
