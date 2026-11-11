/**
 * The plugins of the window: the host and the loader, the commands the plugins declare and the
 * services a plugin is given.
 */
import { createNoticeQueue } from '@numen/workbench'
import { readNote, writeNote } from '@/entities/note'
import { useDeclaredCommands } from '@/features/plugin-commands'
import { createCommandIconFinder, createHostPort, createPluginNotes, isCommandListed, type PluginServices } from '@/features/plugin-host'
import type { WindowFeatures } from './createWindowFeatures'
import type { WindowFoundation } from './createWindowFoundation'
import type { WindowVault } from './createWindowVault'
import {
  createPluginBackgroundTasks,
  createPluginDeclarations,
  createPluginHostSupport,
  createVaultFacingServices,
  createWindowOwnedServices,
} from './createPluginsPart'

/** What the plugins of the window use of the other parts of it. */
export interface WindowPluginsDeps {
  readonly foundation: Pick<
    WindowFoundation,
    'rest' | 'log' | 'writeMessage' | 'status' | 'frame' | 'registry' | 'settingsRequests'
  >
  readonly parts: Pick<WindowVault, 'tabsPart' | 'vault' | 'windowSettings'>
  readonly features: Pick<WindowFeatures, 'itemLists' | 'featureCommands' | 'getTarget'>
}

export function createWindowPlugins(deps: WindowPluginsDeps) {
  const { rest, log, writeMessage, status, frame, registry, settingsRequests } = deps.foundation
  const { windowTabs, tabOpeners } = deps.parts.tabsPart
  const { vault, windowSettings } = deps.parts
  const { editing, window, vaults } = vault
  const { itemLists, getTarget } = deps.features

  const hostPort = createHostPort()
  const pluginNotices = createNoticeQueue()
  const pluginNotes = createPluginNotes({ read: readNote, write: writeNote })
  const declarations = createPluginDeclarations({
    rest,
    log,
    writeMessage,
    hostPort,
    getTarget,
    appCommands: deps.features.featureCommands.commands,
    itemLists,
    tabOpeners,
  })
  const getServices = (): PluginServices => services
  const support = createPluginHostSupport({
    rest,
    status,
    getPlatform: () => frame.platform.value,
    windowTabs,
    tabOpeners,
    declarations,
    access: { getServices, notices: pluginNotices, pluginNotes },
  })
  const { pluginHost } = support
  vault.vaultChanges.subscribe((event) => pluginHost.notifyVaultChange(event))
  const declaredCommands = useDeclaredCommands({
    state: declarations.declared,
    registry,
    isListed: (id) => isCommandListed(pluginHost, id),
    hasBackend: declarations.catalog.hasBackend,
    listItems: itemLists.listItems,
  })
  const owned = createWindowOwnedServices({
    editing,
    window,
    vaults,
    hostPort,
    backgroundTasks: createPluginBackgroundTasks({
      log,
      notices: pluginNotices,
      window,
      windowSettings,
      writeMessage,
      openSettings: settingsRequests.emit,
    }),
    frame,
    writeMessage,
  })
  const services: PluginServices = {
    ...createVaultFacingServices({
      foundation: deps.foundation,
      tabsPart: deps.parts.tabsPart,
      vault,
      pluginNotes,
      getTarget,
      runInvocation: hostPort.runInvocation,
      getPluginCommandIcon: createCommandIconFinder(pluginHost, declaredCommands),
    }),
    buffers: owned.buffers,
    noteBuffer: owned.noteBuffer,
    taskList: owned.taskList,
    backgroundTasks: owned.backgroundTasks,
    commands: owned.commands,
    vaults: owned.vaults,
    window: owned.window,
  }
  return { hostPort, support, declaredCommands }
}

export type WindowPlugins = ReturnType<typeof createWindowPlugins>
