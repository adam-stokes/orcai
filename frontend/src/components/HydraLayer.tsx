import { useEffect, useState, useCallback, useMemo } from 'react';
import { StartSession, StopSession } from '../wailsjs/go/main/App';

// ── Concrete provider bindings ───────────────────────────────────────────────

type ProviderId = 'claude' | 'gemini' | 'copilot' | 'codex' | 'opencode' | 'ollama' | 'shell';

interface ModelOption {
  id: string;
  label: string;
  separator?: boolean;
}

interface ProviderDef {
  key: string;
  id: ProviderId;
  label: string;
  models?: ModelOption[];
}

const PROVIDERS: ProviderDef[] = [
  {
    key: 'C', id: 'claude', label: 'Claude',
    models: [
      { id: 'claude-opus-4-6',   label: 'Opus 4.6'   },
      { id: 'claude-sonnet-4-6', label: 'Sonnet 4.6' },
      { id: 'claude-haiku-4-5',  label: 'Haiku 4.5'  },
    ],
  },
  {
    key: 'G', id: 'gemini', label: 'Gemini',
    models: [
      { id: 'gemini-2.5-pro',    label: 'Gemini 2.5 Pro'    },
      { id: 'gemini-2.0-flash',  label: 'Gemini 2.0 Flash'  },
      { id: 'gemini-1.5-pro',    label: 'Gemini 1.5 Pro'    },
      { id: '', label: '── Vertex AI ──', separator: true },
      { id: 'claude-sonnet-4-6', label: 'Claude Sonnet 4.6' },
      { id: 'claude-opus-4-6',   label: 'Claude Opus 4.6'   },
      { id: 'claude-haiku-4-5',  label: 'Claude Haiku 4.5'  },
    ],
  },
  { key: 'P', id: 'copilot', label: 'Copilot' },
  {
    key: 'X', id: 'codex', label: 'Codex',
    models: [
      { id: 'o3',      label: 'o3'      },
      { id: 'gpt-4.1', label: 'GPT-4.1' },
      { id: 'gpt-4o',  label: 'GPT-4o'  },
    ],
  },
  { key: 'O', id: 'opencode', label: 'OpenCode' },
  { key: 'L', id: 'ollama',   label: 'Ollama'   },
  { key: 'S', id: 'shell',    label: 'Shell'    },
];

const SESSION_KEYS = '123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('');

function shortModel(model: string) {
  const base = model.split('/').pop() ?? model;
  const parts = base.split('-');
  return parts.length > 2 ? parts.slice(1).join('-') : base;
}

// ── Types ────────────────────────────────────────────────────────────────────

interface Session {
  id: string;
  provider: string;
  model: string;
}

interface HydraLayerProps {
  sessions: Session[];
  activeId: string | null;
  ollamaModels: string[];
  onSelectSession: (id: string) => void;
  onNewSession: (id: string, provider: string, model: string) => void;
  onStopSession: (id: string) => void;
  onShowHelp: () => void;
}

// ── Component ────────────────────────────────────────────────────────────────

export function HydraLayer({
  sessions, activeId, ollamaModels,
  onSelectSession, onNewSession, onStopSession, onShowHelp,
}: HydraLayerProps) {
  const [active, setActive] = useState(false);
  const [submenu, setSubmenu] = useState<ProviderDef | null>(null);
  const [killMode, setKillMode] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const providers = useMemo<ProviderDef[]>(() => {
    if (ollamaModels.length === 0) return PROVIDERS;
    return PROVIDERS.map((p) => {
      if (p.id === 'ollama') {
        return { ...p, models: ollamaModels.map((m) => ({ id: m, label: m })) };
      }
      if (p.id === 'opencode') {
        const sep: ModelOption = { id: '', label: '── Local (Ollama) ──', separator: true };
        const extra = ollamaModels.map((m) => ({ id: `ollama/${m}`, label: m }));
        return { ...p, models: [...(p.models ?? []), sep, ...extra] };
      }
      return p;
    });
  }, [ollamaModels]);

  const dismiss = useCallback(() => {
    setActive(false);
    setSubmenu(null);
    setKillMode(false);
    setError(null);
  }, []);

  const launch = useCallback(async (provider: ProviderDef, model = '') => {
    setLoading(true);
    setError(null);
    const id = String(Date.now());
    try {
      await StartSession(id, provider.id, model);
      onNewSession(id, provider.id, model);
      dismiss();
    } catch (e: any) {
      setError(String(e?.message ?? e));
    } finally {
      setLoading(false);
    }
  }, [onNewSession, dismiss]);

  const killSession = useCallback(async (id: string) => {
    try {
      await StopSession(id);
    } catch (e) {
      console.error('stop session:', e);
    }
    onStopSession(id);
    dismiss();
  }, [onStopSession, dismiss]);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      const isLeader = (e.ctrlKey || e.metaKey) && e.key === ';';

      if (!active) {
        if (!isLeader) return;
        e.preventDefault();
        setActive(true);
        return;
      }

      // Ignore lone modifier keydowns — they arrive before the real key
      // (e.g. Shift fires before '?' from Shift+/) and would hit dismiss().
      if (['Shift', 'Control', 'Alt', 'Meta'].includes(e.key)) return;

      e.preventDefault();

      if (e.key === 'Escape') {
        if (killMode) { setKillMode(false); return; }
        if (submenu)  { setSubmenu(null);   return; }
        dismiss();
        return;
      }

      const key = e.key.toUpperCase();

      // Kill-mode: choose session to stop
      if (killMode) {
        const idx = SESSION_KEYS.indexOf(key);
        if (idx >= 0 && idx < sessions.length) killSession(sessions[idx].id);
        else setKillMode(false);
        return;
      }

      // Model sub-menu
      if (submenu) {
        let idx = 0;
        for (const m of submenu.models ?? []) {
          if (m.separator) continue;
          if (key === String.fromCharCode(65 + idx)) { launch(submenu, m.id); return; }
          idx++;
        }
        return;
      }

      // Session switch
      const sessIdx = SESSION_KEYS.indexOf(key);
      if (sessIdx >= 0 && sessIdx < sessions.length) {
        onSelectSession(sessions[sessIdx].id);
        dismiss();
        return;
      }

      // Provider launch / sub-menu
      const provider = providers.find((p) => p.key === key);
      if (provider) {
        if (provider.models && provider.models.length > 0) setSubmenu(provider);
        else launch(provider);
        return;
      }

      // Commands
      if (key === 'K') {
        if (sessions.length === 0) return;
        if (sessions.length === 1) killSession(sessions[0].id);
        else setKillMode(true);
        return;
      }
      if (key === '?') { dismiss(); onShowHelp(); return; }

      dismiss();
    }

    window.addEventListener('keydown', onKeyDown, { capture: true });
    return () => window.removeEventListener('keydown', onKeyDown, { capture: true });
  }, [active, submenu, killMode, sessions, providers, onSelectSession, onShowHelp, dismiss, launch, killSession]);

  if (!active) return null;

  const titleText = submenu
    ? `ORCAI  ›  ${submenu.label.toUpperCase()}`
    : killMode ? 'ORCAI  ›  KILL SESSION' : 'ORCAI';

  return (
    <div
      onClick={dismiss}
      style={{
        position: 'fixed', inset: 0,
        background: 'var(--overlay-bg)',
        zIndex: 300,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          background: 'var(--bg-panel)',
          border: '1px solid var(--border)',
          fontFamily: 'monospace',
          fontSize: '12px',
          color: 'var(--text-normal)',
          minWidth: '440px',
          maxWidth: '600px',
          userSelect: 'none',
        }}
      >
        {/* Title */}
        <div style={{
          padding: '5px 12px',
          borderBottom: '1px solid var(--border)',
          color: 'var(--text-active)',
          fontWeight: 700,
          fontSize: '11px',
          letterSpacing: '0.1em',
        }}>
          ─ {titleText}
        </div>

        <div style={{ padding: '10px 14px' }}>

          {/* Kill-mode */}
          {killMode && (
            <>
              <div style={{ color: 'var(--text-muted)', fontSize: '10px', letterSpacing: '0.08em', marginBottom: '8px' }}>
                SELECT SESSION TO STOP
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 20px' }}>
                {sessions.map((s, i) => (
                  <div key={s.id} onClick={() => killSession(s.id)}
                    style={{ display: 'flex', gap: '6px', cursor: 'pointer' }}>
                    <span style={{ color: 'var(--error)' }}>[{SESSION_KEYS[i]}]</span>
                    <span style={{ color: 'var(--text-dim)', fontSize: '11px' }}>
                      {s.provider}{s.model ? ':' + shortModel(s.model) : ''}
                    </span>
                  </div>
                ))}
              </div>
              <div style={{ marginTop: '10px', borderTop: '1px solid var(--border)', paddingTop: '6px', color: 'var(--text-muted)', fontSize: '10px' }}>
                [Esc] back
              </div>
            </>
          )}

          {/* Model sub-menu */}
          {!killMode && submenu && (
            <>
              <div style={{ color: 'var(--text-muted)', fontSize: '10px', letterSpacing: '0.08em', marginBottom: '8px' }}>
                SELECT MODEL
              </div>
              {(() => {
                let idx = 0;
                return (submenu.models ?? []).map((m, i) => {
                  if (m.separator) return (
                    <div key={`sep-${i}`} style={{ color: 'var(--text-muted)', fontSize: '10px', padding: '3px 0' }}>{m.label}</div>
                  );
                  const hk = String.fromCharCode(65 + idx++);
                  return (
                    <div key={m.id} onClick={() => launch(submenu, m.id)}
                      style={{ display: 'flex', gap: '10px', padding: '2px 0', cursor: 'pointer' }}>
                      <span style={{ color: 'var(--accent)', minWidth: '28px' }}>[{hk}]</span>
                      <span style={{ color: 'var(--text-dim)' }}>{m.label}</span>
                    </div>
                  );
                });
              })()}
              <div style={{ marginTop: '10px', borderTop: '1px solid var(--border)', paddingTop: '6px', color: 'var(--text-muted)', fontSize: '10px' }}>
                [Esc] back
              </div>
            </>
          )}

          {/* Main hydra body */}
          {!killMode && !submenu && (
            <>
              {sessions.length > 0 && (
                <div style={{ marginBottom: '12px' }}>
                  <div style={{ color: 'var(--text-muted)', fontSize: '10px', letterSpacing: '0.08em', marginBottom: '6px' }}>SESSIONS</div>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 20px' }}>
                    {sessions.map((s, i) => (
                      <div key={s.id} onClick={() => { onSelectSession(s.id); dismiss(); }}
                        style={{ display: 'flex', gap: '6px', cursor: 'pointer' }}>
                        <span style={{ color: s.id === activeId ? 'var(--text-active)' : 'var(--accent)' }}>
                          [{SESSION_KEYS[i]}]
                        </span>
                        <span style={{ color: s.id === activeId ? 'var(--text-active)' : 'var(--text-dim)', fontSize: '11px' }}>
                          {s.provider}{s.model ? ':' + shortModel(s.model) : ''}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              <div style={{ marginBottom: '10px' }}>
                <div style={{ color: 'var(--text-muted)', fontSize: '10px', letterSpacing: '0.08em', marginBottom: '6px' }}>NEW SESSION</div>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 20px' }}>
                  {providers.map((p) => (
                    <div key={p.id}
                      onClick={() => { if (p.models?.length) setSubmenu(p); else launch(p); }}
                      style={{ display: 'flex', alignItems: 'center', gap: '6px', minWidth: '130px', cursor: 'pointer' }}>
                      <span style={{ color: 'var(--accent)' }}>[{p.key}]</span>
                      <span style={{ color: 'var(--text-dim)', fontSize: '11px' }}>{p.label}</span>
                      {p.models && p.models.length > 0 && (
                        <span style={{ color: 'var(--text-muted)', fontSize: '10px' }}>›</span>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {sessions.length > 0 && (
                <div style={{ marginBottom: '10px' }}>
                  <div style={{ color: 'var(--text-muted)', fontSize: '10px', letterSpacing: '0.08em', marginBottom: '6px' }}>COMMANDS</div>
                  <div style={{ display: 'flex', gap: '20px' }}>
                    <div onClick={() => sessions.length === 1 ? killSession(sessions[0].id) : setKillMode(true)}
                      style={{ display: 'flex', gap: '6px', cursor: 'pointer' }}>
                      <span style={{ color: 'var(--error)' }}>[K]</span>
                      <span style={{ color: 'var(--text-dim)', fontSize: '11px' }}>kill session</span>
                    </div>
                  </div>
                </div>
              )}

              {error && (
                <div style={{ color: 'var(--error)', fontSize: '10px', marginBottom: '8px', wordBreak: 'break-word' }}>{error}</div>
              )}

              <div style={{
                borderTop: '1px solid var(--border)', paddingTop: '6px',
                display: 'flex', justifyContent: 'space-between',
                fontSize: '10px', color: 'var(--text-muted)',
              }}>
                <span><span style={{ color: 'var(--accent)' }}>[?]</span> help</span>
                <span>{loading ? '…connecting' : '[Esc] cancel'}</span>
              </div>
            </>
          )}

        </div>
      </div>
    </div>
  );
}
