import { useState } from 'react'
import {
  ActionButton,
  ActionButtonGroup,
  Badge,
  Button,
  ListView,
  ListViewItem,
  NumberField,
  Text,
  TextField,
  Tooltip,
  TooltipTrigger,
} from '@react-spectrum/s2'
import { style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Add from '@react-spectrum/s2/icons/Add'
import Delete from '@react-spectrum/s2/icons/Delete'
import Edit from '@react-spectrum/s2/icons/Edit'
import Star from '@react-spectrum/s2/icons/Star'
import { AppConfig, EMPTY_MODEL, ModelConfig, ProviderConfig } from '../types'
import { App } from '../../bindings/github.com/Gabriel-Araujo/network_agent/cmd/ui'
import { errorText, inWails } from '../types'

interface ModelsSectionProps {
  provider: ProviderConfig
  providerIndex: number
  onChanged: (c: AppConfig) => void
  onError: (msg: string) => void
}

const fmt = (n: number) => n.toLocaleString('pt-BR')

// Altura de uma linha do ListView (nome e descrição); a lista é
// virtualizada, então precisa de altura definida.
const ROW = 52

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
    if (!inWails()) return onError('Bindings do Go indisponíveis.')
    setSaving(true)
    try {
      const c: AppConfig = creating
        ? await App.CreateModel(providerIndex, draft)
        : await App.UpdateModel(providerIndex, editIdx!, draft)
      onChanged(c)
      cancel()
    } catch (e) {
      onError(errorText(e))
    } finally {
      setSaving(false)
    }
  }

  const remove = async (i: number) => {
    if (!inWails()) return
    try {
      onChanged(await App.DeleteModel(providerIndex, i))
      if (editIdx === i) cancel()
    } catch (e) {
      onError(errorText(e))
    }
  }

  const setDefault = async (i: number) => {
    if (!inWails()) return
    try {
      onChanged(await App.SetDefaultModel(providerIndex, i))
    } catch (e) {
      onError(errorText(e))
    }
  }

  return (
    <section className={section}>
      <div className={head}>
        <h4 className={title}>Modelos</h4>
        <Button variant="secondary" size="S" onPress={startCreate}>
          <Add />
          <Text>Novo modelo</Text>
        </Button>
      </div>

      {provider.models.length === 0 && !draft && (
        <p className={hint}>Nenhum modelo cadastrado. Cadastre ao menos um para poder conversar.</p>
      )}

      {provider.models.length > 0 && (
        <ListView
          aria-label="Modelos"
          selectionMode="none"
          isQuiet
          UNSAFE_style={{ height: provider.models.length * ROW }}
        >
          {provider.models.map((m, i) => (
            <ListViewItem key={i} id={i} textValue={m.name || m.id}>
              <Text slot="label">
                {m.name || m.id}
                {i === provider.default_model && (
                  <span className={badge}>
                    <Badge variant="accent" fillStyle="subtle" size="S">
                      padrão
                    </Badge>
                  </span>
                )}
              </Text>
              <Text slot="description">
                id {m.id} · contexto {fmt(m.context_size)}
              </Text>
              <ActionButtonGroup>
                {i !== provider.default_model && (
                  <TooltipTrigger>
                    <ActionButton aria-label={`Tornar ${m.name || m.id} o padrão`} onPress={() => setDefault(i)}>
                      <Star />
                    </ActionButton>
                    <Tooltip>Tornar padrão</Tooltip>
                  </TooltipTrigger>
                )}
                <TooltipTrigger>
                  <ActionButton aria-label={`Editar ${m.name || m.id}`} onPress={() => startEdit(m, i)}>
                    <Edit />
                  </ActionButton>
                  <Tooltip>Editar</Tooltip>
                </TooltipTrigger>
                <TooltipTrigger>
                  <ActionButton aria-label={`Excluir ${m.name || m.id}`} onPress={() => remove(i)}>
                    <Delete />
                  </ActionButton>
                  <Tooltip>Excluir</Tooltip>
                </TooltipTrigger>
              </ActionButtonGroup>
            </ListViewItem>
          ))}
        </ListView>
      )}

      {draft && (
        <div className={form}>
          <h5 className={subtitle}>{creating ? 'Novo modelo' : 'Editar modelo'}</h5>

          <TextField
            label="ID do modelo (usado na API)"
            value={draft.id}
            onChange={(v) => set('id', v)}
            placeholder="google/gemma-4-12b-qat"
            styles={full}
          />
          <TextField
            label="Nome de exibição"
            value={draft.name}
            onChange={(v) => set('name', v)}
            placeholder="Gemma 4 12B"
            styles={full}
          />
          <NumberField
            label="Contexto máximo (tokens)"
            minValue={0}
            value={draft.context_size}
            onChange={(v) => set('context_size', Number.isNaN(v) ? 0 : v)}
            formatOptions={{ useGrouping: false }}
            styles={full}
          />

          <div className={actions}>
            <Button variant="secondary" onPress={cancel}>
              Cancelar
            </Button>
            <Button variant="accent" onPress={save} isDisabled={saving}>
              {saving ? 'Salvando...' : creating ? 'Criar modelo' : 'Salvar modelo'}
            </Button>
          </div>
        </div>
      )}
    </section>
  )
}

const section = style({
  display: 'flex',
  flexDirection: 'column',
  gap: 12,
  marginTop: 8,
  paddingTop: 16,
  borderWidth: 0,
  borderTopWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
})
const head = style({ display: 'flex', alignItems: 'center', justifyContent: 'space-between' })
const title = style({ margin: 0, font: 'title' })
const subtitle = style({ margin: 0, font: 'ui', fontWeight: 'bold' })
const hint = style({ margin: 0, font: 'ui-sm', color: 'neutral-subdued' })
const badge = style({ display: 'inline-block', marginStart: 8, marginY: -4, verticalAlign: 'middle' })
const form = style({ display: 'flex', flexDirection: 'column', gap: 12 })
const full = style({ width: 'full' })
const actions = style({ display: 'flex', justifyContent: 'end', gap: 12 })
