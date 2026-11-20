import { useState, type CSSProperties } from 'react'
import {
  ActionButton,
  Menu,
  MenuItem,
  MenuSection,
  MenuTrigger,
  ProgressCircle,
  Text,
} from '@react-spectrum/s2'
import { style } from '@react-spectrum/s2/style' with { type: 'macro' }
import MenuHamburger from '@react-spectrum/s2/icons/MenuHamburger'
import Folder from '@react-spectrum/s2/icons/Folder'
import FolderOpen from '@react-spectrum/s2/icons/FolderOpen'
import Settings from '@react-spectrum/s2/icons/Settings'
import FileText from '@react-spectrum/s2/icons/FileText'
import PainelLateral from '../icons/PainelLateral'
import WindowControls from './WindowControls'
import type { Connection, RecentWorkspace } from '../types'

// A titlebar da janela frameless (docs/FRONTEND.md, Shell e janela). Só o texto
// "Network Agent" e o espaçador do meio são `caption`: arrastam, e o duplo
// clique e o menu de sistema ficam com o Windows. Nenhum controle fica dentro
// de uma caption, porque lá ele não recebe clique.

const caption = { '--wails-non-client-region': 'caption' } as CSSProperties

const bar = style({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  height: 48,
  paddingX: 12,
  backgroundColor: 'layer-1',
  userSelect: 'none',
  minWidth: 0,
})

const brand = style({
  font: 'ui',
  fontWeight: 'bold',
  color: 'neutral',
  whiteSpace: 'nowrap',
  alignSelf: 'stretch',
  display: 'flex',
  alignItems: 'center',
})

const model = style({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  font: 'ui-sm',
  color: 'neutral-subdued',
  whiteSpace: 'nowrap',
})

const PICK = '::pick' // chave que nenhum caminho tem

/** O nome da pasta, do caminho do Windows ou POSIX. */
function folderName(path: string): string {
  return path.replace(/[\\/]+$/, '').split(/[\\/]/).pop() || path
}

interface TitlebarProps {
  sidebarExpanded: boolean
  onToggleSidebar: () => void
  onOpenLogs: () => void
  onOpenSettings: () => void
  workspace: string
  onPickWorkspace: () => void
  /** Os Workspaces recentes com a pasta conferida pelo Go, e o recarregar da lista. */
  recents: RecentWorkspace[]
  onOpenRecents: () => void
  /** Troca para um recente; false se o Go recusou (a pasta sumiu). */
  onPickRecent: (path: string) => Promise<boolean>
  /** Um turno roda: trocar de Workspace fica bloqueado. */
  workspaceLocked: boolean
  /** Provedor e modelo ativos, ou null sem modelo configurado. */
  active: { provider: string; model: string } | null
  connection: Connection | null
  onReconnect: () => void
}

export default function Titlebar({
  sidebarExpanded,
  onToggleSidebar,
  onOpenLogs,
  onOpenSettings,
  workspace,
  onPickWorkspace,
  recents,
  onOpenRecents,
  onPickRecent,
  workspaceLocked,
  active,
  connection,
  onReconnect,
}: TitlebarProps) {
  // O menu do chip é controlado: se a pasta some entre abrir e clicar, o Go
  // recusa e ele reabre, já com o item desabilitado.
  const [recentsOpen, setRecentsOpen] = useState(false)
  const openRecents = (open: boolean) => {
    setRecentsOpen(open)
    if (open) onOpenRecents()
  }
  const pickRecent = async (path: string) => {
    if (!(await onPickRecent(path))) openRecents(true)
  }
  const panelLabel = sidebarExpanded ? 'Recolher painel' : 'Expandir painel'
  return (
    <header className={bar}>
      <ActionButton isQuiet aria-label={panelLabel} onPress={onToggleSidebar}>
        <PainelLateral />
      </ActionButton>

      <MenuTrigger>
        <ActionButton isQuiet aria-label="Menu principal">
          <MenuHamburger />
        </ActionButton>
        <Menu onAction={(key) => (key === 'logs' ? onOpenLogs() : onOpenSettings())}>
          <MenuItem id="logs" textValue="Logs">
            <FileText />
            <Text slot="label">Logs</Text>
          </MenuItem>
          <MenuItem id="settings" textValue="Configurações">
            <Settings />
            <Text slot="label">Configurações</Text>
          </MenuItem>
        </Menu>
      </MenuTrigger>

      {/* A marca do protótipo: três nós ligados. */}
      <svg
        width="16"
        height="16"
        viewBox="0 0 16 16"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.3"
        aria-hidden="true"
        className={style({ color: 'neutral', flexShrink: 0 })}
      >
        <circle cx="8" cy="3.2" r="1.7" />
        <circle cx="3.2" cy="12" r="1.7" />
        <circle cx="12.8" cy="12" r="1.7" />
        <path d="M7 4.7 4 10.4M9 4.7l3 5.7M4.9 12h6.2" />
      </svg>
      <span className={brand} style={caption}>
        Network Agent
      </span>

      <span
        title={
          workspaceLocked
            ? 'Trocar de Workspace fica bloqueado enquanto o agente responde.'
            : workspace || 'Nenhuma pasta escolhida: o agente não grava arquivos.'
        }
      >
        <MenuTrigger isOpen={recentsOpen} onOpenChange={openRecents}>
          <ActionButton
            isQuiet
            isDisabled={workspaceLocked}
            aria-label={`Workspace em uso: ${workspace ? folderName(workspace) : 'nenhum'}`}
          >
            <Folder />
            <Text>{workspace ? folderName(workspace) : 'Nenhuma pasta'}</Text>
          </ActionButton>
          <Menu
            onAction={(key) => (key === PICK ? onPickWorkspace() : key !== workspace && pickRecent(String(key)))}
          >
            {recents.length > 0 && (
              <MenuSection selectionMode="single" selectedKeys={[workspace]} aria-label="Workspaces recentes">
                {recents.map((r) => (
                  <MenuItem key={r.path} id={r.path} textValue={r.path} isDisabled={!r.exists}>
                    <Text slot="label">{folderName(r.path)}</Text>
                    <Text slot="description">{r.exists ? r.path : 'Pasta não encontrada'}</Text>
                  </MenuItem>
                ))}
              </MenuSection>
            )}
            <MenuSection>
              <MenuItem id={PICK} textValue="Abrir outra pasta…">
                <FolderOpen />
                <Text slot="label">Abrir outra pasta…</Text>
              </MenuItem>
            </MenuSection>
          </Menu>
        </MenuTrigger>
      </span>

      <div className={style({ flexGrow: 1, alignSelf: 'stretch' })} style={caption} />

      <div className={model}>
        {active ? `${active.provider} · ${active.model}` : 'Nenhum modelo configurado'}
        {active && connection?.state === 'connecting' && (
          <ProgressCircle size="S" isIndeterminate aria-label="Carregando o modelo" />
        )}
        {active && connection?.state === 'failed' && (
          <span title={connection.cause}>
            <ActionButton size="S" onPress={onReconnect}>
              Reconectar
            </ActionButton>
          </span>
        )}
      </div>

      <WindowControls />
    </header>
  )
}
