import type { ContextMenuRegistration, ContextMenuTarget, MenuEntry } from '@numen/plugin-sdk'
import type { CommandDeps, CommandTarget, runSupport, SearchDeps } from '@/features/command-palette'
import type { FileOpeners, useWindowTabs } from '@/entities/tab'
import type { MessageLog } from '@/shared/notices/messages'
import type { NotePort } from '@/app/ports/notes'
import type { FilePort } from '@/app/ports/files'
import type { TabLinksState } from '@/features/tab-linking'
import type { createTabSync } from '../tabs/useTabSync'
import type { createNoteEditors } from '../tabs/useNoteEditors'
import type { useSettings } from '../window/useSettings'
import type { useVaults } from '../window/useVaults'
import type { useWindowDisplay } from '../window/useWindowDisplay'

export interface WindowKindsDeps {
  core: NotePort & FilePort & Pick<SearchDeps, 'names'>
  log: MessageLog
  tabOpeners: FileOpeners
  windowTabs: ReturnType<typeof useWindowTabs>
  runs: ReturnType<typeof runSupport>
  editing: ReturnType<typeof createNoteEditors>
  settings: ReturnType<typeof useSettings>
  vaults: ReturnType<typeof useVaults>
  window: ReturnType<typeof useWindowDisplay>
  getTarget: () => CommandTarget
  runCommand: (id: string, target: CommandTarget) => void
  commandDeps: () => CommandDeps
  tabLinks?: TabLinksState
  tabSync?: ReturnType<typeof createTabSync>
  getContextMenuItems?: (target: ContextMenuTarget) => readonly ContextMenuRegistration[]
  runContextMenu?: (id: string, target: ContextMenuTarget) => Promise<boolean>
  getContextMenuIcon?: (id: string) => unknown
  /** The icon a plugin draws its command with, and nothing for one it draws none for. */
  getPluginCommandIcon?: (id: string) => unknown
  /** The note put in front by the command a plugin declares for it; false where none declares one. */
  showNote: (path: string) => Promise<boolean>
  /** The commands the plugins declare over one note. */
  getNoteCommands: () => readonly MenuEntry[]
}
