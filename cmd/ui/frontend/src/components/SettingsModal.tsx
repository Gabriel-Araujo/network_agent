import { useCallback, useEffect, useRef, useState } from 'react'
import { Plus, Trash, X, CaretRight } from '@phosphor-icons/react'
import '../Settings.css'
import { EMPTY_PROVIDER, ProviderConfig, Settings } from '../types'
import ModelsSection from './ModelsSection'

interface SettingsModalProps {
  open: boolean
  onClose: () => void
  /** Notifica o App a cada gravação, para o pill do topo refletir na hora. */
  onSettingsChanged?: (s: Settings) => void
}

const api = () => (window as any).go?.main?.App

export default function SettingsModal({ open, onClose, onSettingsChanged }: SettingsModalProps) {
  const [settings, setSettings] = useState<Settings>({ providers: [] })
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [draft, setDraft] = useState<ProviderConfig | null>(null)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const dialogRef = useRef<HTMLDivElement>(null)

  const load = useCallback(async () => {
    const a = api()
    if (!a?.GetSettings) return
    try {
      const s: Settings = await a.GetSettings()
      setSettings(s)
      onSettingsChanged?.(s)
      setSelectedId((cur) => cur ?? s.providers[0]?.id ?? null)
    } catch (e) {
      setError(String(e))
    }
  }, [onSettingsChanged])

  useEffect(() => {
    if (open) {
      load()
      setError('')
    }
  }, [open, load])

  // Fecha com Escape e prende o foco dentro do modal.
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    dialogRef.current?.focus()
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  // Sincroniza o formulário quando muda o provedor selecionado.
  useEffect(() => {
    if (creating) return
    const p = settings.providers.find((x) => x.id === selectedId) ?? null
    setDraft(p ? { ...p } : null)
  }, [selectedId, settings, creating])

  if (!open) return null

  const apply = (s: Settings) => {
    setSettings(s)
    onSettingsChanged?.(s)
    setError('')
  }

  const startCreate = () => {
    setCreating(true)
    setSelectedId(null)
    setDraft({ id: '', ...EMPTY_PROVIDER })
  }

  const cancelCreate = () => {
    setCreating(false)
    setSelectedId(settings.providers[0]?.id ?? null)
  }

  const saveProvider = async () => {
    if (!draft) return
    const a = api()
    if (!a) return setError('Bindings do Go indisponíveis.')
    setSaving(true)
    try {
      const s: Settings = creating
        ? await a.CreateProvider(draft)
        : await a.UpdateProvider(draft)
      apply(s)
      if (creating) {
        setCreating(false)
        const novo = s.providers[s.providers.length - 1]
        setSelectedId(novo?.id ?? null)
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setSaving(false)
    }
  }

  const removeProvider = async (id: string) => {
    const a = api()
    if (!a) return
    try {
      const s: Settings = await a.DeleteProvider(id)
      apply(s)
      setSelectedId(s.providers[0]?.id ?? null)
    } catch (e) {
      setError(String(e))
    }
  }

  const set = <K extends keyof ProviderConfig>(k: K, v: ProviderConfig[K]) =>
    setDraft((d) => (d ? { ...d, [k]: v } : d))

  return (
    <div className="settings-scrim" onMouseDown={onClose} role="presentation">
      <div
        className="settings"
        role="dialog"
        aria-modal="true"
        aria-label="Configurações"
        tabIndex={-1}
        ref={dialogRef}
        onMouseDown={(e) => e.stopPropagation()}
      >
        <header className="settings__head">
          <h2 className="settings__title">Configurações</h2>
          <button className="icon-btn" onClick={onClose} aria-label="Fechar configurações" title="Fechar">
            <X size={16} weight="bold" aria-hidden="true" />
          </button>
        </header>

        {error && <p className="settings__error">{error}</p>}

        <div className="settings__body">
          <nav className="settings__list" aria-label="Provedores">
            <div className="settings__list-head">
              <span className="settings__list-title">Provedores</span>
              <button className="icon-btn" onClick={startCreate} aria-label="Novo provedor" title="Novo provedor">
                <Plus size={15} weight="bold" aria-hidden="true" />
              </button>
            </div>

            {settings.providers.length === 0 && !creating && (
              <p className="settings__empty">Nenhum provedor cadastrado.</p>
            )}

            <ul className="settings__items">
              {settings.providers.map((p) => (
                <li key={p.id}>
                  <button
                    className={`settings__item ${p.id === selectedId && !creating ? 'settings__item--active' : ''}`}
                    onClick={() => {
                      setCreating(false)
                      setSelectedId(p.id)
                    }}
                  >
                    <span className={`settings__dot ${p.enabled ? 'settings__dot--on' : ''}`} aria-hidden="true" />
                    <span className="settings__item-text">
                      <span className="settings__item-name">{p.name}</span>
                      <span className="settings__item-sub">
                        {p.models.length} {p.models.length === 1 ? 'modelo' : 'modelos'}
                      </span>
                    </span>
                    <CaretRight size={13} weight="bold" aria-hidden="true" />
                  </button>
                </li>
              ))}
              {creating && (
                <li>
                  <span className="settings__item settings__item--active settings__item--new">Novo provedor</span>
                </li>
              )}
            </ul>
          </nav>

          <section className="settings__pane">
            {!draft ? (
              <p className="settings__empty">Selecione um provedor à esquerda ou crie um novo.</p>
            ) : (
              <>
                <div className="settings__pane-head">
                  <h3 className="settings__pane-title">
                    {creating ? 'Novo provedor' : 'Editar provedor'}
                  </h3>
                  {!creating && (
                    <button className="btn-danger" onClick={() => removeProvider(draft.id)}>
                      <Trash size={14} weight="bold" aria-hidden="true" />
                      Excluir
                    </button>
                  )}
                </div>

                <div className="field">
                  <label className="field__label" htmlFor="prov-name">Nome</label>
                  <input
                    id="prov-name"
                    className="field__input"
                    value={draft.name}
                    onChange={(e) => set('name', e.target.value)}
                    placeholder="LM Studio (local)"
                  />
                </div>

                <div className="field">
                  <label className="field__label" htmlFor="prov-url">URL base</label>
                  <input
                    id="prov-url"
                    className="field__input"
                    value={draft.baseURL}
                    onChange={(e) => set('baseURL', e.target.value)}
                    placeholder="http://localhost:1234/v1"
                  />
                </div>

                <div className="field">
                  <label className="field__label" htmlFor="prov-key">Chave de API</label>
                  <input
                    id="prov-key"
                    className="field__input"
                    type="password"
                    value={draft.apiKey}
                    onChange={(e) => set('apiKey', e.target.value)}
                    placeholder="deixe vazio para provedores locais"
                  />
                </div>

                <label className="switch">
                  <input
                    type="checkbox"
                    checked={draft.enabled}
                    onChange={(e) => set('enabled', e.target.checked)}
                  />
                  <span>Provedor ativo</span>
                </label>

                <div className="settings__actions">
                  {creating && (
                    <button className="btn-ghost" onClick={cancelCreate}>Cancelar</button>
                  )}
                  <button className="btn-primary" onClick={saveProvider} disabled={saving}>
                    {saving ? 'Salvando...' : creating ? 'Criar provedor' : 'Salvar alterações'}
                  </button>
                </div>

                {/* CRUD de modelos: vive dentro da seção de edição do provedor */}
                {!creating && (
                  <ModelsSection
                    provider={settings.providers.find((p) => p.id === draft.id) ?? draft}
                    onChanged={apply}
                    onError={setError}
                  />
                )}
                {creating && (
                  <p className="settings__hint">
                    Crie o provedor para poder cadastrar os modelos dele.
                  </p>
                )}
              </>
            )}
          </section>
        </div>
      </div>
    </div>
  )
}