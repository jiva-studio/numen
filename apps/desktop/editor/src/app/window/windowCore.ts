/**
 * What every part of the window is handed of the others: the message log, the settings store, the
 * tabs and the plugins, made before the parts that read and draw the vault.
 */
import type { createNoticeQueue, useWindowFrame } from '@numen/workbench'
import type { runSupport } from '@/features/command-palette'
import type { useDeclaredCommands, CommandItemLists } from '@/features/plugin-commands'
import type { Command } from '@/entities/command'
import type { SettingsStore } from '@/entities/settings'
import type { fileOpeners, WindowTabs } from '@/entities/tab'
import type { useTabLinks } from '@/features/tab-linking'
import type { createMessageLog, MessageWriter } from '@/shared/notices/messages'
import type { createWindowSlots } from '../plugins/windowSlots'
import type { createPluginNotes } from '../plugins/pluginNotes'
import type { PluginServices } from '../plugins/pluginServices'
import type { createPluginSupport } from '../plugins/useWindowPlugins'

/** What the window is made of before its parts exist: the tabs, the plugins and the message log. */
export interface WindowCore {
  readonly frame: ReturnType<typeof useWindowFrame>
  readonly rest: SettingsStore
  readonly log: ReturnType<typeof createMessageLog>
  readonly writeMessage: MessageWriter
  readonly runs: ReturnType<typeof runSupport>
  readonly tabs: TabsCore
  readonly plugins: PluginsCore
}

/** The tabs of the window, the links between them and the files they open. */
export interface TabsCore {
  readonly windowId: string
  readonly windowTabs: WindowTabs
  readonly tabLinks: ReturnType<typeof useTabLinks>
  readonly tabReports: { wait: () => Promise<void> }
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
  readonly pluginServices: PluginServices
  readonly pluginNotices: ReturnType<typeof createNoticeQueue>
  readonly pluginNotes: ReturnType<typeof createPluginNotes>
  readonly slotWindowValues: ReturnType<typeof createWindowSlots>['windowValues']
  readonly appCommands: { current?: () => readonly Command[] }
}
