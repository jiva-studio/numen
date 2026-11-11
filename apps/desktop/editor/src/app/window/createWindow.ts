/**
 * The window composed: the services that read the vault, the commands, the kinds of tab and the
 * plugins, made in the order each needs the one before.
 */
import { useWindowFrame } from '@numen/workbench'
import { createCommandRegistry } from '@/shared/model/command'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { createWindowLayout, startPlacedTabs, useOpenTabs } from '@/entities/tab'
import { isSecondaryWindow } from '@/entities/window'
import { createWindowBootstrap } from '@/features/app-start'
import { createCommandNotes } from '@/features/command-palette'
import { useAppHotkeys } from '@/features/hotkeys'
import { createWindowPluginStatus } from '@/features/plugin-commands'
import { createWindowTabLinkSession } from '@/features/tab-linking'
import {
  createMoveAcceptor,
  createMovedTabStarter,
  createWindowTabMover,
  readDetachedTab,
} from '@/features/tab-move'
import { formatError } from '@/shared/api/errors'
import { createEmitter } from '@/shared/lib/emitter'
import { createMessageLog } from '@/shared/model/notices/messages'
import {
  createEditorCommands,
  createTargetAndFeatureCommands,
  createWindowKinds,
} from './createEditorCommands'
import { createPluginsCore, createWindowFrameServices } from './createPluginsPart'
import { createTabsCore } from './createTabsPart'
import { createVaultServices, createWindowServices, type VaultServices } from './createWindowServices'
import type { SettingsRequest, WindowCore, WindowFoundation } from './windowCore'

function createWindowFoundation(): WindowFoundation {
  const log = createMessageLog()
  const rest = createSettingsStore(SETTINGS_WORDS, log.getWriter('configured'))
  const writeMessage = log.getWriter('command')
  const frame = useWindowFrame()
  const status = createWindowPluginStatus(rest)
  const settingsRequests = createEmitter<SettingsRequest>()
  const tabs = createTabsCore({
    writeMessage,
    rest,
    isPluginStarting: status.isPluginStarting,
    openPluginSettings: (pluginId) => settingsRequests.emit({ pluginId, filter: undefined }),
  })
  const registry = createCommandRegistry()
  return {
    frame,
    log,
    rest,
    writeMessage,
    status,
    registry,
    tabs,
    settingsRequests,
  }
}

function createWindowTabParts(
  base: Pick<WindowFoundation, 'tabs' | 'writeMessage'>,
  services: Pick<VaultServices, 'editing' | 'tabSync'>,
) {
  const { windowTabs, windowId, tabLinks, reports } = base.tabs
  const { writeMessage } = base
  const isSecondary = isSecondaryWindow()
  const layout = createWindowLayout(isSecondary)
  const moves = createWindowTabMover({
    windowId,
    windowTabs,
    notes: createCommandNotes(services.editing.reached, windowTabs),
    closeTab: windowTabs.closeTab,
    writeMessage,
  })
  const openTabs = useOpenTabs({
    windowTabs,
    isWriting: layout.isWritingTabs,
    reportError: (error) =>
      writeMessage(`The open tabs were not saved: ${formatError(error)}`, 'error'),
  })
  const openMovedTab = createMovedTabStarter({
    windowTabs,
    closeTab: windowTabs.closeTab,
    acceptMove: createMoveAcceptor(windowId, reports.waitForReports),
    onError: (error) => writeMessage(`The moved tab did not open: ${formatError(error)}`, 'error'),
  })
  const linkSession = createWindowTabLinkSession({
    windowId,
    windowTabs,
    isMain: !isSecondary,
    tabLinks,
    reports,
    applySync: services.tabSync.applySync,
    applyGroupPaths: services.tabSync.applyGroupPaths,
    openMovedTab,
    writeMessage,
  })
  return {
    layout,
    moves,
    openTabs,
    openMovedTab,
    linkSession,
    isSecondary,
    openCurrentVault: () => startPlacedTabs(windowTabs, writeMessage),
  }
}

/** Everything the window is made of, made once and handed to what draws it. */
export const createWindow = () => {
  const foundation = createWindowFoundation()
  const { windowTabs } = foundation.tabs
  const vaultServices = createVaultServices(foundation)
  const tabs = createWindowTabParts(foundation, vaultServices)
  const initialCommands = createTargetAndFeatureCommands(foundation, vaultServices, tabs.moves, {
    pause: async () => tabs.openTabs.pause(),
    resume: () => tabs.openTabs.resume(),
  })
  const plugins = createPluginsCore({
    foundation,
    vaultServices,
    itemLists: initialCommands.itemLists,
    getTarget: initialCommands.getTarget,
    getAppCommands: () => initialCommands.featureCommands.commands,
  })
  const base: WindowCore = { ...foundation, plugins }
  const services = createWindowServices(base, vaultServices)
  const { editing, window, settings, vaults } = services
  base.settingsRequests.subscribe((request) =>
    settings.configured.openSettings(request.pluginId, request.filter),
  )
  const commands = createEditorCommands(base, services, initialCommands)
  const kinds = createWindowKinds(base, services)
  useAppHotkeys(commands.commands.onKeyDown)
  const bootstrap = createWindowBootstrap({
    vaults,
    windowTabs,
    settings: {
      start: () => settings.start(),
      close: () => settings.close(),
      getPlugins: () => settings.rest.state.plugins.value,
    },
    window,
    editing,
    pluginLoader: base.plugins.host.pluginLoader,
    waitForDeclaredCommands: () => base.plugins.declaredCommands.waitForFirstList(),
    writeMessage: base.writeMessage,
    layout: tabs.layout,
    isSecondary: tabs.isSecondary,
    windowId: base.tabs.windowId,
    tabLinkSession: tabs.linkSession,
    readDetachedTab,
    openDetachedTab: tabs.openMovedTab,
  })
  const frame = createWindowFrameServices(base, commands)
  return {
    base,
    services,
    commands,
    kinds,
    frame,
    bootstrap,
    tabs,
  }
}
