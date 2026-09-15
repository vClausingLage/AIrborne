**Output: NUR das JSON-Objekt, exakt dieses Schema (kein Markdown, keine Erklärung, keine Kommentare im JSON).** Halte dein Nachdenken sehr knapp (max. 10 Sätze Planung) und stelle sicher, dass das JSON vollständig und syntaktisch gültig ist.

```json
{
  "game": "il2",
  "title": {"de": "...", "en": "..."},
  "author": "AIrborne",
  "map": "korea",
  "date": "1950-09-14",
  "time": "15:35:00",
  "weather": {"cloudLevel":1500,"cloudHeight":6000,"precLevel":0,"cloudConfig":"summer\\00_Clear_00\\sky.ini","seaState":3,"turbulence":5,"windLayers":[[0,150,4],[500,160,5],[1000,170,6],[2000,180,8],[5000,190,10]]},
  "playerGroups": [
    {"name":"Sowol","aircraft":"il10","count":2,"country":501,"task":"CAS","payload":"4x FAB-100, 8x RS-82, volle Kanonen","start":{"type":"air","x":86500,"z":244000,"alt":1000,"heading":40},
     "route":[{"x":95000,"z":242000,"alt":700},{"x":99800,"z":261800,"alt":500},{"x":88000,"z":247000,"alt":800}]}
  ],
  "enemyGroups": [
    {"name":"Trucks","kind":"vehicle","script":"vehicles/studebakerus6","count":3,"country":601,"position":{"x":99870,"z":261830,"heading":15},"movement":{"type":"static"}},
    {"name":"Patrol","kind":"vehicle","script":"vehicles/willysmb","count":1,"country":601,"position":{"x":103080,"z":264580,"heading":230},"movement":{"type":"route"},"route":[{"x":103400,"z":264900},{"x":102600,"z":264200}]}
  ],
  "friendlyGroups": [],
  "flak": [
    {"script":"fixedobjects/boforsl60","count":2,"position":{"x":100180,"z":261640},"country":601,"engageable":true}
  ],
  "statics": [
    {"kind":"block","script":"blocks/mil_camonet","count":2,"positions":[{"x":99800,"z":261880,"heading":12},{"x":99560,"z":261860,"heading":192}],"country":601}
  ],
  "objectives": [
    {"title":{"de":"...","en":"..."},"desc":{"de":"...","en":"..."},"counter":4}
  ],
  "radioQueue": [
    {"trigger":"mission_begin_delay","delay":10,"speaker":"Basis","textDe":"...","textEn":"..."},
    {"trigger":"check_zone","zone":{"x":99800,"z":261800,"r":2500},"speaker":"Basis","textDe":"...","textEn":"..."},
    {"trigger":"all_destroyed","speaker":"Basis","textDe":"...","textEn":"..."},
    {"trigger":"mission_begin_delay","delay":1500,"speaker":"Basis","textDe":"Zeitlimit ...","textEn":"Time limit ..."}
  ],
  "briefing": {"de":"...","en":"..."},
  "icons": [{"from":{"x":86500,"z":244000},"to":{"x":99800,"z":261800},"label":{"de":"Zielgebiet","en":"Target area"},"desc":{"de":"...","en":"..."}}]
}
```

**Feldregeln (beide Spiele):**
- `game` = Spiel-Kontext ("il2" oder "dcs"). `date` = YYYY-MM-DD, `time` = HH:MM:SS.
- `playerGroups`: genau 1 Gruppe, `count` = Anzahl Spielerplätze (1-4). `start.type`: "air" (Luftstart, empfohlen) oder "runway". `route` = 2-4 Wegpunkte (Anflug/IP, Zielgebiet über den Gegnerpositionen, Rückweg zum Start-/Landeplatz) – die Route ist die Aufgabe des Spielers und muss zu den Zielen führen. `payload` = Pflicht: kurze Klartext-Bewaffnung (z. B. "2x FAB-250, 2x UB-16 S-5, volle Kanonen"), passend zu Typ, Epoche und Aufgabe.
- `enemyGroups`/`friendlyGroups`: `kind` = "vehicle" | "plane" | "helicopter" | "ship". `count` Einheiten stehen in Reihe ab `position`. Bewegung: `movement.type` "static" oder "route" (dann `route` mit 2-4 Wegpunkten; Fahrzeuge fahren die Schleife ab). Flugzeuge/Hubschrauber (`kind` "plane"/"helicopter") brauchen immer eine `route` (Patrouille, CAP-Orbit oder Anflug auf ihr Ziel) und ein `payload` wie bei der Spielergruppe.
- `objectives`: genau 1 Eintrag. `counter` = Anzahl der zerstörbaren Fahrzeuge in `enemyGroups` (Flak zählt nicht). Titel/Beschreibung DE+EN.
- `radioQueue`: 2-5 Meldungen, je max. 2 Sätze, authentischer Funkton. Trigger: `mission_begin_delay` (+`delay` Sekunden nach Start), `check_zone` (Spieler betritt `zone`), `all_destroyed` (alle Ziele vernichtet).
- `briefing`: DE + EN, je 4-8 Sätze (Lage, Auftrag, Durchführung, Gefahren, Rückkehr).
- `icons`: 0-2 Kartenmarkierungen (Linie von `from` nach `to`).

**IL-2 Korea (`game:"il2"`):** `country` numerisch (501-503 rot/Nordkorea, 601-603 blau/UN). Koordinaten `x`/`z` in Metern auf der Korea-Karte, `alt` in Metern. `aircraft` = Script-Name (il10, la11, mig15bis, yak9p, f51d, f80c10, f84e, f86a5, tu2, li2t, c47b). `script` = "vehicles/<name>", "fixedobjects/<name>", "blocks/<name>", "ships/<name>" (nur Namen aus dem Spiel-Kontext). `weather.cloudConfig` nach Muster "summer\\00_Clear_00\\sky.ini". `task` und `payload` wie bei DCS (Pflicht je Flugzeuggruppe): AIrborne setzt daraus `PayloadId` über die Tabelle `reference/il2-payloads.json` (Rolle, dann Wunsch-Übereinstimmung).

**DCS World (`game:"dcs"`):** `map` = Theatre-Name (Caucasus, Syria, PersianGulf, Nevada, Normandy, MarianaIslands, Sinai, Kola, Falklands). `country` = DCS-Country-String ("USA", "Russia", "Syria", "Israel", "Egypt", "Iran", ...). Alle Positionen (`start`, `position`, `route[]`, `zone`, `icons`) als `{"lat":35.40,"lon":35.95,"alt":2000,"heading":270}` in Dezimalgrad (KEIN x/z). `aircraft`/`script` = exakte DCS-`type`-Namen (z. B. "MiG-21Bis", "F-16C_50", "Mi-24P", "Mi-8MT", "M-113", "M-60", "Ural-375", "ZSU-23-4 Shilka", "BTR-80"). `statics[].script` ebenfalls ein Bodeneinheiten-Typ. `kind` "plane" oder "helicopter" bei Luftfahrzeugen. `weather` wie oben (wird sinngemäß übernommen).
**Bewaffnung DCS:** Jede Flugzeug-/Hubschraubergruppe (Spieler, Gegner, Verbündete) bekommt `task` (Pflicht) = Rolle: "CAP" | "Intercept" | "FighterSweep" | "Escort" | "CAS" | "GroundAttack" | "Strike" | "SEAD" | "AntiShip" | "Transport" | "Recon", und `payload` (Klartext-Wunsch mit DCS-Waffenbezeichnungen, z. B. "AIM-120C*4, AIM-9X*2" oder "S-8KOM, 9M114 Shturm" oder "AGM-88 HARM"). Der Generator wählt daraus das passende Mission-Editor-Preset des Typs; die Rolle steuert auch das Verhalten der KI (CAP greift Luftziele an, SEAD Luftabwehr, AntiShip Schiffe).

**Prefabs (nur DCS, optional):** Wenn im Spiel-Kontext Prefabs gelistet sind, kann der Plan sie über `"prefabs": [{"prefab":"<id>","name":"...","side":"player|friendly|enemy","country":"USA","position":{"lat":35.40,"lon":35.95,"heading":270}}]` platzieren (Ursprung + Ausrichtung des Prefabs; `country` optional). Die Einheiten eines Prefabs nicht zusätzlich einzeln auflisten. Ohne passende Prefabs das Feld weglassen.
