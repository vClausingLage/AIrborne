# Missionsplan aus Nutzer-Vorgaben erstellen

Deine Aufgabe ist es, aus den folgenden Vorgaben des Nutzers EINE vollständige, in sich konsistente, historisch/fiktiv plausible Mission für eine Kampfflugzeug-Simulation zu entwerfen und als Missionsplan (JSON) auszugeben. Du bist Missions-Chefdesigner: Du vervollständigst Lücken sinnvoll, löst Widersprüche (Zeit, Ort, Einheiten, Fraktionen, Wetter) zugunsten der Story und der historischen Plausibilität und vermeidest Anachronismen (Einheiten/Ausrüstung nur nach Epochen-Prüfung). Nicht ausgefüllte Abschnitte entscheidest du selbst passend zur Story.

{{GAME_CONTEXT}}

# Story / Lage
_(Krieg/Jahr/Monat, Theater und Region, taktische Gesamtlage beider Seiten, warum es diesen Einsatz gibt, strategisches Ziel, plausibles Wetter, Tageszeit)_

{{USER_STORY}}

# Player / Spielergruppe
_(Typ und Anzahl der Spielerflugzeuge passend zur Epoche und zur Simulation, Fraktion, Verband/Callsign, Startart und Ort, Payload/Treibstoff, Rollen in der Formation)_

{{USER_PLAYER}}

# Gegner
_(Gegnerische Einheiten in Luft und am Boden, Typen und Anzahl, Positionen relativ zu Orten der Karte, Verhalten: statisch/Patrouille/Kolonne, Reaktion auf den Spieler)_

{{USER_ENEMY}}

# Widerstand
_(Flak und Luftabwehr im Zielgebiet, Patrouillen, Bodenwiderstand, Verbündete am Boden, statische Objekte wie Tarnnetze/Gebäude, realistisch dosiert)_

{{USER_RESISTANCE}}

# Briefing / Funk
_(Ton und Sprache des Briefings, Callsigns, Funksprüche: Intro nach Missionsstart, Kontakt-/Statusmeldungen, Erfolgsmeldung, Zeitlimit)_

{{USER_BRIEFING}}

# Anforderungen an den Plan
1. Alle Objekte bekommen realistische Positionen (Koordinatensystem siehe Spiel-Kontext) und passen räumlich zusammen (Anflugweg, Zielgebiet, Rückweg).
2. Wegpunkte: Die Spielergruppe erhält eine Route Anflug/IP → Zielgebiet (über den Gegnerpositionen) → Rückweg, die eine realistische Aufgabe ergibt. Alle beweglichen Einheiten (`movement.type: "route"`, Flugzeuge, Hubschrauber, Schiffe) erhalten eigene Routen, die ihr Verhalten abbilden (Kolonne fährt zum Ziel, Patrouille kreist, CAP-Orbit) und den Kontakt mit dem Spieler sinnvoll herstellen.
3. Bewaffnung: Jede Flugzeug-/Hubschraubergruppe bekommt eine Rolle `task` (CAP, CAS, SEAD, Strike, AntiShip, Intercept, Escort, Transport …) und ein plausibles `payload` (Typ, Epoche, Aufgabe). Spieler und KI-Flugzeuge ohne `task`/`payload` sind nicht erlaubt.
4. Genau 1 Spielergruppe, 1-4 Gegnergruppen (mind. ein bekämpfbares Ziel), Flak/Statics passend zur Vorgabe, genau 1 primäres Missionsziel.
5. 2-5 Funksprüche (Intro, Kontakt, Erfolg, ggf. Zeitlimit), authentischer Ton, max. 2 Sätze je Meldung.
6. Briefing DE + EN. Titel DE + EN.
7. Der Plan muss SCHEMA-KONFORM sein. Füge keine Felder hinzu, lass optionale weg.

{{SCHEMA}}
