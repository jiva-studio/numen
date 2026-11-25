/**
 * The plugins of the window: the host and the loader, the commands the plugins declare and the
 * services a plugin is given, made without the host: the services reach the commands through the
 * port, which the palettes connect once they are made.
 */
import { createCommandTitleReader } from '@/entities/command'
import { useDeclaredCommands } from '@/features/plugin-commands'
import {
  createHostPort,
  createPluginNotes,
  createSlotGroups,
  isCommandListed,
} from '@/features/plugin-host'
import { syncKeybindings } from '@/features/hotkeys'
import type { createWindowBase } from './createWindowBase'
import { createPluginDeclarations } from './createPluginDeclarations'
import {
  createPluginHostSupport,
  createPluginSurfaces,
  createServiceParts,
} from './createPluginHost'
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
  readonly picker: WindowBase['picker']
}

interface BackgroundTaskHost {
  readonly window: WindowBase['vault']['window']
  readonly hostPort: ReturnType<typeof createHostPort>
  readonly notices: ReturnType<typeof createPluginSurfaces>['notices']
  readonly catalog: ReturnType<typeof createPluginDeclarations>['catalog']
}

function createBackgroundTasks(
  deps: Pick<WindowPluginsDeps, 'foundation' | 'windowSettings' | 'appCommands'>,
  host: BackgroundTaskHost,
) {
  return createPluginBackgroundTasks({
    cancelTask: deps.foundation.api.window.tasks.cancelTask,
    log: deps.foundation.log,
    notices: host.notices,
    window: host.window,
    appearance: deps.windowSettings.appearance,
    commands: host.hostPort.commands,
    getCommandTitle: createCommandTitleReader(deps.appCommands),
    getPluginName: host.catalog.getName,
  })
}

interface DeclaredCommandsHost {
  readonly foundation: PluginsFoundation
  readonly itemLists: WindowBase['vault']['itemLists']
  readonly pluginHost: ReturnType<typeof createPluginHostSupport>['pluginHost']
  readonly declarations: ReturnType<typeof createPluginDeclarations>
}

function startDeclaredCommands({
  foundation,
  itemLists,
  pluginHost,
  declarations,
}: DeclaredCommandsHost) {
  const { api, rest, registry } = foundation
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
  return { declaredCommands, slotGroups: createSlotGroups(declaredCommands) }
}

export function createWindowPlugins(deps: WindowPluginsDeps) {
  const { foundation, tabs, vault, getTarget, appCommands } = deps
  const { api, log, rest, frame, status } = foundation
  const { note } = api.vault
  const { editing, window, vaults, itemLists } = vault
  const hostPort = createHostPort()
  const { notices, picker, dialogs } = createPluginSurfaces(api.window.saveText, deps.picker)
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
      cloud: vault.cloud.fresh,
      hostPort,
      frame,
      log,
      backgroundTasks: createBackgroundTasks(deps, {
        window,
        hostPort,
        notices,
        catalog: declarations.catalog,
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
      picker,
      dialogs: dialogs.host,
    }),
  })
  vault.vaultChanges.subscribe((event) => {
    services.notes.clearRelatedNotes()
    pluginHost.notifyVaultChange(event)
  })
  const { declaredCommands, slotGroups } = startDeclaredCommands({
    foundation,
    itemLists,
    pluginHost,
    declarations,
  })
  return { hostPort, pluginHost, pluginLoader, declaredCommands, slotGroups, picker, dialogs }
}
