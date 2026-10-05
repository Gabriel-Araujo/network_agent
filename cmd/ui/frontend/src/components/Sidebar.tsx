import { useEffect, useRef, useState } from 'react'
import {
  CaretLeft,
  ChatCircleDots,
  Command,
  GearSix,
  PencilSimple,
  Plus,
  Trash,
} from '@phosphor-icons/react'
import { BridgeError, Conversation, groupConversations } from '../types'

interface SidebarProps {
  open: boolean
  collapsed: boolean
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
  onClose: () => void
  onToggleCollapse: () => void
  onNewChat: () => void
  onOpenSettings: () => void
  onOpen: (c: Conversation) => void
  /** Devolve true quando o título foi gravado (o input fecha só nesse caso). */
  onRename: (id: string, title: string) => Promise<boolean>
  onDelete: (c: Conversation) => void
  onReload: () => void
  onDismissError: () => void
}

export default function Sidebar({
  open,
  collapsed,
  conversations,
  activeId,
  busy,
  error,
  onClose,
  onToggleCollapse,
  onNewChat,
  onOpenSettings,
  onOpen,
  onRename,
  onDelete,
  onReload,
  onDismissError,
}: SidebarProps) {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [draft, setDraft] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const groups = groupConversations(conversations)

  // Foca e seleciona o título ao entrar em edição. Depende só do id em edição:
  // um efeito que reexecutasse a cada render roubaria o foco do input.
  useEffect(() => {
    if (!editingId) return
    inputRef.current?.focus()
    inputRef.current?.select()
  }, [editingId])

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
    const title = draft.trim()
    if (!title || title === c.title) {
      cancelEdit()
      return
    }
    if (await onRename(c.id, title)) cancelEdit()
  }

  return (
    <>
      {open && <div className="sidebar-scrim" onClick={onClose} aria-hidden="true" />}
      <aside
        className={`sidebar ${open ? 'sidebar--open' : ''} ${collapsed ? 'sidebar--collapsed' : ''}`}
        aria-label="Histórico de conversas"
      >
        <div className="sidebar__brand-row">
          <span className="sidebar__brand">Network Agent</span>
          <button
            className="sidebar__logo"
            onClick={onToggleCollapse}
            aria-label={collapsed ? 'Expandir menu' : 'Recolher menu'}
            title={collapsed ? 'Expandir menu' : 'Recolher menu'}
          >
            <Command size={18} weight="duotone" aria-hidden="true" />
          </button>
          <button
            className="icon-btn sidebar__collapse"
            onClick={onToggleCollapse}
            aria-label="Recolher menu"
            title="Recolher menu"
          >
            <CaretLeft size={16} weight="bold" aria-hidden="true" />
          </button>
        </div>

        <div className="sidebar__header">
          <button className="btn-new" onClick={onNewChat}>
            <Plus size={16} weight="bold" aria-hidden="true" />
            Nova conversa
          </button>
          <button
            className="icon-btn sidebar__new-chat"
            onClick={onNewChat}
            aria-label="Nova conversa"
            title="Nova conversa"
          >
            <Plus size={16} weight="bold" aria-hidden="true" />
          </button>
        </div>

        <nav className="sidebar__list">
          {error && (
            <div className="sidebar__error" role="alert">
              <p className="sidebar__error-title">{error.what}</p>
              {error.why && <p className="sidebar__error-why">{error.why}</p>}
              {error.next && <p className="sidebar__error-next">{error.next}</p>}
              <div className="sidebar__error-actions">
                <button className="link-btn" onClick={onReload}>
                  Recarregar
                </button>
                <button className="link-btn link-btn--faint" onClick={onDismissError}>
                  Fechar
                </button>
              </div>
            </div>
          )}

          {conversations.length === 0 && !error && (
            <div className="sidebar__empty">
              <p className="sidebar__empty-title">Nenhuma conversa ainda</p>
              <p className="sidebar__empty-hint">
                Suas conversas aparecem aqui e ficam salvas em disco, para continuar de onde parou.
              </p>
            </div>
          )}

          {groups.map((g) => (
            <div className="sidebar__group" key={g.label}>
              <p className="sidebar__group-label">{g.label}</p>
              <ul className="sidebar__items">
                {g.items.map((c) => (
                  <li key={c.id}>
                    {editingId === c.id ? (
                      <input
                        ref={inputRef}
                        className="chat-item__input"
                        value={draft}
                        onChange={(e) => setDraft(e.target.value)}
                        onKeyDown={(e) => {
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
                      <div
                        className={`chat-item-row ${activeId === c.id ? 'chat-item-row--active' : ''}`}
                      >
                        <button
                          className="chat-item"
                          onClick={() => onOpen(c)}
                          title={c.title}
                        >
                          <ChatCircleDots
                            size={15}
                            weight="regular"
                            className="chat-item__icon"
                            aria-hidden="true"
                          />
                          <span className="chat-item__title">{c.title}</span>
                        </button>
                        <span className="chat-item__actions">
                          <button
                            className="icon-btn icon-btn--xs"
                            onClick={() => startEdit(c)}
                            disabled={busy}
                            aria-label={`Renomear ${c.title}`}
                            title="Renomear"
                          >
                            <PencilSimple size={13} weight="bold" aria-hidden="true" />
                          </button>
                          <button
                            className="icon-btn icon-btn--xs"
                            onClick={() => onDelete(c)}
                            disabled={busy}
                            aria-label={`Apagar ${c.title}`}
                            title="Apagar"
                          >
                            <Trash size={13} weight="bold" aria-hidden="true" />
                          </button>
                        </span>
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>

        <div className="sidebar__footer">
          <button className="chat-item chat-item--subtle" onClick={onOpenSettings}>
            <GearSix size={15} weight="regular" className="chat-item__icon" aria-hidden="true" />
            <span className="chat-item__title">Configurações</span>
          </button>
          <button
            className="icon-btn sidebar__settings-icon"
            onClick={onOpenSettings}
            aria-label="Configurações"
            title="Configurações"
          >
            <GearSix size={16} weight="regular" aria-hidden="true" />
          </button>
        </div>
      </aside>
    </>
  )
}
