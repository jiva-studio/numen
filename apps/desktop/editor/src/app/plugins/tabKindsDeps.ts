import type { WindowSettings } from '../window/createWindowServices'
import { type CommandDeps } from '@/entities/command'
import type { SearchDeps } from '@/features/command-palette'
import type { CommandTarget } from '@/entities/command'
import type { createTabSync, FileOpeners, WindowTabs } from '@/entities/tab'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { TabLinksState } from '@/features/tab-linking'
import type { createNoteEditors } from '../tabs/createNoteEditors'
import type { WindowDisplay } from '@/features/window-display'
import type { PluginNotes } from './pluginNotes'
import type { Component } from 'vue'

export interface WindowKindsDeps {
  core: Pick<SearchDeps, 'names'>
  log: MessageLog
  tabOpeners: FileOpeners
  windowTabs: WindowTabs
  editing: ReturnType<typeof createNoteEditors>
  pluginNotes: PluginNotes
  settings: WindowSettings
  window: WindowDisplay
  getTarget: () => CommandTarget
  runCommand: (id: string, target: CommandTarget) => void
  commandDeps: () => CommandDeps
  tabLinks?: TabLinksState
  tabSync?: ReturnType<typeof createTabSync>
  getPluginCommandIcon?: (id: string) => Component | undefined
}
