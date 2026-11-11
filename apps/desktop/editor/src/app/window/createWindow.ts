/**
 * The window composed: the services that read the vault, the commands, the kinds of tab and the
 * plugins, made in the order each needs the one before.
 */
import { useWindowFrame } from '@numen/workbench'
import { createCommandRegistry } from '@/entities/command'
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
import { createEditorCommands, createWindowKinds } from './createEditorCommands'
import { createPluginsCore, createWindowFrameServices } from './createPluginsPart'
import { createTabsCore } from './createTabsPart'
import { createWindowServices } from './createWindowServices'
import type { SettingsRequest, WindowCore } from './windowCore'

function createWindowCore(): WindowCore {
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
  const plugins = createPluginsCore({
    log,
    writeMessage,
    frame,
    windowTabs: tabs.windowTabs,
    tabOpeners: tabs.tabOpeners,
    rest,
    status,
    registry,
  })
  return { frame, log, rest, writeMessage, registry, tabs, plugins, settingsRequests }
}

function createWindowTabParts(base: WindowCore, services: ReturnType<typeof createWindowServices>) {
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
  const base = createWindowCore()
  const { windowTabs } = base.tabs
  const services = createWindowServices(base)
  const { editing, window, settings, vaults } = services
  base.settingsRequests.subscribe((request) =>
    settings.configured.openSettings(request.pluginId, request.filter),
  )
  const tabs = createWindowTabParts(base, services)
  const commands = createEditorCommands(base, services, tabs.moves, {
    pause: async () => tabs.openTabs.pause(),
    resume: () => tabs.openTabs.resume(),
  })
  const kinds = createWindowKinds(base, services, commands)
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
  const frame = createWindowFrameServices(base, services, commands, kinds)
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
