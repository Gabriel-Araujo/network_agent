import { useState } from 'react'
import { Plus, Trash, PencilSimple, Star } from '@phosphor-icons/react'
import { AppConfig, EMPTY_MODEL, ModelConfig, ProviderConfig } from '../types'
import { errorText, goApp } from '../wails'

interface ModelsSectionProps {
  provider: ProviderConfig
  providerIndex: number
  onChanged: (c: AppConfig) => void
  onError: (msg: string) => void
}

const fmt = (n: number) => n.toLocaleString('pt-BR')

export default function ModelsSection({
  provider,
  providerIndex,
  onChanged,
  onError,
}: ModelsSectionProps) {
  const [draft, setDraft] = useState<ModelConfig | null>(null)
  // índice do modelo em edição; null = criando
  const [editIdx, setEditIdx] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)

  const creating = draft !== null && editIdx === null

  const startCreate = () => {
    setEditIdx(null)
    setDraft({ ...EMPTY_MODEL })
  }
  const startEdit = (m: ModelConfig, i: number) => {
    setEditIdx(i)
    setDraft({ ...m })
  }
  const cancel = () => {
    setDraft(null)
    setEditIdx(null)
  }

  const set = <K extends keyof ModelConfig>(k: K, v: ModelConfig[K]) =>
    setDraft((d) => (d ? { ...d, [k]: v } : d))

  const save = async () => {
    if (!draft) return
    const a = goApp()
    if (!a) return onError('Bindings do Go indisponíveis.')
    setSaving(true)
    try {
      const c: AppConfig = creating
        ? await a.CreateModel(providerIndex, draft)
        : await a.UpdateModel(providerIndex, editIdx, draft)
      onChanged(c)
      cancel()
    } catch (e) {
      onError(errorText(e))
    } finally {
      setSaving(false)
    }
  }

  const remove = async (i: number) => {
    const a = goApp()
    if (!a) return
    try {
      onChanged(await a.DeleteModel(providerIndex, i))
      if (editIdx === i) cancel()
    } catch (e) {
      onError(errorText(e))
    }
  }

  const setDefault = async (i: number) => {
    const a = goApp()
    if (!a) return
    try {
      onChanged(await a.SetDefaultModel(providerIndex, i))
    } catch (e) {
      onError(errorText(e))
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
        <p className="settings__empty">
          Nenhum modelo cadastrado. Cadastre ao menos um para poder conversar.
        </p>
      )}

      <ul className="models__list">
        {provider.models.map((m, i) => (
          <li key={i} className="models__row">
            <span className="models__row-main">
              <span className="models__name">
                {m.name || m.id}
                {i === provider.default_model && (
                  <span className="models__badge" title="Modelo padrão">
                    <Star size={10} weight="fill" aria-hidden="true" />
                    padrão
                  </span>
                )}
              </span>
              <span className="models__meta">
                id {m.id} · contexto {fmt(m.context_size)}
              </span>
            </span>
            <span className="models__row-actions">
              {i !== provider.default_model && (
                <button
                  className="icon-btn"
                  onClick={() => setDefault(i)}
                  aria-label={`Tornar ${m.name || m.id} o padrão`}
                  title="Tornar padrão"
                >
                  <Star size={14} weight="bold" aria-hidden="true" />
                </button>
              )}
              <button
                className="icon-btn"
                onClick={() => startEdit(m, i)}
                aria-label={`Editar ${m.name || m.id}`}
                title="Editar"
              >
                <PencilSimple size={14} weight="bold" aria-hidden="true" />
              </button>
              <button
                className="icon-btn icon-btn--danger"
                onClick={() => remove(i)}
                aria-label={`Excluir ${m.name || m.id}`}
                title="Excluir"
              >
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
            <label className="field__label" htmlFor="mod-id">
              ID do modelo (usado na API)
            </label>
            <input
              id="mod-id"
              className="field__input"
              value={draft.id}
              onChange={(e) => set('id', e.target.value)}
              placeholder="google/gemma-4-12b-qat"
            />
          </div>

          <div className="field">
            <label className="field__label" htmlFor="mod-name">Nome de exibição</label>
            <input
              id="mod-name"
              className="field__input"
              value={draft.name}
              onChange={(e) => set('name', e.target.value)}
              placeholder="Gemma 4 12B"
            />
          </div>

          <div className="field">
            <label className="field__label" htmlFor="mod-ctx">Contexto máximo (tokens)</label>
            <input
              id="mod-ctx"
              className="field__input"
              type="number"
              min={0}
              value={draft.context_size}
              onChange={(e) => set('context_size', Number(e.target.value))}
            />
          </div>

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
