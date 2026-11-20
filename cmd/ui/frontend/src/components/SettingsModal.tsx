import { useCallback, useEffect, useRef, useState } from 'react'
import {
  Button,
  Content,
  Dialog,
  DialogContainer,
  Heading,
  InlineAlert,
  ListView,
  ListViewItem,
  NumberField,
  Provider,
  SelectBox,
  SelectBoxGroup,
  Tab,
  TabList,
  TabPanel,
  Tabs,
  Text,
  TextField,
  Tooltip,
  TooltipTrigger,
  ActionButton,
} from '@react-spectrum/s2'
import { css, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Add from '@react-spectrum/s2/icons/Add'
import BrightnessContrast from '@react-spectrum/s2/icons/BrightnessContrast'
import Delete from '@react-spectrum/s2/icons/Delete'
import Folder from '@react-spectrum/s2/icons/Folder'
import ListBulleted from '@react-spectrum/s2/icons/ListBulleted'
import { AppConfig, EMPTY_PROVIDER, ProviderConfig } from '../types'
import ModelsSection from './ModelsSection'
import { App } from '../../bindings/github.com/Gabriel-Araujo/network_agent/cmd/ui'
import { errorText, inWails } from '../types'
import { Theme, setTheme, useTheme } from '../theme'

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

type Section = 'workspace' | 'providers' | 'appearance'

// Abaixo de 1080 px o protótipo vira a navegação em linha e empilha a lista de provedores
// sobre o formulário (docs/FRONTEND.md, Configurações).
const NARROW = '(max-width: 1079px)'
const useNarrow = () => {
  const [narrow, setNarrow] = useState(() => window.matchMedia(NARROW).matches)
  useEffect(() => {
    const mq = window.matchMedia(NARROW)
    const on = () => setNarrow(mq.matches)
    mq.addEventListener('change', on)
    return () => mq.removeEventListener('change', on)
  }, [])
  return narrow
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
  const [section, setSection] = useState<Section>('workspace')
  const narrow = useNarrow()
  const theme = useTheme()

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

  // Carrega a configuração uma única vez por abertura do modal. Sem provedor
  // configurado (não dá para fechar), abre direto na aba Provedores.
  const dismissibleRef = useRef(dismissible)
  dismissibleRef.current = dismissible
  useEffect(() => {
    if (open) {
      load()
      setError('')
      setSection(dismissibleRef.current ? 'workspace' : 'providers')
    }
  }, [open, load])

  // O draft é definido explicitamente por quem muda de contexto (selectProvider,
  // startCreate, cancelCreate). Não há efeito de sincronização: ele rodava após
  // o commit e, com selectedIdx = null durante a criação, apagava o formulário.

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

  const providerKey = creating ? 'new' : selectedIdx

  return (
    <DialogContainer onDismiss={onClose}>
      {open && (
        <Dialog size="XL" isDismissible={dismissible} isKeyboardDismissDisabled={!dismissible}>
          <Heading slot="title">Configurações</Heading>
          <Content>
            {!dismissible && (
              <InlineAlert variant="informative" styles={alert}>
                <Heading>Configuração necessária</Heading>
                <Content>
                  Cadastre um provedor com ao menos um modelo para começar a usar o agente.
                </Content>
              </InlineAlert>
            )}
            {error && (
              <InlineAlert variant="negative" styles={alert}>
                <Heading>Não foi possível concluir</Heading>
                <Content>{error}</Content>
              </InlineAlert>
            )}

            <Tabs
              aria-label="Seções de configuração"
              orientation={narrow ? 'horizontal' : 'vertical'}
              selectedKey={section}
              onSelectionChange={(k) => setSection(k as Section)}
              styles={tabs}
            >
              <TabList>
                <Tab id="workspace">
                  <Folder />
                  <Text>Workspace</Text>
                </Tab>
                <Tab id="providers">
                  <ListBulleted />
                  <Text>Provedores</Text>
                </Tab>
                <Tab id="appearance">
                  <BrightnessContrast />
                  <Text>Aparência</Text>
                </Tab>
              </TabList>

              {/* O Workspace é do app, não do provedor (ticket 05, §4): tem aba própria. */}
              <TabPanel id="workspace" styles={panel}>
                <div className={pane}>
                  <div className={row}>
                    <TextField
                      label="Workspace"
                      isReadOnly
                      value={config.workspace || ''}
                      placeholder="Nenhuma pasta escolhida — o agente não grava arquivos."
                      styles={grow}
                    />
                    <Button variant="secondary" onPress={pickWorkspace} isDisabled={workspaceLocked}>
                      {config.workspace ? 'Trocar...' : 'Escolher...'}
                    </Button>
                  </div>
                  <p className={hint}>
                    A única pasta onde o agente grava.
                    {workspaceLocked && ' Trocar de Workspace fica bloqueado enquanto o agente responde.'}
                  </p>
                </div>
              </TabPanel>

              <TabPanel id="providers" styles={panel}>
                <div className={split}>
                  <div className={column}>
                    <div className={listHead}>
                      <span className={colLabel}>Provedores</span>
                      <TooltipTrigger>
                        <ActionButton isQuiet size="S" aria-label="Novo provedor" onPress={startCreate}>
                          <Add />
                        </ActionButton>
                        <Tooltip>Novo provedor</Tooltip>
                      </TooltipTrigger>
                    </div>
                    <ListView
                      aria-label="Provedores"
                      selectionMode="single"
                      selectionStyle="highlight"
                      disallowEmptySelection
                      selectedKeys={providerKey === null ? [] : [providerKey]}
                      onSelectionChange={(keys) => {
                        const k = keys === 'all' ? null : [...keys][0]
                        if (typeof k === 'number') selectProvider(k)
                      }}
                      renderEmptyState={() => <p className={hint}>Nenhum provedor cadastrado.</p>}
                      styles={providerList}
                    >
                      {[
                        ...config.providers.map((p, i) => (
                          <ListViewItem key={i} id={i} textValue={p.name}>
                            <Text slot="label">{p.name}</Text>
                            <Text slot="description">
                              {p.models.length} {p.models.length === 1 ? 'modelo' : 'modelos'}
                              {i === config.selected ? ' · em uso' : ''}
                            </Text>
                          </ListViewItem>
                        )),
                        ...(creating
                          ? [
                              <ListViewItem key="new" id="new" textValue="Novo provedor">
                                <Text slot="label">Novo provedor</Text>
                              </ListViewItem>,
                            ]
                          : []),
                      ]}
                    </ListView>
                  </div>

                  <div className={`${column} ${scroll}`}>
                    {!draft ? (
                      <p className={hint}>Selecione um provedor à esquerda ou crie um novo.</p>
                    ) : (
                      <>
                        <div className={editHead}>
                          <h3 className={editTitle}>{creating ? 'Novo provedor' : 'Editar provedor'}</h3>
                          {!creating && selectedIdx != null && (
                            <div className={row}>
                              {selectedIdx !== config.selected && (
                                <Button variant="secondary" size="S" onPress={() => useProvider(selectedIdx)}>
                                  Usar este
                                </Button>
                              )}
                              <Button
                                variant="negative"
                                fillStyle="outline"
                                size="S"
                                onPress={() => removeProvider(selectedIdx)}
                              >
                                <Delete />
                                <Text>Excluir</Text>
                              </Button>
                            </div>
                          )}
                        </div>

                        <TextField
                          label="Nome"
                          value={draft.name}
                          onChange={(v) => set('name', v)}
                          placeholder="OpenAI Compatible"
                          styles={full}
                        />
                        <TextField
                          label="URL base"
                          value={draft.url}
                          onChange={(v) => set('url', v)}
                          placeholder="http://localhost:1234/v1"
                          styles={full}
                        />
                        <TextField
                          label="Chave de API"
                          type="password"
                          value={draft.api_key}
                          onChange={(v) => set('api_key', v)}
                          placeholder="deixe vazio para provedores locais"
                          styles={full}
                        />

                        <div className={actions}>
                          {creating && (
                            <Button variant="secondary" onPress={cancelCreate}>
                              Cancelar
                            </Button>
                          )}
                          <Button variant="accent" onPress={saveProvider} isDisabled={saving}>
                            {saving ? 'Salvando...' : creating ? 'Criar provedor' : 'Salvar alterações'}
                          </Button>
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
                          <p className={hint}>Crie o provedor para poder cadastrar os modelos dele.</p>
                        )}
                      </>
                    )}
                  </div>
                </div>
              </TabPanel>

              <TabPanel id="appearance" styles={panel}>
                <div className={pane}>
                  <h3 className={editTitle} id="theme-label">
                    Tema
                  </h3>
                  <SelectBoxGroup
                    aria-labelledby="theme-label"
                    selectedKeys={[theme]}
                    disallowEmptySelection
                    onSelectionChange={(keys) => {
                      const k = keys === 'all' ? undefined : [...keys][0]
                      if (k === 'light' || k === 'dark' || k === 'system') setTheme(k as Theme)
                    }}
                  >
                    <SelectBox id="light" textValue="Claro">
                      <ThemePreview mode="light" />
                      <Text slot="label">Claro</Text>
                    </SelectBox>
                    <SelectBox id="dark" textValue="Escuro">
                      <ThemePreview mode="dark" />
                      <Text slot="label">Escuro</Text>
                    </SelectBox>
                    <SelectBox id="system" textValue="Sistema">
                      <ThemePreview mode="system" />
                      <Text slot="label">Sistema</Text>
                    </SelectBox>
                  </SelectBoxGroup>
                  <p className={hint}>O tema "Sistema" acompanha a preferência do sistema operacional.</p>
                </div>
              </TabPanel>
            </Tabs>
          </Content>
        </Dialog>
      )}
    </DialogContainer>
  )
}

// A prévia de um tema: uma janela em miniatura (sidebar e duas linhas). Cada metade usa um
// `Provider` aninhado com o esquema dela, então as cores são as do S2 nos dois temas, não
// hex soltos, e valem qualquer que seja o tema em uso. O Sistema parte uma de cada.
function ThemePreview({ mode }: { mode: Theme }) {
  return (
    <div className={`${preview} ${previewSlot}`} aria-hidden="true">
      {mode === 'system' ? (
        <>
          <div className={`${half} ${halfStart}`}>
            <Mini scheme="light" />
          </div>
          <div className={`${half} ${halfEnd}`}>
            <div className={shift}>
              <Mini scheme="dark" />
            </div>
          </div>
        </>
      ) : (
        <Mini scheme={mode} />
      )}
    </div>
  )
}

function Mini({ scheme }: { scheme: 'light' | 'dark' }) {
  return (
    <Provider colorScheme={scheme} background="base" styles={mini}>
      <div className={miniSide} />
      <div className={miniBody}>
        <div className={miniBar} />
        <div className={miniBar} />
      </div>
    </Provider>
  )
}

const alert = style({ width: 'full', marginBottom: 12 })

// O corpo do modal: a altura segue a janela (a gaveta de rolagem é de cada painel).
const tabs = style({ height: '[min(480px, calc(100vh - 260px))]', minHeight: 0, width: 'full' })
const panel = style({ minWidth: 0, minHeight: 0, flexGrow: 1 })
const pane = style({
  display: 'flex',
  flexDirection: 'column',
  gap: 12,
  height: 'full',
  overflowY: 'auto',
  paddingX: 4,
})
const row = style({ display: 'flex', alignItems: 'end', gap: 12 })
const grow = style({ flexGrow: 1, minWidth: 0 })
const full = style({ width: 'full' })
const hint = style({ margin: 0, font: 'ui-sm', color: 'neutral-subdued' })

// Provedores: a lista (220 px) ao lado do formulário; abaixo de 1080 px, empilhados.
const split = style({
  display: 'grid',
  gridTemplateColumns: {
    default: [220, 'minmax(0, 1fr)'],
    '@media (max-width: 1079px)': 'minmax(0, 1fr)',
  },
  gridTemplateRows: {
    default: 'minmax(0, 1fr)',
    '@media (max-width: 1079px)': ['minmax(0, 40%)', 'minmax(0, 1fr)'],
  },
  columnGap: 16,
  rowGap: 12,
  height: 'full',
  minHeight: 0,
})
const column = style({ display: 'flex', flexDirection: 'column', gap: 12, minWidth: 0, minHeight: 0 })
const scroll = style({ overflowY: 'auto', paddingX: 4 })
const listHead = style({ display: 'flex', alignItems: 'center', justifyContent: 'space-between' })
const colLabel = style({
  font: 'ui-sm',
  fontWeight: 'bold',
  color: 'neutral-subdued',
  textTransform: 'uppercase',
})
const providerList = style({ flexGrow: 1, minHeight: 0 })
const editHead = style({ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 })
const editTitle = style({ margin: 0, font: 'title' })
const actions = style({ display: 'flex', justifyContent: 'end', gap: 12 })

const preview = style({
  position: 'relative',
  overflow: 'hidden',
  width: 96,
  height: 62,
  borderRadius: 'default',
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-400',
})
// O S2 só posiciona na área `illustration` o que passa pelo `IllustrationContext`.
const previewSlot = css(`grid-area: illustration; justify-self: center; align-self: center;`)
const half = style({ position: 'absolute', top: 0, bottom: 0, width: '[50%]', overflow: 'hidden' })
const halfStart = style({ insetStart: 0 })
const halfEnd = style({ insetEnd: 0 })
const shift = style({ marginStart: -48 })
const mini = style({ display: 'flex', width: 96, height: 62 })
const miniSide = style({ width: '[34%]', backgroundColor: 'gray-100' })
const miniBody = style({
  flexGrow: 1,
  display: 'flex',
  flexDirection: 'column',
  gap: 8,
  paddingStart: 12,
  paddingEnd: 8,
  paddingTop: 16,
})
const miniBar = style({ height: 9, borderRadius: 'sm', backgroundColor: 'gray-200' })
