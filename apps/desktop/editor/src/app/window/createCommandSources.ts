/**
 * The commands the palettes list: the commands of the application, the settings and the plugins,
 * and the items the commands of the plugins offer.
 */
import type { PluginHost } from '@numen/plugin-loader'
import { createAppItemRunner } from '@/features/command-palette'
import { createCommandItems, createPaletteCommandList, type DeclaredCommands, type usePluginCommands } from '@/features/plugin-commands'
import type { Command } from '@/entities/command'
import type { CommandDeps, CommandTarget } from '@/shared/model/command'

/** What the commands the palettes list are taken from. */
export interface CommandSourcesDeps {
  readonly appCommands: readonly Command[]
  readonly settingPalette: { readonly getCommands: () => readonly Command[] }
  readonly declaredCommands: DeclaredCommands
  readonly pluginHost: Pick<PluginHost, 'commands'>
  readonly plugin: Pick<
    ReturnType<typeof usePluginCommands>,
    'getPaletteEntries' | 'getCommandNote'
  >
  readonly getTarget: () => CommandTarget
  readonly commandDeps: CommandDeps
}

export function createCommandSources(deps: CommandSourcesDeps) {
  const paletteCommands = createPaletteCommandList({
    getAppCommands: () => deps.appCommands,
    getSettingCommands: deps.settingPalette.getCommands,
    declaredCommands: deps.declaredCommands,
    getPluginEntries: deps.plugin.getPaletteEntries,
  })
  const items = createCommandItems({
    getAppCommands: paletteCommands.listAppCommands,
    pluginCommands: deps.pluginHost.commands,
    getTarget: deps.getTarget,
    runApp: createAppItemRunner(deps.commandDeps),
    getCommandNote: deps.plugin.getCommandNote,
  })
  return { paletteCommands, items }
}
