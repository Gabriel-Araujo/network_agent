import { useState } from 'react'
import { GridList, GridListHeader, GridListItem, GridListSection } from 'react-aria-components'
import {
  ActionButton,
  AlertDialog,
  Button,
  Content,
  DialogContainer,
  Heading,
  InlineAlert,
  Text,
} from '@react-spectrum/s2'
import { style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Add from '@react-spectrum/s2/icons/Add'
import Chat from '@react-spectrum/s2/icons/Chat'
import Edit from '@react-spectrum/s2/icons/Edit'
import Delete from '@react-spectrum/s2/icons/Delete'
import { BridgeError, Conversation, groupConversations } from '../types'

// A sidebar e a faixa recolhida são um estado só (docs/FRONTEND.md, Shell e
// janela): expandida tem 244 px, recolhida vira a faixa de 56 px. A coluna anima
// a largura em 0,3 s, e o conteúdo troca na hora.

const column = style({
  width: { default: 244, isCollapsed: 56 },
  transition: '[width]',
  transitionDuration: 300,
  transitionTimingFunction: 'default',
  overflow: 'hidden',
  backgroundColor: 'base',
  display: 'flex',
  flexDirection: 'column',
  minHeight: 0,
})

const panel = style({
  width: 244,
  display: 'flex',
  flexDirection: 'column',
  minHeight: 0,
  flexGrow: 1,
  paddingTop: 12,
  gap: 8,
})

const scroll = style({ flexGrow: 1, minHeight: 0, overflowY: 'auto', paddingX: 8, paddingBottom: 8 })

const header = style({
  font: 'detail-sm',
  fontWeight: 'bold',
  textTransform: 'uppercase',
  color: 'neutral-subdued',
  paddingX: 8,
  paddingTop: 12,
  paddingBottom: 4,
})

const row = style({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  height: 36,
  paddingX: 8,
  borderRadius: 'lg',
  font: 'ui',
  color: { default: 'neutral-subdued', isActive: 'neutral', isHovered: 'neutral' },
  backgroundColor: {
    default: 'transparent',
    isHovered: 'gray-100',
    isActive: 'accent-subtle',
  },
  outlineStyle: { default: 'none', isFocusVisible: 'solid' },
  outlineColor: 'focus-ring',
  outlineWidth: 2,
  outlineOffset: -2,
  cursor: 'default',
  transition: 'colors',
  '--iconPrimary': { type: 'color', value: '[currentColor]' },
})

const title = style({ flexGrow: 1, minWidth: 0, truncate: true })

const actions = style({
  display: 'flex',
  opacity: { default: 0, isShown: 1 },
})

const input = style({
  flexGrow: 1,
  minWidth: 0,
  font: 'ui',
  color: 'neutral',
  backgroundColor: 'layer-1',
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: 'blue-800',
  borderRadius: 'default',
  paddingX: 8,
  height: 28,
  outlineStyle: 'none',
})

const empty = style({ margin: 0, font: 'ui-sm', color: 'neutral-subdued', paddingX: 8, paddingTop: 12 })

const stripe = style({
  width: 56,
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  paddingTop: 12,
})

interface SidebarProps {
  /** Expandida (244 px) ou faixa recolhida (56 px). */
  expanded: boolean
  conversations: Conversation[]
  /** Conversa aberta no chat, destacada na lista. */
  activeId: string | null
  /**
   * Um turno está em voo. Abrir/navegar NÃO bloqueia: o turno segue no Go e,
   * se a conversa não é mais a da tela, os tokens são descartados — o disco é
   * a fonte quando ela for reaberta. Renomear e apagar continuam bloqueados.
   */
  busy: boolean
  /** Falha da listagem ou de uma mutação, já interpretada da ponte. */
  error: BridgeError | null
  onNewChat: () => void
  onOpen: (c: Conversation) => void
  /** Devolve true quando o título foi gravado (o input fecha só nesse caso). */
  onRename: (id: string, title: string) => Promise<boolean>
  /** Chamado depois da confirmação do AlertDialog. */
  onDelete: (c: Conversation) => void
  onReload: () => void
  onDismissError: () => void
}

export default function Sidebar({
  expanded,
  conversations,
  activeId,
  busy,
  error,
  onNewChat,
  onOpen,
  onRename,
  onDelete,
  onReload,
  onDismissError,
}: SidebarProps) {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [draft, setDraft] = useState('')
  const [pendingDelete, setPendingDelete] = useState<Conversation | null>(null)
  const groups = groupConversations(conversations)
  const byId = new Map(conversations.map((c) => [c.id, c]))

  const startEdit = (c: Conversation) => {
    setEditingId(c.id)
    setDraft(c.title)
  }

  const cancelEdit = () => {
    setEditingId(null)
    setDraft('')
  }

  // Enter grava, Esc cancela, clicar fora cancela. Em falha o input fica aberto
  // com o texto digitado — o card de erro explica o motivo e nada se perde.
  const commit = async (c: Conversation) => {
    const t = draft.trim()
    if (!t || t === c.title) {
      cancelEdit()
      return
    }
    if (await onRename(c.id, t)) cancelEdit()
  }

  return (
    <div className={column({ isCollapsed: !expanded })}>
      {expanded ? (
        <aside className={panel} aria-label="Conversas">
          <div className={style({ paddingX: 12 })}>
            <Button variant="accent" onPress={onNewChat} styles={style({ width: 'full' })}>
              <Add />
              <Text>Nova conversa</Text>
            </Button>
          </div>

          <div className={scroll}>
            {error && (
              <InlineAlert variant="negative">
                <Heading>{error.what}</Heading>
                <Content>
                  {error.why && <p>{error.why}</p>}
                  {error.next && <p>{error.next}</p>}
                  <div className={style({ display: 'flex', gap: 8, marginTop: 8 })}>
                    <ActionButton size="S" onPress={onReload}>
                      Tentar de novo
                    </ActionButton>
                    <ActionButton size="S" isQuiet onPress={onDismissError}>
                      Dispensar
                    </ActionButton>
                  </div>
                </Content>
              </InlineAlert>
            )}

            {conversations.length === 0 && !error && (
              <p className={empty}>
                Nenhuma conversa ainda. Elas aparecem aqui e ficam salvas em disco.
              </p>
            )}

            {conversations.length > 0 && (
              <GridList
                aria-label="Conversas"
                // `tab`: as setas ficam para o input do renomear, e o Tab anda
                // entre a linha e as ações dela.
                keyboardNavigationBehavior="tab"
                onAction={(key) => {
                  const c = byId.get(String(key))
                  if (c && editingId !== c.id) onOpen(c)
                }}
              >
                {groups.map((g) => (
                  <GridListSection key={g.label} id={g.label}>
                    <GridListHeader className={header}>{g.label}</GridListHeader>
                    {g.items.map((c) => (
                      <GridListItem
                        key={c.id}
                        id={c.id}
                        textValue={c.title}
                        className={({ isHovered, isFocusVisible }) =>
                          row({ isHovered, isFocusVisible, isActive: c.id === activeId })
                        }
                      >
                        {({ isHovered, isFocusVisibleWithin }) =>
                          editingId === c.id ? (
                            <input
                              // Foca e seleciona o título ao entrar em edição. Um ref
                              // com efeito não serve: a coleção do RAC também monta
                              // o item numa cópia fora da tela.
                              autoFocus
                              onFocus={(e) => e.currentTarget.select()}
                              className={input}
                              value={draft}
                              onChange={(e) => setDraft(e.target.value)}
                              // A grade não pode ver as teclas nem o clique do
                              // input: seriam navegação, typeahead e abrir.
                              onPointerDown={(e) => e.stopPropagation()}
                              onKeyDown={(e) => {
                                e.stopPropagation()
                                if (e.key === 'Enter') {
                                  e.preventDefault()
                                  commit(c)
                                }
                                if (e.key === 'Escape') {
                                  e.preventDefault()
                                  cancelEdit()
                                }
                              }}
                              onBlur={cancelEdit}
                              aria-label={`Renomear ${c.title}`}
                            />
                          ) : (
                            <>
                              <Chat />
                              <span className={title} title={c.title}>
                                {c.title}
                              </span>
                              <div
                                className={actions({
                                  isShown: isHovered || isFocusVisibleWithin || c.id === activeId,
                                })}
                              >
                                <ActionButton
                                  isQuiet
                                  size="XS"
                                  isDisabled={busy}
                                  aria-label={`Renomear ${c.title}`}
                                  onPress={() => startEdit(c)}
                                >
                                  <Edit />
                                </ActionButton>
                                <ActionButton
                                  isQuiet
                                  size="XS"
                                  isDisabled={busy}
                                  aria-label={`Apagar ${c.title}`}
                                  onPress={() => setPendingDelete(c)}
                                >
                                  <Delete />
                                </ActionButton>
                              </div>
                            </>
                          )
                        }
                      </GridListItem>
                    ))}
                  </GridListSection>
                ))}
              </GridList>
            )}
          </div>
        </aside>
      ) : (
        <nav className={stripe} aria-label="Conversas">
          <Button variant="accent" aria-label="Nova conversa" onPress={onNewChat}>
            <Add />
          </Button>
        </nav>
      )}

      <DialogContainer onDismiss={() => setPendingDelete(null)}>
        {pendingDelete && (
          <AlertDialog
            variant="destructive"
            title="Apagar conversa"
            primaryActionLabel="Apagar"
            cancelLabel="Cancelar"
            autoFocusButton="cancel"
            onPrimaryAction={() => onDelete(pendingDelete)}
          >
            Apagar <strong>{pendingDelete.title}</strong> e o transcript dela do disco? Não dá para
            desfazer.
          </AlertDialog>
        )}
      </DialogContainer>
    </div>
  )
}
