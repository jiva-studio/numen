/**
 * The plugins of the window: the host and the loader, the commands the plugins declare and the
 * services a plugin is given, made without the host: the services reach the commands through the
 * port, which the palettes connect once they are made.
 */
import { createNoticeQueue } from '@numen/workbench'
import { createCommandTitleReader } from '@/entities/command'
import { useDeclaredCommands } from '@/features/plugin-commands'
import { createHostPort, createPluginNotes, isCommandListed } from '@/features/plugin-host'
import { syncKeybindings } from '@/features/hotkeys'
import type { createWindowBase } from './createWindowBase'
import { createPluginDeclarations } from './createPluginDeclarations'
import { createPluginHostSupport, createServiceParts } from './createPluginHost'
import {
  createPluginAccess,
  createPluginBackgroundTasks,
  createPluginWorkspace,
  createWindowOwnedServices,
} from './createPluginServices'

type WindowBase = ReturnType<typeof createWindowBase>
type WindowFoundation = WindowBase['foundation']

/** What the plugins of the window read of the foundation. */
type PluginsFoundation = Pick<
  WindowFoundation,
  'api' | 'log' | 'rest' | 'frame' | 'status' | 'registry'
>

/** What the plugins of the window read of the base. */
export interface WindowPluginsDeps {
  readonly foundation: PluginsFoundation
  readonly tabs: WindowBase['tabs']
  readonly vault: WindowBase['vault']
  readonly windowSettings: WindowBase['windowSettings']
  readonly getTarget: WindowBase['getTarget']
  readonly appCommands: WindowBase['appCommands']
}

export function createWindowPlugins(deps: WindowPluginsDeps) {
  const { foundation, tabs, vault, windowSettings, getTarget, appCommands } = deps
  const { api, log, rest, frame, status, registry } = foundation
  const { note } = api.vault
  const { editing, window, vaults, itemLists } = vault
  const hostPort = createHostPort()
  const notices = createNoticeQueue()
  const pluginNotes = createPluginNotes({ read: note.readNote, write: note.writeNote })
  const declarations = createPluginDeclarations({
    api: api.plugins.declaredCommands,
    rest,
    log,
    isPaletteOpen: hostPort.isPaletteOpen,
    getTarget,
    appCommands,
    itemLists,
    tabOpeners: tabs.tabOpeners,
  })
  const services = {
    workspace: createPluginWorkspace({ log, tabs, window, hostPort }),
    ...createPluginAccess({
      api: api.vault,
      making: editing.making,
      pluginNotes,
      getTarget,
      hostPort,
    }),
    ...createWindowOwnedServices({
      editing,
      window,
      vaults,
      hostPort,
      frame,
      log,
      backgroundTasks: createPluginBackgroundTasks({
        cancelTask: api.window.tasks.cancelTask,
        log,
        notices,
        window,
        appearance: windowSettings.appearance,
        commands: hostPort.commands,
        getCommandTitle: createCommandTitleReader(appCommands),
        getPluginName: declarations.catalog.getName,
      }),
    }),
  }
  const { pluginHost, pluginLoader } = createPluginHostSupport({
    rest,
    catalog: declarations.catalog,
    status,
    frame,
    tabs,
    reports: declarations.reports,
    parts: createServiceParts({
      api,
      rest,
      services,
      notices,
      pluginNotes,
      hostParts: declarations.hostParts,
    }),
  })
  vault.vaultChanges.subscribe((event) => pluginHost.notifyVaultChange(event))
  const declaredCommands = useDeclaredCommands({
    list: api.plugins.declaredCommands.listDeclaredCommands,
    run: api.plugins.declaredCommands.runDeclaredCommand,
    state: declarations.declared,
    registry,
    isListed: (id) => isCommandListed(pluginHost, id),
    hasBackend: declarations.catalog.hasBackend,
    listItems: itemLists.listItems,
  })
  syncKeybindings(
    pluginHost.keybindings,
    declaredCommands.contributions.keybindings,
    rest.state.plugins,
  )
  return { hostPort, pluginHost, pluginLoader, declaredCommands }
}
