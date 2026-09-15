# DCS: .miz-Dateien – Entpacken, Editieren, Zurückpacken

Verifiziert anhand einer echten Referenz-Mission (`GROZA_v1.8_SP_Mi24.miz`, Theatre Syria, 8,8 MB). Pfade: `%DCS_ROOT%` = `C:\Program Files\Eagle Dynamics\DCS World` (typisch; Missions-Output z. B. `C:\Users\<user>\Saved Games\DCS\Missions\`).

---

## 1. Grundprinzip

Eine `.miz` ist ein **normales ZIP-Archiv** ohne Kompressions-Exoten:

```
NAME.miz  --(umbenennen zu .zip)-->  entpacken  --(editieren)-->  zippen  --(umbenennen zu .miz)-->  DCS lädt es
```

PowerShells `Expand-Archive` verweigert die Endung `.miz` → **erst zu `.zip` kopieren/umbenennen**.

## 2. Struktur im Archiv (verifiziert)

```
mission          -- Die Mission: Lua-Tabellen-Text, OHNE Dateiendung, UTF-8. Kernstück.
options          -- Lua-Tabelle: Schwierigkeit, Realism-Optionen der Mission
theatre          -- 5-Byte-Textdatei: Kartenname (z. B. "Syria", "Caucasus", "Korea")
warehouses       -- Lua: Logistik/Lager der Fraktionen
KNEEBOARD\IMAGES\*.png                  -- Kneeboard-Seiten (alle Flugzeuge; alphabetische Reihenfolge → AIrborne nummeriert 01_, 02_ …)
l10n\DEFAULT\
    dictionary   -- UTF-8 (kein BOM): Lua-Tabelle mit ALLEN Texten (Briefings, Funk, Trigger-Texte)
    mapResource  -- Lua-Tabelle: ResKey -> Dateiname (Sound/Bild-Ressourcen)
    *.ogg        -- Radio-/ATC-Sounds
    *.png        -- Briefing-Bilder, Intro-Bilder, Banner
```

### `mission` (Auszug der relevanten Top-Level-Keys)

```lua
mission =
{
  ["theatre"] = "Syria",            -- (auch in theatre-Datei)
  ["date"] = { ["Year"]=2018, ["Month"]=9, ["Day"]=25 },
  ["start_time"] = 21900,           -- Missionsstart in Sekunden nach Mitternacht (6:05 → 21900)
  ["coalition"] = { ... },          -- blue/red: Countries → Gruppen → Units
  ["trig"] = {
    ["actions"] = {                 -- Lua-Snippets als Strings, z. B.:
      [1] = "a_out_text_delay(getValueDictByKey(\"DictKey_ActionText_66\"), 4, false, 0); mission.trig.func[1]=nil;",
      [2] = "a_out_sound(getValueResourceByKey(\"ResKey_Action_62\"), 0);a_out_picture(...);",
      [5] = "a_ai_task(65, 1); mission.trig.func[5]=nil;",
    },
    ["zones"], ["rules"] ...
  },
  ["triggers"] = { ... },           -- Zonen (TriggerZones: Typ 0 = zylindrisch, 2 = Polar...),
  ["map"], ["weather"], ["airdromes"], ["nav_points"], ...
}
```

**Texte leben NIEMALS direkt in `mission`**, sondern:
1. Trigger-Action-String referenziert `DictKey_ActionText_66` / `ResKey_Action_62`
2. `l10n/DEFAULT/dictionary` definiert `["DictKey_ActionText_66"] = "Funktext…"` (UTF-8, `\`-escaped Zeilenumbrüche im Lua-String)
3. `l10n/DEFAULT/mapResource` mappt `["ResKey_Action_62"] = "INTRO_SOUND2.ogg"` auf die Datei im selben Ordner

→ **Für AI-generierte Briefings/Funksprüche:** nur `dictionary` (Texte) + `mapResource` (ogg/png-Referenzen) + die Trigger-Action-Strings editieren. DCS-Trigger-System kennt u. a. `a_out_text_delay`, `a_out_sound`, `a_out_picture`, `a_ai_task`, `a_remove_scene_objects`, `a_set_ATC_silent_mode` (alle in `trig.actions` als kompakte Lua-Zeilen mit `mission.trig.func[N]=nil;` am Ende).

### Units/Gruppen (Muster in `mission`)

```lua
["coalition"] = {
  ["blue"] = {
    ["country"] = {
      [1] = { ["name"]="USA", ["plane"]={ ["group"]={ ["gonna_go"] = {
        ["units"] = { [1] = { ["type"]="F-16C_50", ["x"]=..., ["y"]=..., ["alt"]=..., ["heading"]=..., ["payload"]={...} } },
        ["route"] = { ["points"]={ [1]={ ["x"]=..., ["y"]=..., ["type"]="TakeOff", ... } } },
        ["tasks"] = {...}, ["start_time"]=0, ...
```

Achtung: DCS-Koordinaten sind **nicht Meter wie in IL-2**, sondern interne Kartekoordinaten (doppelte Genauigkeit, mapabhängig, `x` ≈ Nord, `y` ≈ Ost); Konversion nur über Map-Daten (Terrain-Script oder aus bestehenden Missionen ableiten).

## 3. Verifizierter Roundtrip (PowerShell)

```powershell
# 1) Entpacken (Endung ist die Hürde)
Copy-Item "C:\Users\vince\Downloads\GROZA_v1.8_SP_Mi24.miz" "$env:TEMP\work.zip"
Expand-Archive "$env:TEMP\work.zip" "$env:TEMP\miz\extracted"

# 2) Editieren: mission / l10n\DEFAULT\dictionary / mapResource / Medien

# 3) Zurückpacken – WICHTIG: Forward-Slashes in ZIP-Einträgen!
#    System.IO.Compression (nicht Compress-Archive, das \-Trenner schreiben kann):
Add-Type -AssemblyName System.IO.Compression, System.IO.Compression.FileSystem
$dst = "$env:TEMP\out.miz"
[IO.Compression.ZipFile]::CreateFromDirectory("$env:TEMP\miz\extracted", $dst,
  [IO.Compression.CompressionLevel]::Optimal, $false)   # $false = ohne Basisordner → mission im Root

# 4) In Saved Games legen, DCS lädt .miz direkt
```

Gotchas (wichtig für die App-Implementierung):
- **`mission` und `theatre` liegen ohne Ordner im ZIP-Root** – beim Zippen keinen übergeordneten Ordner einpacken (`includeBaseDirectory = $false`).
- **UTF-8 ohne BOM** für `mission`, `dictionary`, `mapResource`, `theatre`; keine CRLF-Umwandlung bei Lua-Dateien nötig, aber `\r\n` in Strings escapen (`\` + Zeilenumbruch = Lua-Zeilenfortführung, wie im dictionary-Beispiel).
- **Zippen mit `/`-Separatoren**: In Go kein Problem (`archive/zip` schreibt `toSlash`-Pfade); in PowerShell ggf. `ZipArchive`-API statt `Compress-Archive` verwenden.
- **Manche gekauften .miz sind geschützt** (mission binär/verschlüsselt) → Editierbarkeit vorher prüfen (`mission` beginnt mit `mission =`?).
- **Medien**: OGG (beliebige Bitrate, DCS wandelt), PNG/JPG für Briefings; Einträge in `mapResource` nicht vergessen, sonst findet DCS die Datei nicht. Briefing-Bilder hängen zusätzlich an `mission.pictureFileNameB` / `pictureFileNameR` (Array der ResKeys je Koalition; AIrborne setzt beim Generieren die Spielerseite, im Missions-Editor beide).
- **Kneeboards**: PNGs unter `KNEEBOARD/IMAGES/` im Archiv brauchen keinen `mapResource`-Eintrag; DCS zeigt sie in jedem Flugzeug. Flugzeugspezifisch wäre `KNEEBOARD/<Typ>/IMAGES/`.
- **Herausforderung (AIrborne)**: Gegnergruppen bekommen `["hidden"] = true` (+ `hiddenOnPlanner`, `hiddenOnMFD`), und `["forcedOptions"] = { ["optionsView"] = "optview_myaircraft" }` beschränkt die F10-Karte.
- **Editieren mit AIrborne** (`app/internal/missionfile`): Top-Level-Felder (`sortie`, `descriptionText`, `start_time`, `date`, `pictureFileName*`) werden an der ersten Einrückungsebene erkannt – Gruppen haben ebenfalls `start_time`, liegen aber tiefer.
- Vor jedem Edit die Original-.miz sichern; DCS ist beim Parsen streng (fehlendes Komma = Mission lädt nicht, Fehler im dcs.log unter `Saved Games\DCS\Logs\dcs.log`).

## 4. Assets/Zeit in DCS (Planungsdokument, in App als Datenmodell)

Anders als IL-2 Korea hängt das Flugzeug-/Einheitenangebot in DCS von **Karte + Epoche + gekauften Modulen** ab:

- **Kalter Krieg früh (50er):** Korea-Assets nur teilweise (Map „Korea" TFW + MiG-15/F-86 Module), begrenzte Bodeneinheiten
- **Mittlerer KK (60er–80er):** Syria (1973/1982 Szenarien), Sinai, F-4E, Mirage F1, MiG-21, A-4, …
- **Später/modern:** F-16, F/A-18, MiG-29, Su-27, Drohnen, Modern-AAW
- Vorgehen in der App: **Asset-Katalog als JSON** (`reference/dcs-assets.json`): Einheit → Module (benötigtes DLC) → Fraktion(en) → Epochen → Karte(n)-Verfügbarkeit. Missionsgenerator filtert danach; IL-2 entsprechend `reference/il2-assets.md` (siehe docs/il2-korea-missions.md §4).
