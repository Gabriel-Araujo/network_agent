import {
  CaretLeft,
  ChatCircleDots,
  Command,
  GearSix,
  Plus,
} from '@phosphor-icons/react'
import { ChatMeta } from '../types'

interface SidebarProps {
  open: boolean
  collapsed: boolean
  chats: ChatMeta[]
  onClose: () => void
  onToggleCollapse: () => void
  onNewChat: () => void
  onOpenSettings: () => void
}

export default function Sidebar({
  open,
  collapsed,
  chats,
  onClose,
  onToggleCollapse,
  onNewChat,
  onOpenSettings,
}: SidebarProps) {
  const groups = [...new Set(chats.map((c) => c.group))]

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
          {groups.map((g) => (
            <div className="sidebar__group" key={g}>
              <p className="sidebar__group-label">{g}</p>
              <ul className="sidebar__items">
                {chats
                  .filter((c) => c.group === g)
                  .map((c) => (
                    <li key={c.id}>
                      <button className="chat-item">
                        <ChatCircleDots size={15} weight="regular" className="chat-item__icon" aria-hidden="true" />
                        <span className="chat-item__title">{c.title}</span>
                      </button>
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