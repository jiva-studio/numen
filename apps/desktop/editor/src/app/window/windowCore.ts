/**
 * What every part of the window is handed of the others: the message log, the settings store, the
 * tabs and the plugins, made before the parts that read and draw the vault.
 */
import type { PluginStatus } from '@numen/plugin-loader'
import type { createNoticeQueue, useWindowFrame } from '@numen/workbench'
import type {
  CommandItemLists,
  createPluginNotes,
  createPluginSupport,
  HostPort,
  PluginServices,
  useDeclaredCommands,
  WindowFactsSource,
} from '@/features/plugin-commands'
import type { CommandRegistry } from '@/shared/model/command'
import type { SettingsStore } from '@/entities/settings'
import type { fileOpeners, WindowTabs } from '@/entities/tab'
import type { TabReports, useTabLinks } from '@/features/tab-linking'
import type { createMessageLog, MessageWriter } from '@/shared/model/notices/messages'
import type { Emitter } from '@/shared/lib/emitter'

/** The core state of the window built before the vault services and the plugin host. */
export interface WindowFoundation {
  readonly frame: ReturnType<typeof useWindowFrame>
  readonly rest: SettingsStore
  readonly log: ReturnType<typeof createMessageLog>
  readonly writeMessage: MessageWriter
  readonly status: PluginStatus
  readonly registry: CommandRegistry
  readonly tabs: TabsCore
  readonly settingsRequests: Emitter<SettingsRequest>
}

/** What the window is made of before its commands exist: the tabs, the plugins and the message log. */
export interface WindowCore extends WindowFoundation {
  readonly plugins: PluginsCore
}

/** The settings page a request opens: a plugin, and a filter of its settings. */
export interface SettingsRequest {
  readonly pluginId: string | undefined
  readonly filter: string | undefined
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
  readonly facts: WindowFactsSource
  readonly services: PluginServices
  readonly pluginNotices: ReturnType<typeof createNoticeQueue>
  readonly pluginNotes: ReturnType<typeof createPluginNotes>
}
