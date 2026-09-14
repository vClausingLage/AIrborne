package pipeline

const il2Context = `## Spiel-Kontext: IL-2 Korea
- Karte: "il2" / LANDSCAPE_Korea (Jahreszeiten su/sp/wi/sw), Region Korea, Referenzorte: Kimpo Airfield (105934, 269477), Seoul (~107000, 286000), Incheon (~96000, 253000), Hafeneinfahrt Palmido (~96000, 251500). Koordinaten in METERN (x, z), Hoehen absolut in Metern.
- Fraktionen: 501-503 = Ostblock/Nordkorea (Koalition 1), 601-603 = UN/USA (Koalition 2), 0 = neutral.
- Verfuegbare Flugzeuge (Script-Namen): il10, la11, mig15bis, yak9p (rot); f51d, f80c10, f84e, f86a5 (blau); tu2, li2t, c47b (beide Seiten nutzbar).
- Verfuegbare Bodeneinheiten (Beispiele): willysmb, studebakerus6, gaz67b, m16-mgmc, m3a1-halftrack, m4a3e8, m46, t34-85, su76m, isu122; Flak: boforsl60, dshk-aa, m2cal50-aa, ks19, s60.
- Zeitraum: 1950-1953. Keine Einheiten ausserhalb dieses Zeitraums.`

const dcsContext = `## Spiel-Kontext: DCS World
- Karten (theatre): Syria, Sinai, Caucasus, Persian Gulf, Normandy, Kola, Marianas. Waehle die zur Epoche/Region passende.
- Einheiten: Verwende exakte DCS-Einheiten-"type"-Namen (z. B. "MiG-21Bis", "Mi-24P", "Mi-8MT", "F-16C_50"). Pruefe die Aera-Plausibilitaet (frueher/mittlerer/spaeter Kalter Krieg vs. modern).
- Fraktionen als DCS-Country-Namen (z. B. "USA", "Russia", "Syria", "Israel", "Egypt", "Ukraine"). Weise jede Gruppe einer Country zu.
- Der Spieler muss die Module besitzen; markiere im Plan, welche Module bentigt werden (aircraft = type-Name).
- Positionen: lat/lon auf der gewaehlten Karte; wenn unbekannt, lat/lon = 0 lassen und "notes" mit relativer Beschreibung fuellen (Generator/platzieren spaeter).
- Funk/Briefing-Texte landen spaeter im dictionary (DictKey_*), Sounds als OGG.`

func GameContext(game string) string {
	if game == "dcs" {
		return dcsContext
	}
	return il2Context
}
