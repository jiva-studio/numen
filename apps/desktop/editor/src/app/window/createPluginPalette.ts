/**
 * What the commands of the plugins add to the palettes: the lists of their items, their commands and the
 * ways they are run.
 */
import type { PluginHost } from '@numen/plugin-loader'
import type {
  createCommandDeps,
  PaletteLists,
  PluginPaletteLists,
} from '@/features/command-palette'
import type { Command } from '@/entities/command'
import {
  createCommandNoteReader,
  createItemLists,
  createPluginCommandPort,
  usePluginCommands,
  type DeclaredCommands,
} from '@/features/plugin-commands'
import type { CommandRegistry, CommandTarget } from '@/shared/model/command'
import { createCommandSources } from './createCommandSources'

/** What the commands of the plugins add to the palettes. */
interface PluginPaletteDeps {
  readonly pluginHost: PluginHost
  readonly declaredCommands: DeclaredCommands
  readonly itemLists: Parameters<typeof createCommandNoteReader>[0]
  readonly commandDeps: ReturnType<typeof createCommandDeps>
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
    forgetNotes: declaredCommands.whenContext.forgetNotes,
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
