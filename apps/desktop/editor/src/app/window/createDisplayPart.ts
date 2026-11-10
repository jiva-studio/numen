/**
 * What every part of the window is handed of the others, and the parts that read and draw the
 * vault: the note editors, the display, the settings and the vaults.
 */
import { formatError } from '@/shared/api/errors'
import type { createNoticeQueue, useWindowFrame } from '@numen/workbench'
import type { runSupport } from '@/features/command-palette'
import type { Command, FeatureCommands } from '@/entities/command'
import { createTabSync, type fileOpeners, type WindowTabs } from '@/entities/tab'
import type { createMessageLog, MessageWriter } from '@/shared/notices/messages'
import { WORDS as words } from '@/shared/words'
import { syncTabPath, type useTabLinks } from '@/features/tab-linking'
import { createNoteEditors } from '../tabs/useNoteEditors'
import { useSettings } from './useSettings'
import { useVaults } from './useVaults'
import type { TabsHold } from '@/features/vault-commands'
import type { createCommands } from './createCommands'
import type { useDeclaredCommands } from '../plugins/commands/declaredCommands'
import type { CommandItemLists } from '../plugins/commands/commandItemLists'
import { useWindowDisplay } from '@/features/window-display'
import { createDisplayOptions } from './displayOptions'
import type { useTabKinds } from './useTabKinds'
import type { createWindowSlots } from '../plugins/windowSlots'
import type { createPluginNotes } from '../plugins/pluginNotes'
import type { PluginServices } from '../plugins/pluginServices'
import type { createPluginSupport } from '../plugins/useWindowPlugins'

/** What the window is made of before its parts exist: the tabs, the plugins and the message log. */
export interface WindowCore {
  readonly frame: ReturnType<typeof useWindowFrame>
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

export interface WindowContext {
  commands?: ReturnType<typeof createCommands>
  kinds?: ReturnType<typeof useTabKinds>
  vaults?: ReturnType<typeof useVaults>
  tabsHold?: TabsHold
  featureCommands?: FeatureCommands
}

export function getCommands(ctx: WindowContext): NonNullable<WindowContext['commands']> {
  // Stryker disable next-line ConditionalExpression: definite-assignment guard for cyclic window wiring
  if (!ctx.commands) throw new Error('window commands are read before they are created')
  return ctx.commands
}

export function getKinds(ctx: WindowContext): NonNullable<WindowContext['kinds']> {
  // Stryker disable next-line ConditionalExpression: definite-assignment guard for cyclic window wiring
  if (!ctx.kinds) throw new Error('window kinds are read before they are created')
  return ctx.kinds
}

export function getTabsHold(ctx: WindowContext): TabsHold {
  // Stryker disable next-line ConditionalExpression: definite-assignment guard for cyclic window wiring
  if (!ctx.tabsHold) throw new Error('window tabs hold is read before it is created')
  return ctx.tabsHold
}

export function getVaults(ctx: WindowContext): NonNullable<WindowContext['vaults']> {
  // Stryker disable next-line ConditionalExpression: definite-assignment guard for cyclic window wiring
  if (!ctx.vaults) throw new Error('window vaults are read before they are created')
  return ctx.vaults
}

export function initWindowServices(
  base: WindowCore,
  ctx: WindowContext,
  settingsHolder: { current?: ReturnType<typeof useSettings> },
) {
  const editing = createNoteEditors({
    log: base.log,
    tabOpeners: base.tabs.tabOpeners,
    windowTabs: base.tabs.windowTabs,
  })

  const tabSync = createTabSync({
    windowTabs: base.tabs.windowTabs,
    tabLinks: base.tabs.tabLinks,
    sendSync: syncTabPath,
    onError: (error) => base.writeMessage(formatError(error), 'error'),
  })

  const window = useWindowDisplay(
    createDisplayOptions({
      editing,
      getCommands: () => getCommands(ctx),
      getKinds: () => getKinds(ctx),
      tabOpeners: base.tabs.tabOpeners,
      windowTabs: base.tabs.windowTabs,
      getVaults: () => getVaults(ctx),
      notifyPlugins: (event) => base.plugins.host.pluginHost.notifyVaultChange(event),
      writeMessage: base.writeMessage,
    }),
  )

  const settings = useSettings({
    words,
    log: base.log,
    windowTabs: base.tabs.windowTabs,
    onSizeChanged: () => editing.noted.measureAll(),
    getSettingsViews: (pluginId) => base.plugins.host.pluginHost.getRegisteredSettingsViews(pluginId),
    getPlatform: () => base.frame.platform.value,
  })
  settingsHolder.current = settings

  ctx.vaults = useVaults({
    log: base.log,
    chunks: window.vault.chunks,
    embedded: window.vault.embedded,
    isEmbedding: window.vault.isEmbedding,
  })

  return { editing, tabSync, window, settings }
}

export type WindowServices = ReturnType<typeof initWindowServices>
