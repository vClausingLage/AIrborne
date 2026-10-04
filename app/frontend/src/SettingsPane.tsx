import { useEffect, useState } from 'react'
import { pipeline } from '../wailsjs/go/models'
import { Lang, useT } from './i18n'

interface Props {
  config: pipeline.ConfigState
  busy: string
  lang: Lang
  onSaveMaxTokens: (n: number) => void
  onSetLanguage: (lang: Lang) => void
}

// Settings: the LLM/token limit and UI language the app writes to
// settings.json, plus the read-only values that come from .env so the user
// sees what is in effect.
export default function SettingsPane({ config, busy, lang, onSaveMaxTokens, onSetLanguage }: Props) {
  const t = useT()
  const fmt = (n: number) => n.toLocaleString(t.locale)
  const [draft, setDraft] = useState(String(config.maxTokens))
  useEffect(() => setDraft(String(config.maxTokens)), [config.maxTokens])

  const parsed = parseInt(draft.replace(/[.,\s_]/g, ''), 10)
  const valid = !isNaN(parsed) && (parsed === 0 || (parsed >= 1000 && parsed <= 1_000_000))
  const dirty = valid && parsed !== config.maxTokens
  const usesEnv = config.maxTokens === config.maxTokensEnv
  const usage = config.lastUsage
  const share = usage && config.maxTokens > 0 ? Math.round((usage.completion_tokens / config.maxTokens) * 100) : 0

  return (
    <section className="pane">
      <div className="pane-col">
        <h3>{t.settingsAi}</h3>
        <div className="form-grid">
          <label>{t.language}</label>
          <div>
            <select value={lang} onChange={(e) => onSetLanguage(e.target.value as Lang)} disabled={busy !== ''} style={{ width: 160 }}>
              <option value="de">Deutsch</option>
              <option value="en">English</option>
            </select>
            <div className="hint" style={{ lineHeight: 1.6 }}>{t.languageHint}</div>
          </div>

          <label>{t.maxTokens}</label>
          <div>
            <div className="row" style={{ marginTop: 0 }}>
              <input
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && dirty && onSaveMaxTokens(parsed)}
                inputMode="numeric"
                style={{ width: 140 }}
              />
              <button className="primary" onClick={() => onSaveMaxTokens(parsed)} disabled={busy !== '' || !dirty}>
                {busy === 'settings' ? t.saving : t.apply}
              </button>
              {!usesEnv && (
                <button onClick={() => onSaveMaxTokens(0)} disabled={busy !== ''} title={t.resetToEnvTitle}>
                  {t.defaultN(fmt(config.maxTokensEnv))}
                </button>
              )}
            </div>
            {!valid && <div className="hint warn">{t.maxTokensInvalid}</div>}
            <div className="hint" style={{ lineHeight: 1.6 }}>
              {t.maxTokensHint}
              {usesEnv ? t.maxTokensUsesEnv : t.maxTokensEnvDefault(fmt(config.maxTokensEnv))}
            </div>
          </div>

          <label>{t.lastRequest}</label>
          <div className="hint" style={{ lineHeight: 1.6 }}>
            {usage ? (
              <>
                {t.usage(fmt(usage.prompt_tokens), fmt(usage.completion_tokens), fmt(usage.total_tokens))}
                {share > 0 && t.usageShare(share)}
              </>
            ) : (
              t.noRequestYet
            )}
          </div>
        </div>

        <h3 style={{ marginTop: 20 }}>{t.fromEnv}</h3>
        <div className="form-grid">
          <label>{t.model}</label>
          <div className="hint">{config.model || t.modelMissing}</div>
          <label>Endpoint</label>
          <div className="hint">{config.base}</div>
          <label>{t.apiKey}</label>
          <div className="hint">{config.hasKey ? t.keySet : t.keyMissing}</div>
          <label>{t.il2Missions}</label>
          <div className="hint">{config.il2MissionsDir || t.il2MissionsMissing}</div>
          <label>{t.dcsMissions}</label>
          <div className="hint">{config.dcsMissionsDir || t.dcsMissionsMissing}</div>
          <label>Prompts</label>
          <div className="hint">{config.promptsDir}</div>
          <label>{t.settingsFile}</label>
          <div className="hint">{config.settingsPath}</div>
          <label>{t.log}</label>
          <div className="hint">{config.logPath || t.notInitialized}</div>
        </div>
      </div>
    </section>
  )
}
