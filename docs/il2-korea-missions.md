# IL-2 Korea: Missionsdateien – Format, Werkzeuge, Workflow

Verifiziert in der Praxis (IL-2 Korea / IL2Series, Stand 09/2026, Mission `VCL_Chromite_Spione` erfolgreich im Editor geöffnet). Pfade beziehen sich auf
`C:\Program Files (x86)\Steam\steamapps\common\IL2Series` (= `%IL2ROOT%`).

---

## 1. Datei-Sets pro Mission

Eine Mission besteht aus einem Set von Dateien mit gleichem Basisnamen:

| Datei | Zweck | Format |
|---|---|---|
| `NAME.Mission` | **Die Mission selbst** (Objekte, MCU-Logik, Wetter, Startzeit) | Text, eigenes Key/Value-Format, CRLF |
| `NAME.msnbin` | Binär-Spiegel der Mission, vom Editor/Spiel geschrieben | Binär, proprietär |
| `NAME.ger` / `.eng` / `.rus` / `.fra` / `.spa` / `.chs` | Lokalisierte Texte (Name, Briefing, Autor, Objektiv-Texte, Funk/Subtitle) | UTF-16LE mit BOM, Zeilen `Index:Text` |
| `NAME.list` | Liste der Missionsbriefing-Bilder (leer = ok) | Text |

**Wichtig:**
- Das **Spiel und der Editor laden die Text-.Mission vollwertig** – offizielle Demo-Missionen (`data\Missions\[DEMO]*.Mission`, ~411k Zeilen) existieren als reine Textdateien **ohne** .msnbin. .msnbin nicht selbst editieren; er wird beim Speichern im Editor regeneriert.
- Die Sprachdateien sind **UTF-16LE mit BOM (FF FE)**, Zeilenformat `Index:Text`, CRLF. Umlaute sind kein Problem.
- `LCName/LCDesc/LCAuthor` in `Options` verweisen auf Sprach-Indizes 0/1/2.

## 2. Aufbau der .Mission (verifizierte Syntax)

```text
# Mission File Version = 1.0;

Options
{
  LCName = 0; LCDesc = 1; LCAuthor = 2;
  PlayerConfig = "LuaScripts\WorldObjects\Planes\il10.txt";
  Time = 15:35:0;                 -- H:M:S
  Date = 14.9.1950;               -- D.M.YYYY
  HMap = "graphics\LANDSCAPE_Korea_sp\height.hini";   -- Jahreszeit: _su Sommer, _sp Frühling/Herbst, _wi Winter, _sw ?
  Textures = "...textures.tini";
  Forests = "...trees\woods.wds";
  GuiMap = "landscape_korea_su";  -- GUI-Kartenname, unabhängig von SeasonPrefix
  SeasonPrefix = "su";
  MissionType = 1;                -- 0 = Single/Dogfight, 1 = Cooperative
  CloudLevel = 1500; CloudHeight = 6000; PrecLevel = 0; PrecType = 0;
  CloudConfig = "summer\00_Clear_00\sky.ini";   -- Pfad enthält Saison!
  SeaState = 3; Turbulence = 5; Temperature = 22; Pressure = 760;
  Haze = 0.4; LayerFog = 0; CloudsShift = 0.2;
  WindLayers { 0 : 150 : 4; 500 : 160 : 5; 1000 : 170 : 6; 2000 : 180 : 8; 5000 : 190 : 10; }  -- Höhe : Richtung : Stärke
  Countries { 0 : 0; 501 : 1; 502 : 1; 503 : 1; 601 : 2; }
}
```

**Ländercodes:** `0` = neutral, `5xx` = Ostblock/Korea-Nord (501,502,503), `6xx` = UN/USA (601,602,603). `Country = 501` für KPAF, `601` für UN. Werte in `Countries` = Koalition (0 neutral, 1 rot, 2 blau).

### Objekt-Blöcke (alle mit einheitlichem Muster)

```text
Plane
{
  Name = "Leader"; Index = 3; LinkTrId = 4;
  XPos = 86500.000; YPos = 1000.000; ZPos = 244000.000;   -- Meter, Karte
  XOri = 0; YOri = 40; ZOri = 0;                          -- Heading = YOri (0–360)
  Script = "LuaScripts\WorldObjects\Planes\il10.txt";
  Model  = "graphics\planes\il10\il10.mgm";
  Country = 501;
  AILevel = 4; CoopStart = 1;          -- CoopStart = 1 ⇒ belegbarer Spielerplatz (Coop)
  StartType = 0;                       -- 0 = in der Luft, 1 = am Parkstand/Runway
  NumberInFormation = 0;               -- 0 Leader, 1..n Wingman
  Callsign = 1; Callnum = 1; PayloadId = 0; Fuel = 1; Skin = ""; Emblem = 0;
  Vulnerable = 1; Engageable = 1; LimitAmmo = 1; DamageReport = 50; DamageThreshold = 1;
  DeleteAfterDeath = 1; DeleteAfterLand = 1; Spotter = -1;
  TCode = ""; TCodeColor = ""; GunLoad = []; GunBelt = []; VictoryCount = 0; ModMask = 1; AiRTBDecision = 0;
}
```

`Vehicle`, `Airfield`, `Block` (Gebäude) folgen demselben Muster (Airfield mit `Callsign=46`, `MaintenanceRadius`; Block zusätzlich `Flags = 0`). Fahrzeuge/Blöcke: `PinToTerrain = 1` zwingend, dann normalisiert das Spiel/der Editor die Y-Höhe aufs Terrain (Y nur als Richtwert nötig).

**Kopplungsregel (essentiell):** Jedes spielbare/triggerbare Objekt bekommt ein **Paar**: das Objekt (Index N, LinkTrId N+1) und eine `MCU_TR_Entity` (Index N+1, `MisObjID = N`). Alle MCU-Verweise (`Objects = [...]`) zeigen auf die **Entity**, nie direkt auf das Objekt.

## 3. MCU-Katalog (verifizierte Blöcke)

Alle MCUs: `Index`, `Name`, `Desc`, `Targets = [...]`, `Objects = [...]`, `XPos/YPos/ZPos`, Orientierung. `Targets` = Folge-Trigger (wer wird bei Auslösung aktiviert), `Objects` = betroffene Entities.

```text
MCU_TR_MissionBegin      -- Start-Event; Targets = Timer/Logik
{
  Index = 43; Name = "Mission Begin"; Desc = ""; Targets = [44, 51]; Objects = [];
  XPos = 88000.000; YPos = 5.000; ZPos = 244000.000; XOri = 0; YOri = 0; ZOri = 0; Enabled = 1;
}

MCU_Timer                -- Zeitverzögerung/Zufall; Targets = nächste MCUs
{
  Index = 44; Targets = [45]; Objects = []; Time = 10; Random = 100;   -- Sekunden; Random = % Trefferquote
}

MCU_CheckZone            -- Positions-Trigger
{
  Index = 47; Targets = [48]; Objects = [4, 6];   -- Entities, die die Zone betreten
  Zone = 2500; Cylinder = 1; Closer = 1;
}

MCU_Waypoint             -- Wegpunkt für Bewegte Objekte (Flug + Boden!)
{
  Index = 7; Targets = [8]; Objects = [4];        -- Objects = Entity-Index (TR_Entity!)
  Area = 2000; Speed = 320; Priority = 1;         -- Flug: Area groß, Speed in km/h
}                                                 -- Boden: Area 20–30, Speed 30–40; Loop durch wechselseitige Targets

MCU_Counter              -- Zähler; feuert Targets wenn Counter erreicht
{
  Index = 40; Targets = [41]; Objects = []; Counter = 7; Dropcount = 0;
}

MCU_TR_MissionObjective  -- Missionsziel
{
  Index = 42; Targets = []; Objects = []; Enabled = 1;
  LCName = 3; LCDesc = 4;       -- Sprach-Indizes für Titel/Beschreibung
  TaskType = 0; Coalition = 1; Success = 1; IconType = 559;
}

MCU_TR_Subtitle          -- Funk-/Untertitel-Text
{
  Index = 45; Targets = []; Objects = []; Enabled = 1;
  SubtitleInfo { Duration = 8; FontSize = 20; HAlign = 1; VAlign = 0; RColor = 255; GColor = 255; BColor = 255; LCText = 13; }
  Coalitions = [0, 1, 2];
}

MCU_Icon                 -- Kartenmarkierung (Linie vom Icon zum Ziel)
{
  Index = 53; Targets = [54]; Objects = []; Enabled = 1;
  LCName = 5; LCDesc = 6; IconId = 901; RColor = 255; GColor = 255; BColor = 255; LineType = 14; Coalitions = [1];
}
```

Weitere vorhandene MCU-Typen (aus der Inchon-Demo): `MCU_Activate`, `MCU_Deactivate`, `MCU_Counter`, `MCU_DateTime`, `MCU_CMD_AttackArea` (Parameter `AttackGround`, `AttackAir`, `AttackGTargets`, `AttackArea`, `Time`, `Priority`), `MCU_CMD_TakeOff` (`NoTaxiTakeoff`), `MCU_CMD_Land`, `MCU_CMD_Formation`, `MCU_CMD_Flare`, `MCU_CMD_Effect`, `MCU_CMD_Reposition`, `MCU_CMD_ForceComplete`, `MCU_TR_Media`, `MCU_TR_CameraOperator`, `MCU_H_PresetDescription`, `MCU_ModifierAddVal`.

**Event-Typen in `MCU_TR_Entity.OnEvents` (verifiziert):** `Type = 13` = Objekt **zerstört** (feuert `TarId`), `Type = 6` = Takeoff, `Type = 9`, `Type = 14`, `Type = 19` u. a. Kette für „alle Ziele vernichtet": jede Ziel-Entity → `OnEvent Type 13` → gemeinsamer `MCU_Counter` (Counter = Anzahl Ziele) → `MCU_Timer` (4 s Verzögerung) → Objective + Subtitle.

## 4. Asset-Referenz (in der Installation verifiziert)

Alle Skripte unter `data\LuaScripts\WorldObjects\...`, Modelle unter `data\graphics\...` (im Spiel gepackt in `*.gtp`):

- **Flugzeuge IL-2 Korea:** `il10`, `la11`, `mig15bis`, `yak9p`, `f51d`, `f80c10`, `f84e`, `f86a5`, `tu2`, `li2t`, `c47b` (Pfade `Planes\<name>.txt`)
- **Fahrzeuge:** `willysmb`, `studebakerus6` (+`-tanker`, `-refueler`, `-bm13`), `gaz55/63/67b`, `dodgewc52/54`, `m16-mgmc`, `m3a1-halftrack`, `m4a3e8`, `m46`, `t34-85`, `isu122`, `su76m`, `is2` (Ordner `vehicles\`)
- **Flak/Fixed:** `boforsl60`, `dshk-aa`, `dshk`, `m2cal50-aa`, `ks19`, `s60`, `m1aaa120`, `m2aaa90`, `zpuvz53` (`fixedobjects\`)
- **Schiffe:** `gleaves`, `libertyship`, `cargoship1/2`, `tankership`, `lst`, `lci`, `lcvp` (`ships\`)
- **Gebäude:** Ordner `blocks\` (`kor_*` Korea-Gebäude, `mil_*` Militär inkl. `mil_camonet`, `ind_*`, `port_*`, `rw_*`), Detailobjekte `blocksdetail\`, Brücken `bridges\`
- **Sky-Configs:** `data\graphics\...\sky.ini` nach Muster `Saison\NN_Name_TT\sky.ini` (z. B. `summer\00_Clear_00\`, `winter\02_Medium_07\`)
- **Karten (Korea):** `LANDSCAPE_Korea_su` (Sommer), `_sp` (Frühling/Herbst), `_wi` (Winter), `_sw` — **alle teilen sich dasselbe Koordinatengitter** (Airfields identisch platzierbar). Referenzkoordinaten: Kimpo K-14 (105934, 269477), Seoul K-16 (102349, 280779), Incheon-Stadt ~ (96000, 252000–256000), Hafeneinfahrt/Palmido ~ (95100–98600, 249800–253500).

## 5. Empfohlener Workflow (wie in der Praxis angewandt)

1. **Recherche** (historische Grundlage, Wikipedia o. ä.) → Seite, Zeit, Fraktionen, Einheiten, taktische Lage.
2. **Referenz-Missionen studieren:** `data\Missions\[DEMO]*.Mission`, `data\Multiplayer\Cooperative\_test_cooperative_basic.Mission` (Coop-Platz-Logik `CoopStart = 1`, LinkTrId-Verkettung, TCode).
3. **.Mission als Text schreiben** (nicht .msnbin!) — Objekte paarweise mit TR_Entity, Logik-Kette MissionBegin → Timer → Subtitle, Zonen/Waypoints/Counter/Objective.
4. **Sprachdateien** `.ger`/`.eng` als UTF-16LE mit BOM, `Index:Text` pro Zeile (Name=0, Desc=1, Author=2, dann Objective- und Subtitle-Texte), `.list` leer anlegen.
5. **Editor starten** – zwingend mit Arbeitsverzeichnis `bin\editor` (sonst findet er die .gtp-Archive → Fehler `Graphics\Shaders\Trash\AtmSphere.fx not found` und gefülltes `data\tex.log`). Am besten über Steam starten oder `-WorkingDirectory` korrekt setzen.
6. Mission laden, **Höhen/Placement prüfen** (PinToTerrain zieht Bodenobjekte aufs Terrain), im Editor speichern → .msnbin entsteht.
7. Test im Spiel (Coop-Liste), Iteration.

### Typische Stolperfallen
- `.msnbin` manuell anfassen → nein. Nur Text-.Mission editieren, Editor regeneriert den Rest.
- Editor falsch gestartet (CWD) → alle Shader/Texturen „fehlen" (fxerr.log/tex.log im `data\`-Ordner zeigen FAILED loads).
- MCU-Verweise zeigen immer auf **TR_Entity-Indizes**, nicht auf Objekt-Indizes.
- Missionsdateien ohne `LinkTrId`-Paarung oder mit doppelten `Index` → Ladefehler.
- `PlayerConfig` in Options muss zur Spielerflugezeug-Script-Passage passen (z. B. `LuaScripts\WorldObjects\Planes\il10.txt`).
- Coop-Missionen: für jeden Spielerplatz ein eigenes `Plane`-Objekt mit `CoopStart = 1`, gemeinsame Formation über `NumberInFormation`/`Callsign`.
