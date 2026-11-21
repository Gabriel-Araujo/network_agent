import { useState } from 'react'
import { Button as RacButton, GridList, GridListItem } from 'react-aria-components'
import {
  Accordion,
  ActionButton,
  AlertDialog,
  Content,
  DialogContainer,
  Disclosure,
  DisclosurePanel,
  DisclosureTitle,
  Heading,
  InlineAlert,
} from '@react-spectrum/s2'
import { css, focusRing, lightDark, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Add from '@react-spectrum/s2/icons/Add'
import Chat from '@react-spectrum/s2/icons/Chat'
import Edit from '@react-spectrum/s2/icons/Edit'
import Delete from '@react-spectrum/s2/icons/Delete'
import { BridgeError, Conversation, GROUP_ORDER, groupConversations } from '../types'

// A sidebar e a faixa recolhida são um estado só (docs/FRONTEND.md, Shell e
// janela): uma árvore só, que a coluna recorta. A coluna anima a largura (244 px
// <-> 56 px, 0,3 s, curva `default`), o botão "Nova conversa" encolhe junto até
// ficar só o ícone, e a lista de Conversas some e volta com fade. O painel tem
// sempre 244 px; ao recolher, o que sobra fora da coluna já está invisível.

const column = style({
  width: { default: 244, isCollapsed: 56 },
  transition: '[width]',
  transitionDuration: 300,
  transitionTimingFunction: 'default',
  overflow: 'hidden',
  display: 'flex',
  flexDirection: 'column',
  minHeight: 0,
})

const panel = style({
  width: 244,
  flexShrink: 0,
  display: 'flex',
  flexDirection: 'column',
  minHeight: 0,
  flexGrow: 1,
  paddingTop: 12,
  gap: 8,
})

// Ao recolher, o fade (150 ms, curva `in`) termina antes de a coluna recortar o
// texto, e o `visibility` só vira depois dele. Ao expandir, a coluna abre e a
// lista entra depois (atraso de 100 ms, 200 ms, curva `out`), como o SideNavPanel.
const fade = style({
  opacity: { default: 1, isCollapsed: 0 },
  visibility: { default: 'visible', isCollapsed: 'hidden' },
  transition: '[opacity, visibility]',
  transitionDuration: { default: 200, isCollapsed: 150 },
  transitionDelay: { default: 100, isCollapsed: 0 },
  transitionTimingFunction: { default: 'out', isCollapsed: 'in' },
})

// O "Nova conversa" é um botão só que muda de forma: 220 px com ícone e rótulo,
// ou 32 px só com o ícone. A largura, o respiro e o vão anulam-se juntos, e o
// rótulo encolhe (a largura) e some (o fade, mais curto que a largura). Mesmas cores do `Button variant="accent"` do S2.
const newChat = style({
  ...focusRing(),
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  flexShrink: 0,
  boxSizing: 'border-box',
  height: 32,
  width: { default: 220, isCollapsed: 32 },
  paddingX: { default: 16, isCollapsed: 0 },
  columnGap: { default: 8, isCollapsed: 0 },
  marginX: 12,
  borderWidth: 0,
  borderRadius: 'pill',
  font: 'ui',
  fontWeight: 'bold',
  color: 'white',
  backgroundColor: {
    default: lightDark('accent-900', 'accent-700'),
    isHovered: lightDark('accent-1000', 'accent-600'),
    isPressed: lightDark('accent-1000', 'accent-600'),
    isFocusVisible: lightDark('accent-1000', 'accent-600'),
  },
  cursor: 'default',
  userSelect: 'none',
  transition: '[width, padding-inline, column-gap, background-color]',
  transitionDuration: 300,
  transitionTimingFunction: 'default',
  '--iconPrimary': { type: 'color', value: '[currentColor]' },
})

const newChatLabel = style({
  overflow: 'hidden',
  whiteSpace: 'nowrap',
  maxWidth: { default: 160, isCollapsed: 0 },
  transition: '[max-width]',
  transitionDuration: 300,
  transitionTimingFunction: 'default',
})

const scroll = style({ flexGrow: 1, minHeight: 0, overflowY: 'auto', paddingBottom: 8 })

// O título da seção é o do Disclosure do S2, com o `font` que ele deixa trocar;
// a caixa alta vai no `h3` e o botão a herda (o navegador põe `none` nele).
const sectionTitle = css(`text-transform: uppercase; & button { text-transform: inherit; }`)
const sectionTitleFont = style({ font: 'detail-sm', fontWeight: 'bold' })

// O painel do Disclosure põe 8 px em cima e 16 embaixo em volta da lista; a
// margem negativa devolve parte disso e deixa as seções mais juntas.
const sectionList = style({ marginTop: -4, marginBottom: -8 })

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

const empty = style({ margin: 0, font: 'ui-sm', color: 'neutral-subdued', paddingX: 16, paddingTop: 12 })

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

  const collapsed = !expanded

  return (
    <div className={column({ isCollapsed: collapsed })}>
      <aside className={panel} aria-label="Conversas">
        {/* Um botão só: a pílula larga expandida, o ícone sozinho na faixa. */}
        <RacButton
          aria-label="Nova conversa"
          onPress={onNewChat}
          className={({ isHovered, isPressed, isFocusVisible }) =>
            newChat({ isCollapsed: collapsed, isHovered, isPressed, isFocusVisible })
          }
        >
          <Add />
          <span className={newChatLabel({ isCollapsed: collapsed })}>
            {/* O fade do texto é o mesmo da lista: some antes de o botão cortá-lo. */}
            <span className={fade({ isCollapsed: collapsed })}>Nova conversa</span>
          </span>
        </RacButton>

        {/* `inert` na faixa: o que está invisível também sai da ordem de Tab. */}
        <div className={`${scroll} ${fade({ isCollapsed: collapsed })}`} inert={collapsed}>
          {error && (
            <div className={style({ paddingX: 8 })}>
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
            </div>
          )}

          {conversations.length === 0 && !error && (
            <p className={empty}>
              Nenhuma conversa ainda. Elas aparecem aqui e ficam salvas em disco.
            </p>
          )}

          {conversations.length > 0 && (
            // Uma seção por grupo de data, todas abertas ao montar e várias ao
            // mesmo tempo. Um grupo que aparece depois (a primeira Conversa de
            // Hoje) já nasce aberto, porque a chave dele está na lista inicial.
            <Accordion allowsMultipleExpanded defaultExpandedKeys={GROUP_ORDER}>
              {groups.map((g) => (
                <Disclosure key={g.label} id={g.label} isQuiet size="S">
                  <DisclosureTitle
                    UNSAFE_className={sectionTitle}
                    styles={sectionTitleFont}
                  >
                    {g.label}
                  </DisclosureTitle>
                  <DisclosurePanel>
                    {/* `tab`: as setas ficam para o input do renomear, e o Tab anda
                        entre a linha e as ações dela. Cada seção é uma lista, então
                        as setas não passam de uma seção para a outra. */}
                    <GridList
                      aria-label={g.label}
                      className={sectionList}
                      keyboardNavigationBehavior="tab"
                      onAction={(key) => {
                        const c = byId.get(String(key))
                        if (c && editingId !== c.id) onOpen(c)
                      }}
                    >
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
                    </GridList>
                  </DisclosurePanel>
                </Disclosure>
              ))}
            </Accordion>
          )}
        </div>
      </aside>

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
