/**
 * The palettes of the window: what they share, and what the plugins add to the commands they list.
 */
import type { PluginHost } from '@numen/plugin-loader'
import {
  createCommandDeps,
  createNoteLookup,
  createSearchRequests,
  createWindowPalette,
  runSupport,
  type PaletteLists,
  type PluginPaletteLists,
} from '@/features/command-palette'
import { getVaultList } from '@/features/window-display'
import type { Command } from '@/entities/command'
import { createCommandNoteReader, createItemLists, createPluginCommandPort, usePluginCommands, watchCommandLists, type DeclaredCommands } from '@/features/plugin-commands'
import { createCommandIconFinder, createSearchMenu } from '@/features/plugin-host'
import type { CommandRegistry, CommandTarget, Notes } from '@/shared/model/command'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'
import { createCommandSources } from './createCommandSources'
import { createHostConnection } from './createHostConnection'
import type { createWindowBase } from './createWindowBase'
import type { createWindowPlugins } from './createWindowPlugins'

/** What the palettes and the runs of the commands share. */
export interface PaletteBaseDeps {
  readonly windowTabs: Pick<WindowTabs, 'tabs' | 'panes'>
  readonly reached: Notes
  readonly settingPalette: { choose: (command: string, item: string) => Promise<void> }
  readonly registry: Pick<CommandRegistry, 'get'>
  readonly writeMessage: MessageWriter
}

export function createPaletteBase(deps: PaletteBaseDeps) {
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
      search: searchRequests.request,
    }),
  }
}

/** What the commands of the plugins add to the palettes. */
export interface PluginPaletteDeps {
  readonly pluginHost: PluginHost
  readonly declaredCommands: DeclaredCommands
  readonly itemLists: Parameters<typeof createCommandNoteReader>[0]
  readonly commandDeps: ReturnType<typeof createPaletteBase>['commandDeps']
  readonly registry: CommandRegistry
  readonly appCommands: readonly Command[]
  readonly getTarget: () => CommandTarget
  readonly windowSettings: {
    readonly settingPalette: { readonly getCommands: () => readonly Command[] }
    readonly paletteLists: PaletteLists
  }
}

export function createPluginPalette(deps: PluginPaletteDeps) {
  const { pluginHost, declaredCommands, commandDeps, getTarget, windowSettings } = deps
  const plugin = usePluginCommands(
    deps.registry,
    createCommandNoteReader(deps.itemLists),
    pluginHost,
    {
      context: declaredCommands.whenContext,
      getHandlers: () => declaredCommands.contributions.handlers.value,
    },
  )
  const sources = createCommandSources({
    appCommands: deps.appCommands,
    settingPalette: windowSettings.settingPalette,
    declaredCommands,
    pluginHost,
    plugin,
    getTarget,
    commandDeps,
  })
  const lists: PluginPaletteLists = createItemLists(sources.items, {
    paletteLists: windowSettings.paletteLists,
    getTarget,
    runs: commandDeps.runSupport,
  })
  return {
    sources,
    pluginWays: plugin.pluginWays,
    plugin: {
      lists,
      getCommands: () => sources.paletteCommands.listCommands(),
      pluginCommands: createPluginCommandPort(pluginHost, plugin.getCommandNote),
    },
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
    searchOptions: { coverage: vault.vaults.coverage, menu: createSearchMenu(pluginHost.slots) },
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
