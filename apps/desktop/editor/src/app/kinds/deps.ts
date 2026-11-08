import type { ContextMenuRegistration, ContextMenuTarget, MenuEntry } from '@numen/plugin-host'
import type {
  CommandDeps,
  CommandTarget,
  PathContribution,
  SearchDeps,
} from '@/features/command-palette'
import type { FileOpeners, useWindowTabs } from '@/entities/tab'
import type { MessageLog } from '@/shared/notices/messages'
import type { NotePort } from '@/app/ports/notes'
import type { FilePort } from '@/app/ports/files'
import type { TabLinksState } from '@/features/tab-linking'
import type { createTabSync } from '../tabs/useTabSync'
import type { createNoteEditors } from '../tabs/useNoteEditors'
import type { useSettings } from '../window/useSettings'
import type { useWindowDisplay } from '../window/useWindowDisplay'
import type { PluginNotes } from '../plugins/pluginNotes'

export interface WindowKindsDeps {
  core: NotePort & FilePort & Pick<SearchDeps, 'names'>
  log: MessageLog
  tabOpeners: FileOpeners
  windowTabs: ReturnType<typeof useWindowTabs>
  editing: ReturnType<typeof createNoteEditors>
  pluginNotes: PluginNotes
  settings: ReturnType<typeof useSettings>
  window: ReturnType<typeof useWindowDisplay>
  getTarget: () => CommandTarget
  runCommand: (id: string, target: CommandTarget) => void
  commandDeps: () => CommandDeps
  tabLinks?: TabLinksState
  tabSync?: ReturnType<typeof createTabSync>
  getContextMenuItems?: (target: ContextMenuTarget) => readonly ContextMenuRegistration[]
  runContextMenu?: (id: string, target: ContextMenuTarget) => Promise<boolean>
  getFileMenuItems?: (path: string) => readonly PathContribution[]
  getContextMenuIcon?: (id: string) => unknown
  getPluginCommandIcon?: (id: string) => unknown
  getNoteCommands: () => readonly MenuEntry[]
}
