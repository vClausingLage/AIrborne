import { useState } from 'react'
import { pipeline } from '../wailsjs/go/models'

interface Props {
  game: string
  aircraft: pipeline.AircraftOption[]
  maps: string[]
  busy: string
  onStart: (aircraft: string, mapHint: string) => void
}

const OTHER = '__other__'

// Herausforderung: the player picks only the aircraft (and optionally a DCS
// map); the LLM designs a challenging scenario whose enemy details never
// reach the briefing.
export default function ChallengePane({ game, aircraft, maps, busy, onStart }: Props) {
  const [choice, setChoice] = useState('')
  const [custom, setCustom] = useState('')
  const [map, setMap] = useState('')
  const isDcs = game === 'dcs'
  const type = choice === OTHER ? custom.trim() : choice
  const red = aircraft.filter((a) => a.side === 'red')
  const blue = aircraft.filter((a) => a.side === 'blue')
  const any = aircraft.filter((a) => a.side !== 'red' && a.side !== 'blue')

  const group = (label: string, list: pipeline.AircraftOption[]) =>
    list.length > 0 && (
      <optgroup label={label}>
        {list.map((a) => (
          <option key={a.type} value={a.type}>
            {a.label}
          </option>
        ))}
      </optgroup>
    )

  return (
    <section className="pane">
      <div className="pane-col narrow">
        <h3>🎯 Herausforderung – nur das Flugzeug wählen</h3>
        <div className="form-grid">
          <label>Flugzeug</label>
          <select value={choice} onChange={(e) => setChoice(e.target.value)}>
            <option value="">– wählen –</option>
            {group(isDcs ? 'Blau / West' : 'UN / USA', blue)}
            {group(isDcs ? 'Rot / Ost' : 'KPAF / VVS', red)}
            {group('Beide Seiten', any)}
            {isDcs && <option value={OTHER}>Anderes Modul (Typname eingeben)…</option>}
          </select>
          {choice === OTHER && (
            <>
              <label>DCS-Typ</label>
              <input value={custom} onChange={(e) => setCustom(e.target.value)} placeholder='z. B. "A-4E-C"' />
            </>
          )}
          {isDcs && (
            <>
              <label>Karte</label>
              <select value={map} onChange={(e) => setMap(e.target.value)}>
                <option value="">KI wählt passend</option>
                {maps.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            </>
          )}
        </div>
        <button className="primary" onClick={() => onStart(type, map)} disabled={busy !== '' || type === ''}>
          {busy === 'challenge' ? 'Entwirft Herausforderung…' : 'Herausforderung generieren'}
        </button>
      </div>
      <div className="pane-col">
        <h3>So funktioniert es</h3>
        <div className="hint" style={{ fontSize: 13, lineHeight: 1.7, wordBreak: 'normal' }}>
          Die KI entwirft aus deinem Flugzeug allein eine kleine, fordernde Mission: Epoche, Fraktion, Karte, Wetter, Auftrag und
          Gegner wählt sie selbst – und zwar so, dass das Flugzeug gefordert wird (Jäger: Überzahl und Positionsnachteil plus Flak,
          Bodenangriff: verteidigte Ziele mit Jagdschutz in der Nähe, Hubschrauber: mobile Luftabwehr).
          <br />
          <br />
          <b>Alle Gegnerinformationen bleiben verborgen.</b> Briefing, Missionsziel und Funksprüche nennen weder Typen noch Anzahl
          noch Positionen; Karten-Icons entfallen. Der Plan wird zusätzlich deterministisch geprüft: Erscheint ein Gegnertyp in einem
          Spielertext, meldet „Plan &amp; Export“ eine Warnung.
          {isDcs && (
            <>
              <br />
              <br />
              <b>DCS:</b> Alle Gegnergruppen werden „Hidden on map/planner/MFD“ exportiert, und die F10-Karte wird per{' '}
              <code>forcedOptions</code> auf das eigene Flugzeug beschränkt.
            </>
          )}
          <br />
          <br />
          Nach dem Generieren unter „Plan &amp; Export“ prüfen und exportieren. Der Plan enthält die Gegner natürlich vollständig –
          wer nicht spoilern will, klappt das JSON nicht auf.
        </div>
      </div>
    </section>
  )
}
