# Modus: Herausforderung (Blind-Skirmish)

Der Spieler hat NUR sein Flugzeug gewählt. Du entwirfst dafür eine kleine, fordernde Mission – und der Spieler darf vorab **nichts** über die Gegner erfahren. Er muss aufklären, reagieren und improvisieren.

{{GAME_CONTEXT}}

**Spielerflugzeug:** {{AIRCRAFT}}
**Karte:** {{MAP_HINT}}

**Aufgabe:** Wähle selbst Epoche, Fraktion des Spielers (passend zum Flugzeug), Karte/Region, Datum, Tageszeit, Wetter, Startpunkt und einen glaubwürdigen Auftrag (z. B. Abfangen, Patrouille über einem Sektor, Freie Jagd, Bewaffnete Aufklärung, Angriff auf ein gemeldetes Ziel). Baue daraus eine **fordernde** Skirmish (10-20 Minuten): 2-4 Gegnergruppen, die die Stärken des Spielerflugzeugs auf die Probe stellen (bei Jägern: Luftgegner in Überzahl oder mit Positionsvorteil plus Flak im Zielraum; bei Bodenangriff: gut verteidigte Ziele mit Flak und einer Jagdpatrouille in der Nähe; bei Hubschraubern: mobile Luftabwehr, MANPADS/Flak und Bodenverbände). Mindestens ein bekämpfbares Ziel, genau 1 primäres Missionsziel (`counter` = Anzahl zerstörbarer Zielfahrzeuge bzw. -flugzeuge), 2-4 Funksprüche, Briefing DE + EN. Die Spielergruppe hat genau 1 Spielerplatz (`count`: 1) und ein plausibles, vielseitiges Standard-Payload. Keine Anachronismen. Eine Route Anflug → Einsatzraum → Rückweg, alle beweglichen Gegner mit eigenen Routen, die den Kontakt herbeiführen.

**Absolute Regel – keine Gegnerinformationen für den Spieler:**
- `briefing`, `objectives[].title/desc`, `radioQueue[].text*` und `icons` dürfen KEINE gegnerischen Typen, Anzahlen, Positionen, Höhen, Bewaffnung oder Verhalten nennen – auch nicht umschrieben („zwei Panzer", „MiG-Rotte", „Flak-Stellung bei …"). Erlaubt sind nur vage Meldungen („Aufklärung meldet Aktivität im Raum X", „Rechne mit Gegenwehr", „Kontakt unbekannt, Peilung 030").
- Das Briefing beschreibt Lage, Auftrag, Einsatzraum (grob), Rückkehr und Regeln – sonst nichts. Der Auftrag darf ein *vermutetes* Ziel nennen (z. B. „Versorgungsverkehr auf der Straße nach …"), aber keine Details zu dessen Schutz.
- Funksprüche informieren nur über den Missionsverlauf („Auftrag läuft", „Zielraum erreicht", Erfolg), nie über Typen oder Zahlen des Gegners. Ein Kontakt-Funkspruch nennt höchstens eine grobe Richtung.
- `icons`: leer lassen (`[]`).
- Die vollständigen Gegnerdaten gehören ausschließlich in `enemyGroups` und `flak`.

{{SCHEMA}}
