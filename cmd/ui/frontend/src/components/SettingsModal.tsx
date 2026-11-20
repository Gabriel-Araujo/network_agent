import { useCallback, useEffect, useRef, useState } from 'react'
import { Plus, Trash, X, CaretRight } from '@phosphor-icons/react'
import '../Settings.css'
import { AppConfig, EMPTY_PROVIDER, ProviderConfig } from '../types'
import ModelsSection from './ModelsSection'
import { App } from '../../bindings/github.com/Gabriel-Araujo/network_agent/cmd/ui'
import { errorText, inWails } from '../types'

interface SettingsModalProps {
  open: boolean
  onClose: () => void
  /** Notifica o App a cada gravação, para o pill do topo refletir na hora. */
  onConfigChanged?: (c: AppConfig) => void
  /** false = configuração obrigatória pendente: não pode fechar. */
  dismissible?: boolean
  /** Um turno roda: trocar de Workspace fica bloqueado. */
  workspaceLocked?: boolean
}

export default function SettingsModal({
  open,
  onClose,
  onConfigChanged,
  dismissible = true,
  workspaceLocked = false,
}: SettingsModalProps) {
  const [config, setConfig] = useState<AppConfig>({ selected: 0, providers: [] })
  // índice do provedor selecionado; null = nenhum
  const [selectedIdx, setSelectedIdx] = useState<number | null>(null)
  const [draft, setDraft] = useState<ProviderConfig | null>(null)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const dialogRef = useRef<HTMLDivElement>(null)

  // O callback do App fica num ref para que `load` tenha identidade estável.
  // Com `onConfigChanged` nas deps, cada render do App criava um `load` novo e
  // o efeito abaixo reexecutava sem parar (GetConfig ~800x/s com o modal aberto).
  const onConfigChangedRef = useRef(onConfigChanged)
  useEffect(() => {
    onConfigChangedRef.current = onConfigChanged
  }, [onConfigChanged])

  const load = useCallback(async () => {
    if (!inWails()) return
    try {
      const c: AppConfig = await App.GetConfig()
      setConfig(c)
      onConfigChangedRef.current?.(c)
      if (c.providers.length) {
        setSelectedIdx((cur) => cur ?? 0)
        setDraft((cur) => cur ?? { ...c.providers[0] })
      }
    } catch (e) {
      setError(errorText(e))
    }
  }, [])

  // Carrega a configuração uma única vez por abertura do modal.
  useEffect(() => {
    if (open) {
      load()
      setError('')
    }
  }, [open, load])

  // Fecha com Escape.
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && dismissible) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose, dismissible])

  // Foca o diálogo apenas quando ele abre, e só se o foco ainda não estiver
  // dentro dele. Reexecutar este efeito a cada render do App roubava o foco do
  // input que o usuário acabara de clicar (o foco durava ~1 ms).
  useEffect(() => {
    if (!open) return
    const el = dialogRef.current
    if (el && !el.contains(document.activeElement)) el.focus()
  }, [open])

  // O draft é definido explicitamente por quem muda de contexto (selectProvider,
  // startCreate, cancelCreate). Não há efeito de sincronização: ele rodava após
  // o commit e, com selectedIdx = null durante a criação, apagava o formulário.

  if (!open) return null

  const apply = (c: AppConfig) => {
    setConfig(c)
    onConfigChanged?.(c)
    setError('')
  }

  const selectProvider = (i: number) => {
    setCreating(false)
    setSelectedIdx(i)
    const p = config.providers[i]
    setDraft(p ? { ...p } : null)
  }

  const startCreate = () => {
    setCreating(true)
    setSelectedIdx(null)
    setDraft({ ...EMPTY_PROVIDER })
  }

  const cancelCreate = () => {
    setCreating(false)
    if (config.providers.length) selectProvider(0)
    else {
      setSelectedIdx(null)
      setDraft(null)
    }
  }

  const saveProvider = async () => {
    if (!draft) return
    if (!inWails()) return setError('Bindings do Go indisponíveis.')
    setSaving(true)
    try {
      const c: AppConfig = creating
        ? await App.CreateProvider(draft)
        : await App.UpdateProvider(selectedIdx!, draft)
      apply(c)
      if (creating) {
        setCreating(false)
        const idx = c.providers.length - 1
        setSelectedIdx(idx)
        setDraft({ ...c.providers[idx] })
      }
    } catch (e) {
      setError(errorText(e))
    } finally {
      setSaving(false)
    }
  }

  const removeProvider = async (idx: number) => {
    if (!inWails()) return
    try {
      const c: AppConfig = await App.DeleteProvider(idx)
      apply(c)
      if (c.providers.length) {
        setSelectedIdx(0)
        setDraft({ ...c.providers[0] })
      } else {
        setSelectedIdx(null)
        setDraft(null)
      }
    } catch (e) {
      setError(errorText(e))
    }
  }

  // O diálogo é nativo e mora no Go; cancelar devolve a configuração como
  // está, então não há caso de "cancelei e perdi a pasta".
  const pickWorkspace = async () => {
    if (!inWails()) return setError('Bindings do Go indisponíveis.')
    try {
      apply(await App.PickWorkspace())
    } catch (e) {
      setError(errorText(e))
    }
  }

  const useProvider = async (idx: number) => {
    if (!inWails()) return
    try {
      apply(await App.SelectProvider(idx))
    } catch (e) {
      setError(errorText(e))
    }
  }

  const set = <K extends keyof ProviderConfig>(k: K, v: ProviderConfig[K]) =>
    setDraft((d) => (d ? { ...d, [k]: v } : d))

  return (
    <div
      className="settings-scrim"
      onMouseDown={() => dismissible && onClose()}
      role="presentation"
    >
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
          {dismissible && (
            <button className="icon-btn" onClick={onClose} aria-label="Fechar configurações" title="Fechar">
              <X size={16} weight="bold" aria-hidden="true" />
            </button>
          )}
        </header>

        {!dismissible && (
          <p className="settings__notice">
            Cadastre um provedor com ao menos um modelo para começar a usar o agente.
          </p>
        )}
        {error && <p className="settings__error">{error}</p>}

        {/* O Workspace é do app, não do provedor (ticket 05, §4): fica acima
            do grid de provedores, não dentro do formulário de um deles. */}
        <div className="field settings__workspace">
          <span className="field__label">Workspace — a única pasta onde o agente grava</span>
          <div className="settings__workspace-row">
            <span
              className={`field__input settings__workspace-path ${
                config.workspace ? '' : 'settings__workspace-path--empty'
              }`}
              title={config.workspace || ''}
            >
              {config.workspace || 'Nenhuma pasta escolhida — o agente não grava arquivos.'}
            </span>
            <button
              className="btn-ghost btn-ghost--sm"
              onClick={pickWorkspace}
              disabled={workspaceLocked}
              title={workspaceLocked ? 'Trocar de Workspace fica bloqueado enquanto o agente responde.' : undefined}
            >
              {config.workspace ? 'Trocar...' : 'Escolher...'}
            </button>
          </div>
        </div>

        <div className="settings__body">
          <nav className="settings__list" aria-label="Provedores">
            <div className="settings__list-head">
              <span className="settings__list-title">Provedores</span>
              <button className="icon-btn" onClick={startCreate} aria-label="Novo provedor" title="Novo provedor">
                <Plus size={15} weight="bold" aria-hidden="true" />
              </button>
            </div>

            {config.providers.length === 0 && !creating && (
              <p className="settings__empty">Nenhum provedor cadastrado.</p>
            )}

            <ul className="settings__items">
              {config.providers.map((p, i) => (
                <li key={i}>
                  <button
                    className={`settings__item ${i === selectedIdx && !creating ? 'settings__item--active' : ''}`}
                    onClick={() => selectProvider(i)}
                  >
                    <span
                      className={`settings__dot ${i === config.selected ? 'settings__dot--on' : ''}`}
                      aria-hidden="true"
                    />
                    <span className="settings__item-text">
                      <span className="settings__item-name">{p.name}</span>
                      <span className="settings__item-sub">
                        {p.models.length} {p.models.length === 1 ? 'modelo' : 'modelos'}
                        {i === config.selected ? ' · em uso' : ''}
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
                  {!creating && selectedIdx != null && (
                    <div className="settings__pane-actions">
                      {selectedIdx !== config.selected && (
                        <button className="btn-ghost btn-ghost--sm" onClick={() => useProvider(selectedIdx)}>
                          Usar este
                        </button>
                      )}
                      <button className="btn-danger" onClick={() => removeProvider(selectedIdx)}>
                        <Trash size={14} weight="bold" aria-hidden="true" />
                        Excluir
                      </button>
                    </div>
                  )}
                </div>

                <div className="field">
                  <label className="field__label" htmlFor="prov-name">Nome</label>
                  <input
                    id="prov-name"
                    className="field__input"
                    value={draft.name}
                    onChange={(e) => set('name', e.target.value)}
                    placeholder="OpenAI Compatible"
                  />
                </div>

                <div className="field">
                  <label className="field__label" htmlFor="prov-url">URL base</label>
                  <input
                    id="prov-url"
                    className="field__input"
                    value={draft.url}
                    onChange={(e) => set('url', e.target.value)}
                    placeholder="http://localhost:1234/v1"
                  />
                </div>

                <div className="field">
                  <label className="field__label" htmlFor="prov-key">Chave de API</label>
                  <input
                    id="prov-key"
                    className="field__input"
                    type="password"
                    value={draft.api_key}
                    onChange={(e) => set('api_key', e.target.value)}
                    placeholder="deixe vazio para provedores locais"
                  />
                </div>

                <div className="settings__actions">
                  {creating && (
                    <button className="btn-ghost" onClick={cancelCreate}>Cancelar</button>
                  )}
                  <button className="btn-primary" onClick={saveProvider} disabled={saving}>
                    {saving ? 'Salvando...' : creating ? 'Criar provedor' : 'Salvar alterações'}
                  </button>
                </div>

                {/* CRUD de modelos: dentro da seção de edição do provedor */}
                {!creating && selectedIdx != null && (
                  <ModelsSection
                    provider={config.providers[selectedIdx] ?? draft}
                    providerIndex={selectedIdx}
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
