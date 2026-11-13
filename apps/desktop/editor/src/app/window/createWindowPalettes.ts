/**
 * The palettes of the window: what they share, and what the plugins add to the commands they list.
 */
import {
  createCommandDeps,
  createNoteLookup,
  createSearchRequests,
  createWindowPalette,
  runSupport,
} from '@/features/command-palette'
import { getVaultList } from '@/features/window-display'
import { watchCommandLists } from '@/features/plugin-commands'
import { createCommandIconFinder, createSearchMenu } from '@/features/plugin-host'
import type { CommandRegistry, Notes, SettingsRequest } from '@/shared/model/command'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'
import { createHostConnection } from './createHostConnection'
import { createPluginPalette } from './createPluginPalette'
import type { createWindowBase } from './createWindowBase'
import type { createWindowPlugins } from './createWindowPlugins'

/** What the palettes and the runs of the commands share. */
export interface PaletteBaseDeps {
  readonly windowTabs: Pick<WindowTabs, 'tabs' | 'panes'>
  readonly reached: Notes
  readonly settingPalette: { choose: (command: string, item: string) => Promise<void> }
  readonly registry: Pick<CommandRegistry, 'get'>
  readonly writeMessage: MessageWriter
  readonly openSettings: (request: SettingsRequest) => void
}

function createPaletteBase(deps: PaletteBaseDeps) {
  const searchRequests = createSearchRequests()
  return {
    searchRequests,
    commandDeps: createCommandDeps({
      windowTabs: deps.windowTabs,
      reached: deps.reached,
      settingPalette: deps.settingPalette,
      runs: runSupport(),
      registry: deps.registry,
      writeMessage: deps.writeMessage,
      navigation: { search: searchRequests.request, openSettings: deps.openSettings },
    }),
  }
}

type WindowBase = ReturnType<typeof createWindowBase>
type WindowPlugins = ReturnType<typeof createWindowPlugins>

/** What the palettes of the window read of the base and the plugins. */
export interface WindowPalettesDeps {
  readonly foundation: WindowBase['foundation']
  readonly tabs: WindowBase['tabs']
  readonly vault: WindowBase['vault']
  readonly windowSettings: WindowBase['windowSettings']
  readonly getTarget: WindowBase['getTarget']
  readonly appCommands: WindowBase['appCommands']
  readonly plugins: WindowPlugins
}

export function createWindowPalettes(deps: WindowPalettesDeps) {
  const { foundation, vault, windowSettings, getTarget, appCommands, plugins } = deps
  const { rest, writeMessage } = foundation
  const { pluginHost, declaredCommands, hostPort } = plugins
  const { windowTabs } = deps.tabs
  const { commandDeps, searchRequests } = createPaletteBase({
    windowTabs,
    reached: vault.editing.reached,
    settingPalette: windowSettings.settingPalette,
    registry: foundation.registry,
    writeMessage,
    openSettings: foundation.settingsRequests.emit,
  })
  const pluginPalette = createPluginPalette({
    pluginHost,
    declaredCommands,
    itemLists: vault.itemLists,
    commandDeps,
    registry: foundation.registry,
    appCommands,
    getTarget,
    windowSettings,
  })
  const palette = createWindowPalette({
    commandDeps,
    getTarget,
    knows: createNoteLookup(vault.editing, windowTabs),
    plugin: pluginPalette.plugin,
    searchRequests,
    searchOptions: { menu: createSearchMenu(pluginHost.slots) },
    getVaultList,
  })
  watchCommandLists(
    {
      declaredCommands,
      writeMessage,
      plugins: rest.state.plugins,
      getCommittedSetting: rest.read.getCommittedSetting,
    },
    palette.commands.palette.open,
  )
  vault.renames.subscribe((renamed) => palette.commands.navigation.applyRenames(renamed))
  return {
    palette,
    commandDeps,
    pluginPalette,
    windowCommands: createHostConnection({
      hostPort,
      pluginHost,
      appCommands,
      getTarget,
      items: pluginPalette.sources.items,
      palette,
      commandDeps,
      getCommandIcon: createCommandIconFinder(pluginHost, declaredCommands),
    }),
  }
}
