package dcs

import (
	"math"
	"strings"
)

// terrain holds the Transverse-Mercator parameters DCS uses per map
// (WGS84, scale 0.9996). DCS x points north (= northing), y east (= easting).
type terrain struct {
	theatre       string
	centralMerid  float64
	falseEasting  float64
	falseNorthing float64
}

var terrains = map[string]terrain{
	"caucasus":       {"Caucasus", 33, -99516.9999999732, -4998114.999999984},
	"syria":          {"Syria", 39, 282801.00000003, -3879865.9999999},
	"persiangulf":    {"PersianGulf", 57, 75755.99999999645, -2894933.0000000377},
	"nevada":         {"Nevada", -117, -193996.80999964548, -4410028.063999966},
	"normandy":       {"Normandy", -3, -195526.00000000204, -5484812.999999951},
	"thechannel":     {"TheChannel", 3, 99376.00000000288, -5636889.00000001},
	"marianaislands": {"MarianaIslands", 147, 238417.99999989968, -1491840.000000048},
	"falklands":      {"Falklands", -57, 147639.99999999712, 5815417.000000032},
	"sinaimap":       {"SinaiMap", 33, 169221.9999999585, -3325312.9999999693},
	"kola":           {"Kola", 21, -62702.00000000191, -7543624.999999931},
}

var terrainAliases = map[string]string{
	"persian gulf": "persiangulf", "gulf": "persiangulf", "pg": "persiangulf",
	"marianas": "marianaislands", "mariana islands": "marianaislands", "mariana": "marianaislands",
	"sinai": "sinaimap", "south atlantic": "falklands", "southatlantic": "falklands",
	"channel": "thechannel", "the channel": "thechannel", "nttr": "nevada",
}

// lookupTerrain resolves a user/LLM supplied map name.
func lookupTerrain(name string) (terrain, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if a, ok := terrainAliases[key]; ok {
		key = a
	}
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, "_", "")
	t, ok := terrains[key]
	return t, ok
}

// toXY converts lat/lon (degrees) into DCS map coordinates (x north, y east).
func (t terrain) toXY(lat, lon float64) (x, y float64) {
	const (
		a  = 6378137.0
		f  = 1 / 298.257223563
		k0 = 0.9996
	)
	e2 := f * (2 - f)
	ep2 := e2 / (1 - e2)
	phi := lat * math.Pi / 180
	dl := (lon - t.centralMerid) * math.Pi / 180
	sin, cos := math.Sin(phi), math.Cos(phi)
	N := a / math.Sqrt(1-e2*sin*sin)
	T := math.Tan(phi) * math.Tan(phi)
	C := ep2 * cos * cos
	A := dl * cos
	e4 := e2 * e2
	e6 := e4 * e2
	M := a * ((1-e2/4-3*e4/64-5*e6/256)*phi -
		(3*e2/8+3*e4/32+45*e6/1024)*math.Sin(2*phi) +
		(15*e4/256+45*e6/1024)*math.Sin(4*phi) -
		(35*e6/3072)*math.Sin(6*phi))
	east := k0 * N * (A + (1-T+C)*A*A*A/6 + (5-18*T+T*T+72*C-58*ep2)*A*A*A*A*A/120)
	north := k0 * (M + N*math.Tan(phi)*(A*A/2+(5-T+9*C+4*C*C)*A*A*A*A/24+(61-58*T+T*T+600*C-330*ep2)*A*A*A*A*A*A/720))
	return north + t.falseNorthing, east + t.falseEasting
}
