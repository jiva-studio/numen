import type { CommandDeps, SearchDeps } from '@/features/command-palette'
import type { CommandTarget } from '@/entities/command'
import type { createTabSync, FileOpeners, WindowTabs } from '@/entities/tab'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { TabLinksState } from '@/features/tab-linking'
import type { createNoteEditors } from '../tabs/useNoteEditors'
import type { useSettings } from '../window/useSettings'
import type { WindowDisplay } from '@/features/window-display'
import type { PluginNotes } from './pluginNotes'

export interface WindowKindsDeps {
  core: Pick<SearchDeps, 'names'>
  log: MessageLog
  tabOpeners: FileOpeners
  windowTabs: WindowTabs
  editing: ReturnType<typeof createNoteEditors>
  pluginNotes: PluginNotes
  settings: ReturnType<typeof useSettings>
  window: WindowDisplay
  getTarget: () => CommandTarget
  runCommand: (id: string, target: CommandTarget) => void
  commandDeps: () => CommandDeps
  tabLinks?: TabLinksState
  tabSync?: ReturnType<typeof createTabSync>
  getPluginCommandIcon?: (id: string) => unknown
}
