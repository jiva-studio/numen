/**
 * The commands of the window: the commands the features declare and the entries the plugins
 * declare, collected and passed to the palettes, and the run of a command.
 */
import type { PluginHost } from '@numen/plugin-loader'
import {
  createAppItemRunner,
  createChoiceRunner,
  createCommandPalette,
  createCommandRunner,
  createPaletteSwitch,
  createValuesRunner,
  PALETTE_WORDS,
  type Commands,
  type NoteLookup,
  type PaletteLists,
  type SearchDeps,
  type SearchState,
} from '@/features/command-palette'
import {
  createCommandItems,
  createItemLists,
  createPaletteCommandList,
  watchCommandLists,
  type DeclaredCommands,
} from '@/features/plugin-commands'
import type { createPluginCommandPort, usePluginCommands } from '@/features/plugin-commands'
import { getVaultList } from '@/features/window-display'
import type { Command } from '@/entities/command'
import type { CommandDeps, CommandTarget, RunSupport } from '@/shared/model/command'
import type { SettingsStore } from '@/entities/settings'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { WORDS } from '@/shared/ui/words'

/** What the commands the palettes list are taken from. */
export interface CommandSourcesDeps {
  readonly appCommands: readonly Command[]
  readonly settingPalette: { readonly getCommands: () => readonly Command[] }
  readonly declaredCommands: DeclaredCommands
  readonly pluginHost: Pick<PluginHost, 'commands'>
  readonly plugin: Pick<ReturnType<typeof usePluginCommands>, 'getPaletteEntries' | 'getCommandNote'>
  readonly getTarget: () => CommandTarget
  readonly commandDeps: CommandDeps
}

/** The commands of the application, the settings and the plugins, and the items they list. */
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

/** What the palette of the commands is made from. */
export interface PaletteCommandsDeps {
  readonly core: SearchDeps
  readonly words: typeof WORDS
  readonly knows: NoteLookup
  readonly getTarget: () => CommandTarget
  readonly paletteLists: PaletteLists
  readonly runs: RunSupport
  readonly sources: ReturnType<typeof createCommandSources>
}

/** The palette of the commands, over the commands and the items the sources list. */
export function createPaletteCommands(deps: PaletteCommandsDeps): Commands {
  const { core, words, getTarget, runs, sources } = deps
  const { lists, resetWhenClosed, removeEmptyCommands } = createItemLists(sources.items, {
    paletteLists: deps.paletteLists,
    getTarget,
    runs,
  })
  const commands = createCommandPalette({
    core: { ...core, vaults: getVaultList },
    words: { ...words, ...PALETTE_WORDS },
    at: getTarget,
    knows: deps.knows,
    lists,
    runs,
    getCommands: () => removeEmptyCommands(sources.paletteCommands.listCommands()),
  })
  resetWhenClosed(commands.palette.open)
  return commands
}

/** What the switch between the two palettes is made from. */
export interface PaletteSwitchingDeps {
  readonly declaredCommands: DeclaredCommands
  readonly writeMessage: MessageWriter
  readonly settings: Pick<SettingsStore, 'read' | 'state'>
  readonly commands: Commands
  readonly search: SearchState
  readonly requests: { connect(paletteSwitch: ReturnType<typeof createPaletteSwitch>): void }
}

/** The switch between the search and the commands, and the commands lists kept as the settings change. */
export function createPaletteSwitching(deps: PaletteSwitchingDeps) {
  const { declaredCommands, writeMessage, settings, commands, search } = deps
  watchCommandLists(
    {
      declaredCommands,
      writeMessage,
      plugins: settings.state.plugins,
      getCommittedSetting: settings.read.getCommittedSetting,
    },
    commands.palette.open,
  )
  const paletteSwitch = createPaletteSwitch(commands, search)
  deps.requests.connect(paletteSwitch)
  return paletteSwitch
}

/** What the run of a command is made from. */
export interface CommandRunsDeps {
  readonly commands: Commands
  readonly paletteSwitch: Pick<ReturnType<typeof createPaletteSwitch>, 'closeSearch'>
  readonly pluginCommands: ReturnType<typeof createPluginCommandPort>
  readonly commandDeps: CommandDeps
  readonly writeMessage: MessageWriter
  readonly getTarget: () => CommandTarget
}

/** The ways a command is run: by its id, to its end, from a pick and with values. */
export function createCommandRuns(deps: CommandRunsDeps) {
  const { commandDeps, writeMessage, getTarget } = deps
  const { runCommand, runCommandToEnd } = createCommandRunner({
    commands: deps.commands,
    closeSearch: deps.paletteSwitch.closeSearch,
    pluginCommands: deps.pluginCommands,
    getCommandDeps: () => commandDeps,
    writeMessage,
  })
  return {
    runCommand,
    runCommandToEnd,
    runChoice: createChoiceRunner(runCommand, getTarget),
    runWithValues: createValuesRunner(commandDeps, writeMessage),
  }
}
