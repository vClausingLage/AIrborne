package prefab

import (
	"sort"
	"strings"
)

// StaticInfo describes a DCS static object: the "type" the mission file uses,
// its category and the 3D model (shape_name) the mission editor writes.
type StaticInfo struct {
	Type     string
	Category string
	Shape    string
	Desc     string
}

// statics is the curated catalog of stock DCS static objects the prefab
// prompt may use. Aircraft/helicopter/vehicle statics need no shape_name
// (their unit type is the model) and are therefore not listed here.
var statics = []StaticInfo{
	// Fortifications / buildings
	{"Hangar A", "Fortifications", "hangar_a", "Grosser Flugzeughangar"},
	{"Hangar B", "Fortifications", "hangar_b", "Flugzeughangar"},
	{"Shelter", "Fortifications", "ukrytie", "Flugzeug-Shelter (HAS)"},
	{"Shelter B", "Fortifications", "ukrytie_b", "Flugzeug-Shelter, klein"},
	{"Barracks 2", "Fortifications", "kazarma2", "Kaserne"},
	{"Military staff", "Fortifications", "aviashtab", "Stabsgebaeude"},
	{".Command Center", "Fortifications", "ComCenter", "Kommandozentrale (Bunker)"},
	{"Comms tower M", "Fortifications", "tele_bash_m", "Funkmast"},
	{"TV tower", "Fortifications", "tele_bash", "Fernsehturm"},
	{"Repair workshop", "Fortifications", "tech", "Werkstatt"},
	{"Workshop A", "Fortifications", "tec_a", "Werkstatt A"},
	{"Garage A", "Fortifications", "garage_a", "Garage gross"},
	{"Garage B", "Fortifications", "garage_b", "Garage"},
	{"Garage small A", "Fortifications", "garagh-small-a", "Garage klein A"},
	{"Garage small B", "Fortifications", "garagh-small-b", "Garage klein B"},
	{"Cafe", "Fortifications", "stolovaya", "Kantine"},
	{"Shop", "Fortifications", "magazin", "Laden"},
	{"Restaurant 1", "Fortifications", "restaurant1", "Restaurant"},
	{"Small house 1A", "Fortifications", "domik1a", "Kleines Haus"},
	{"Small house 1B", "Fortifications", "domik1b", "Kleines Haus B"},
	{"House 1A", "Fortifications", "home1_a", "Wohnhaus"},
	{"Farm A", "Fortifications", "ferma_a", "Bauernhof A"},
	{"Farm B", "Fortifications", "ferma_b", "Bauernhof B"},
	{"Subsidiary structure 1", "Fortifications", "saray-1", "Schuppen"},
	{"Subsidiary structure 2", "Fortifications", "saray-2", "Schuppen 2"},
	{"Water tower A", "Fortifications", "wodokachka_a", "Wasserturm"},
	{"Boiler-house A", "Fortifications", "kotelnaya_a", "Heizhaus"},
	{"Electric power box", "Fortifications", "tr_budka", "Trafohaeuschen"},
	{"Pump station", "Fortifications", "nasos", "Pumpstation"},
	{"Oil derrick", "Fortifications", "derrick", "Bohrturm"},
	{"Oil platform", "Fortifications", "oil_platform", "Oelplattform (Wasser)"},
	{"Railway station", "Fortifications", "r_station", "Bahnhof"},
	{"Windsock", "Fortifications", "H-Windsock_RW", "Windsack"},
	{"Red_Flag", "Fortifications", "H-Flag_R", "Rote Fahne"},
	{"White_Flag", "Fortifications", "H-Flag_W", "Weisse Fahne"},
	{"Black_Tyre", "Fortifications", "H-tyre_B", "Reifen schwarz (Markierung)"},
	{"White_Tyre", "Fortifications", "H-tyre_W", "Reifen weiss (Markierung)"},
	{"Container white", "Fortifications", "konteiner_white", "Container weiss"},
	{"Container brown", "Fortifications", "konteiner_brown", "Container braun"},
	{"Container red 1", "Fortifications", "konteiner_red1", "Container rot 1"},
	{"Container red 2", "Fortifications", "konteiner_red2", "Container rot 2"},
	{"Container red 3", "Fortifications", "konteiner_red3", "Container rot 3"},
	{"Container 20ft", "Fortifications", "container_20ft", "Seecontainer 20 ft"},
	{"Container 40ft", "Fortifications", "container_40ft", "Seecontainer 40 ft"},
	{"Landmine", "Fortifications", "landmine", "Landmine"},
	// FARP equipment
	{"FARP Tent", "Fortifications", "PalatkaB", "FARP-Zelt"},
	{"FARP Ammo Dump Coating", "Fortifications", "SetkaKP", "FARP-Munitionslager"},
	{"FARP Fuel Depot", "Fortifications", "GSM Rus", "FARP-Tanklager"},
	{"FARP CP Blindage", "Fortifications", "kp_ug", "FARP-Gefechtsstand"},
	// Heliports
	{"FARP", "Heliports", "FARPS", "FARP mit 4 Landeplaetzen"},
	{"Invisible FARP", "Heliports", "invisiblefarp", "Unsichtbarer FARP (Versorgung)"},
	{"SINGLE_HELIPAD", "Heliports", "FARP_SINGLE_01", "Einzelner Hubschrauberlandeplatz"},
	// Warehouses
	{"Warehouse", "Warehouses", "warehouse", "Lagerhalle"},
	{"Tank", "Warehouses", "bak", "Treibstofftank"},
	{".Ammunition depot", "Warehouses", "SkladC", "Munitionsdepot"},
	// Cargos (sling-loadable)
	{"container_cargo", "Cargos", "container_cargo", "Cargo-Container"},
	{"iso_container", "Cargos", "iso_container", "ISO-Container (Cargo)"},
	{"iso_container_small", "Cargos", "iso_container_small", "ISO-Container klein"},
	{"ammo_cargo", "Cargos", "ammo_cargo", "Munitionskisten"},
	{"barrels_cargo", "Cargos", "barrels_cargo", "Faesser"},
	{"fueltank_cargo", "Cargos", "fueltank_cargo", "Tankcontainer"},
	{"oiltank_cargo", "Cargos", "oiltank_cargo", "Oeltank"},
	{"pipes_big_cargo", "Cargos", "pipes_big_cargo", "Rohre gross"},
	{"pipes_small_cargo", "Cargos", "pipes_small_cargo", "Rohre klein"},
	{"trunks_long_cargo", "Cargos", "trunks_long_cargo", "Baumstaemme lang"},
	{"trunks_small_cargo", "Cargos", "trunks_small_cargo", "Baumstaemme kurz"},
	{"tetrapod_cargo", "Cargos", "tetrapod_cargo", "Tetrapoden"},
	{"m117_cargo", "Cargos", "m117_cargo", "M117-Bomben-Palette"},
	{"uh1h_cargo", "Cargos", "uh1h_cargo", "UH-1H-Rumpf (Cargo)"},
}

var staticIndex = func() map[string]StaticInfo {
	m := map[string]StaticInfo{}
	for _, s := range statics {
		m[strings.ToLower(s.Type)] = s
		m[strings.ToLower(s.Shape)] = s
	}
	return m
}()

// LookupStatic resolves a static type (or shape name), case-insensitively.
func LookupStatic(typ string) (StaticInfo, bool) {
	s, ok := staticIndex[strings.ToLower(strings.TrimSpace(typ))]
	return s, ok
}

// CatalogText renders the static catalog for the prefab prompt.
func CatalogText() string {
	byCat := map[string][]StaticInfo{}
	for _, s := range statics {
		byCat[s.Category] = append(byCat[s.Category], s)
	}
	cats := make([]string, 0, len(byCat))
	for c := range byCat {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	var b strings.Builder
	for _, c := range cats {
		b.WriteString("- " + c + ": ")
		parts := make([]string, 0, len(byCat[c]))
		for _, s := range byCat[c] {
			parts = append(parts, `"`+s.Type+`" (`+s.Desc+`)`)
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\n")
	}
	return b.String()
}
