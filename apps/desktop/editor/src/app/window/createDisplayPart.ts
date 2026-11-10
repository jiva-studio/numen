/**
 * What every part of the window is handed of the others, and the parts that read and draw the
 * vault: the note editors, the display, the settings and the vaults.
 */
import { formatError } from '@/shared/api/errors'
import type { createNoticeQueue, useWindowFrame } from '@numen/workbench'
import { pluginCatalog } from '@/shared/api/clients'
import { core } from '../vault'
import type { runSupport } from '@/features/command-palette'
import type { Command } from '@/entities/command'
import type { fileOpeners, useWindowTabs } from '@/entities/tab'
import type { createMessageLog, MessageWriter } from '@/shared/notices/messages'
import { WORDS as words } from '@/shared/words'
import type { useTabLinks } from '@/features/tab-linking'
import { createTabSync } from '../tabs/useTabSync'
import { createNoteEditors } from '../tabs/useNoteEditors'
import { useSettings } from './useSettings'
import { useVaults } from './useVaults'
import type { TabsHold } from '../commands/leaveVault'
import type { useCommands } from '../commands/useCommands'
import type { TabCommands } from '../tabs/tabCommands'
import type { createVersionCommands } from '../commands/versionCommands'
import type { useDeclaredCommands } from '../commands/declaredCommands'
import type { CommandItemLists } from '../commands/commandItemLists'
import { useWindowDisplay } from './useWindowDisplay'
import { createDisplayOptions } from './useWindowDisplayOptions'
import type { useWindowKinds } from './useWindowKinds'
import type { createWindowSlots } from './windowSlots'
import type { createPluginNotes } from '../plugins/pluginNotes'
import type { PluginServices } from '../plugins/pluginServices'
import type { createPluginSupport } from '../plugins/useWindowPlugins'

/** What the window is made of before its parts exist: the tabs, the plugins and the message log. */
export interface WindowCore {
  readonly frame: ReturnType<typeof useWindowFrame>
  readonly log: ReturnType<typeof createMessageLog>
  readonly writeMessage: MessageWriter
  readonly windowId: string
  readonly windowTabs: ReturnType<typeof useWindowTabs>
  readonly tabLinks: ReturnType<typeof useTabLinks>
  readonly tabReports: { wait: () => Promise<void> }
  readonly tabOpeners: ReturnType<typeof fileOpeners>
  readonly runs: ReturnType<typeof runSupport>
  readonly declaredCommands: ReturnType<typeof useDeclaredCommands>
  readonly itemLists: CommandItemLists
  readonly pluginHost: ReturnType<typeof createPluginSupport>['pluginHost']
  readonly pluginLoader: ReturnType<typeof createPluginSupport>['pluginLoader']
  readonly pluginServices: PluginServices
  readonly pluginNotices: ReturnType<typeof createNoticeQueue>
  readonly pluginNotes: ReturnType<typeof createPluginNotes>
  readonly slotWindowValues: ReturnType<typeof createWindowSlots>['windowValues']
  readonly appCommands: { current?: () => readonly Command[] }
}

export interface WindowContext {
  commands?: ReturnType<typeof useCommands>
  kinds?: ReturnType<typeof useWindowKinds>
  vaults?: ReturnType<typeof useVaults>
  tabsHold?: TabsHold
  tabCommands?: TabCommands
  versionCommands?: ReturnType<typeof createVersionCommands>
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
    core,
    log: base.log,
    tabOpeners: base.tabOpeners,
    windowTabs: base.windowTabs,
  })

  const tabSync = createTabSync({
    windowTabs: base.windowTabs,
    tabLinks: base.tabLinks,
    sendSync: (sync) => core.syncTabPath(sync),
    onError: (error) => base.writeMessage(formatError(error), 'error'),
  })

  const window = useWindowDisplay(
    core,
    createDisplayOptions({
      editing,
      getCommands: () => getCommands(ctx),
      getKinds: () => getKinds(ctx),
      tabOpeners: base.tabOpeners,
      windowTabs: base.windowTabs,
      getVaults: () => getVaults(ctx),
      notifyPlugins: (event) => base.pluginHost.notifyVaultChange(event),
      writeMessage: base.writeMessage,
    }),
  )

  const settings = useSettings({
    core,
    words,
    log: base.log,
    windowTabs: base.windowTabs,
    onSizeChanged: () => editing.noted.measureAll(),
    catalog: pluginCatalog,
    getSettingsViews: (pluginId) => base.pluginHost.getRegisteredSettingsViews(pluginId),
    getPlatform: () => base.frame.platform.value,
  })
  settingsHolder.current = settings

  ctx.vaults = useVaults({
    core,
    words,
    log: base.log,
    chunks: window.chunks,
    embedded: window.embedded,
    isEmbedding: window.isEmbedding,
  })

  return { editing, tabSync, window, settings }
}

export type WindowServices = ReturnType<typeof initWindowServices>
