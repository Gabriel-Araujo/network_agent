import { useState } from 'react'
import { Plus, Trash, PencilSimple, Star } from '@phosphor-icons/react'
import { EMPTY_MODEL, ModelConfig, ProviderConfig, Settings } from '../types'

interface ModelsSectionProps {
  provider: ProviderConfig
  onChanged: (s: Settings) => void
  onError: (msg: string) => void
}

const api = () => (window as any).go?.main?.App

const fmt = (n: number) => n.toLocaleString('pt-BR')

export default function ModelsSection({ provider, onChanged, onError }: ModelsSectionProps) {
  // draft === null -> nenhum formulário aberto
  // draft.id === '' -> criando; caso contrário, editando
  const [draft, setDraft] = useState<ModelConfig | null>(null)
  const [saving, setSaving] = useState(false)

  const creating = draft?.id === ''

  const startCreate = () => setDraft({ id: '', ...EMPTY_MODEL })
  const startEdit = (m: ModelConfig) => setDraft({ ...m })
  const cancel = () => setDraft(null)

  const set = <K extends keyof ModelConfig>(k: K, v: ModelConfig[K]) =>
    setDraft((d) => (d ? { ...d, [k]: v } : d))

  const save = async () => {
    if (!draft) return
    const a = api()
    if (!a) return onError('Bindings do Go indisponíveis.')
    setSaving(true)
    try {
      const s: Settings = creating
        ? await a.CreateModel(provider.id, draft)
        : await a.UpdateModel(provider.id, draft)
      onChanged(s)
      setDraft(null)
    } catch (e) {
      onError(String(e))
    } finally {
      setSaving(false)
    }
  }

  const remove = async (modelID: string) => {
    const a = api()
    if (!a) return
    try {
      const s: Settings = await a.DeleteModel(provider.id, modelID)
      onChanged(s)
      if (draft?.id === modelID) setDraft(null)
    } catch (e) {
      onError(String(e))
    }
  }

  return (
    <section className="models">
      <div className="models__head">
        <h4 className="models__title">Modelos</h4>
        <button className="btn-ghost btn-ghost--sm" onClick={startCreate}>
          <Plus size={14} weight="bold" aria-hidden="true" />
          Novo modelo
        </button>
      </div>

      {provider.models.length === 0 && !draft && (
        <p className="settings__empty">Nenhum modelo cadastrado neste provedor.</p>
      )}

      <ul className="models__list">
        {provider.models.map((m) => (
          <li key={m.id} className="models__row">
            <span className="models__row-main">
              <span className="models__name">
                {m.name}
                {m.default && (
                  <span className="models__badge" title="Modelo padrão">
                    <Star size={10} weight="fill" aria-hidden="true" />
                    padrão
                  </span>
                )}
              </span>
              <span className="models__meta">
                contexto {fmt(m.maxContext)} · saída {fmt(m.maxTokens)} · temp {m.temperature}
              </span>
            </span>
            <span className="models__row-actions">
              <button className="icon-btn" onClick={() => startEdit(m)} aria-label={`Editar ${m.name}`} title="Editar">
                <PencilSimple size={14} weight="bold" aria-hidden="true" />
              </button>
              <button className="icon-btn icon-btn--danger" onClick={() => remove(m.id)} aria-label={`Excluir ${m.name}`} title="Excluir">
                <Trash size={14} weight="bold" aria-hidden="true" />
              </button>
            </span>
          </li>
        ))}
      </ul>

      {draft && (
        <div className="models__form">
          <p className="models__form-title">{creating ? 'Novo modelo' : 'Editar modelo'}</p>

          <div className="field">
            <label className="field__label" htmlFor="mod-name">Nome do modelo</label>
            <input
              id="mod-name"
              className="field__input"
              value={draft.name}
              onChange={(e) => set('name', e.target.value)}
              placeholder="google/gemma-4-12b-qat"
            />
          </div>

          <div className="field-grid">
            <div className="field">
              <label className="field__label" htmlFor="mod-ctx">Contexto máximo (tokens)</label>
              <input
                id="mod-ctx"
                className="field__input"
                type="number"
                min={1}
                value={draft.maxContext}
                onChange={(e) => set('maxContext', Number(e.target.value))}
              />
            </div>

            <div className="field">
              <label className="field__label" htmlFor="mod-out">Saída máxima (tokens)</label>
              <input
                id="mod-out"
                className="field__input"
                type="number"
                min={1}
                value={draft.maxTokens}
                onChange={(e) => set('maxTokens', Number(e.target.value))}
              />
            </div>

            <div className="field">
              <label className="field__label" htmlFor="mod-temp">Temperatura</label>
              <input
                id="mod-temp"
                className="field__input"
                type="number"
                min={0}
                max={2}
                step={0.1}
                value={draft.temperature}
                onChange={(e) => set('temperature', Number(e.target.value))}
              />
            </div>

            <div className="field">
              <label className="field__label" htmlFor="mod-topp">Top P</label>
              <input
                id="mod-topp"
                className="field__input"
                type="number"
                min={0}
                max={1}
                step={0.05}
                value={draft.topP}
                onChange={(e) => set('topP', Number(e.target.value))}
              />
            </div>
          </div>

          <label className="switch">
            <input
              type="checkbox"
              checked={draft.default}
              onChange={(e) => set('default', e.target.checked)}
            />
            <span>Usar como modelo padrão deste provedor</span>
          </label>

          <div className="settings__actions">
            <button className="btn-ghost" onClick={cancel}>Cancelar</button>
            <button className="btn-primary" onClick={save} disabled={saving}>
              {saving ? 'Salvando...' : creating ? 'Criar modelo' : 'Salvar modelo'}
            </button>
          </div>
        </div>
      )}
    </section>
  )
}