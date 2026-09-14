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
2. Genau 1 Spielergruppe, 1-4 Gegnergruppen (mind. ein bekämpfbares Ziel), Flak/Statics passend zur Vorgabe, genau 1 primäres Missionsziel.
3. 2-5 Funksprüche (Intro, Kontakt, Erfolg, ggf. Zeitlimit), authentischer Ton, max. 2 Sätze je Meldung.
4. Briefing DE + EN. Titel DE + EN.
5. Der Plan muss SCHEMA-KONFORM sein. Füge keine Felder hinzu, lass optionale weg.

{{SCHEMA}}
