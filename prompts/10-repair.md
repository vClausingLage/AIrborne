# Modus: Reparatur (Korrektur eines bestehenden Plans)

Der folgende Missionsplan wurde bereits von dir erstellt. Die Validierung und ein Probelauf des Missionsgenerators haben Probleme gemeldet. Deine Aufgabe ist es, **nur diese Probleme zu beheben** und den vollständigen, korrigierten Plan zurückzugeben.

{{GAME_CONTEXT}}

**Gemeldete Probleme:**
{{ISSUES}}

**Regeln für die Korrektur:**
- Behebe jedes gemeldete Problem an der Stelle, auf die es sich bezieht (Feldpfade wie `enemyGroups[1].script` zeigen auf das betroffene Element).
- Alles andere bleibt **unverändert**: Story, Karte, Datum, Einheiten, Positionen, Routen, Bewaffnung, Funksprüche, Briefing-Texte. Keine neuen Ideen, keine Umformulierungen, nichts weglassen.
- Ist ein Typ-/Script-Name unbekannt, ersetze ihn durch den passenden gültigen Namen aus dem Spiel-Kontext (gleiche Epoche, gleiche Rolle) – nicht durch etwas völlig anderes. Ist ein Name laut Kontext korrekt, lass ihn stehen.
- Fehlen Ziele oder Positionen, ergänze das Minimum, das der Plan braucht, passend zur bestehenden Story.
- Meldet die Prüfung Gegnerinformationen in Spieler-Texten (Herausforderung), formuliere nur die betroffenen Sätze neutral um.
- Gib den **gesamten** Plan zurück, nicht nur die geänderten Teile.

**Bisheriger Plan:**
```json
{{PLAN_JSON}}
```

{{SCHEMA}}
