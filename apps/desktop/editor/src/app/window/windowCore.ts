/**
 * What every part of the window is handed of the others: the message log, the settings store, the
 * tabs and the plugins, made before the parts that read and draw the vault.
 */
import type { createNoticeQueue, useWindowFrame } from '@numen/workbench'
import type { runSupport } from '@/features/command-palette'
import type {
  CommandItemLists,
  createPluginNotes,
  HostPort,
  useDeclaredCommands,
} from '@/features/plugin-commands'
import type { CommandRegistry } from '@/entities/command'
import type { SettingsStore } from '@/entities/settings'
import type { fileOpeners, WindowTabs } from '@/entities/tab'
import type { TabReports, useTabLinks } from '@/features/tab-linking'
import type { createMessageLog, MessageWriter } from '@/shared/model/notices/messages'
import type { createPluginSupport } from '../plugins/pluginSupport'

/** What the window is made of before its parts exist: the tabs, the plugins and the message log. */
export interface WindowCore {
  readonly frame: ReturnType<typeof useWindowFrame>
  readonly rest: SettingsStore
  readonly log: ReturnType<typeof createMessageLog>
  readonly writeMessage: MessageWriter
  readonly runs: ReturnType<typeof runSupport>
  readonly registry: CommandRegistry
  readonly tabs: TabsCore
  readonly plugins: PluginsCore
}

/** The tabs of the window, the links between them and the files they open. */
export interface TabsCore {
  readonly windowId: string
  readonly windowTabs: WindowTabs
  readonly tabLinks: ReturnType<typeof useTabLinks>
  readonly reports: TabReports
  readonly tabOpeners: ReturnType<typeof fileOpeners>
}

/** What the plugins of the window are loaded with and what they are given. */
export interface PluginsCore {
  readonly declaredCommands: ReturnType<typeof useDeclaredCommands>
  readonly itemLists: CommandItemLists
  readonly host: {
    readonly pluginHost: ReturnType<typeof createPluginSupport>['pluginHost']
    readonly pluginLoader: ReturnType<typeof createPluginSupport>['pluginLoader']
  }
  readonly hostPort: HostPort
  readonly pluginNotices: ReturnType<typeof createNoticeQueue>
  readonly pluginNotes: ReturnType<typeof createPluginNotes>
}
