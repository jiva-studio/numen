/**
 * The palettes of a window over the commands it offers: the palette of the commands, the search,
 * the switch between them, and the ways a command is run. What the plugins add to the commands is
 * given in, because a feature does not reach another.
 */
import type { Ref } from 'vue'
import type { Command } from '@/entities/command'
import type { CommandDeps, CommandTarget } from '@/shared/model/command'
import { WORDS } from '@/shared/ui/words'
import { searchCore } from '../../api/search'
import type { CommandsDeps, NoteLookup, PaletteLists } from '../../types'
import { PALETTE_WORDS } from '../../words'
import { createCommandPalette } from '../palette/palette'
import { createChoiceRunner } from '../run/createChoiceRunner'
import { createCommandRunner, type PluginCommandPort } from '../run/createCommandRunner'
import { createPaletteSwitch } from '../run/createPaletteSwitch'
import type { createSearchRequests } from '../run/createSearchRequests'
import { createValuesRunner } from '../run/createValuesRunner'
import { useSearch, type SearchOptions } from '../search/search'

/** The lists the palette of the commands shows for the commands the plugins declare. */
export interface PluginPaletteLists {
  readonly lists: PaletteLists
  readonly removeEmptyCommands: (commands: readonly Command[]) => readonly Command[]
  readonly resetWhenClosed: (isOpen: Readonly<Ref<boolean>>) => void
}

/** What the plugins add to the palettes: their lists, the commands they list and the commands they run. */
export interface PluginPaletteParts {
  readonly lists: PluginPaletteLists
  /** The commands the palette of the commands lists, before the empty ones are left out. */
  readonly getCommands: () => readonly Command[]
  readonly pluginCommands: PluginCommandPort
}

/** What the palettes of the window are made from. */
export interface WindowPaletteDeps {
  readonly commandDeps: CommandDeps
  readonly getTarget: () => CommandTarget
  readonly knows: NoteLookup
  readonly plugin: PluginPaletteParts
  readonly searchRequests: ReturnType<typeof createSearchRequests>
  readonly searchOptions: Pick<SearchOptions, 'coverage' | 'menu'>
  /** The vaults the installation holds. */
  readonly getVaultList: CommandsDeps['vaults']
}

export function createWindowPalette(deps: WindowPaletteDeps) {
  const { commandDeps, getTarget, plugin } = deps
  const { lists } = plugin
  const words = { ...WORDS, ...PALETTE_WORDS }
  const search = useSearch(searchCore, words, { ...deps.searchOptions, getTarget })
  const commands = createCommandPalette({
    core: { ...searchCore, vaults: deps.getVaultList },
    words,
    at: getTarget,
    knows: deps.knows,
    lists: lists.lists,
    runs: commandDeps.runSupport,
    getCommands: () => lists.removeEmptyCommands(plugin.getCommands()),
  })
  lists.resetWhenClosed(commands.palette.open)
  const paletteSwitch = createPaletteSwitch(commands, search)
  deps.searchRequests.connect(paletteSwitch)
  const { runCommand, runCommandToEnd } = createCommandRunner({
    commands,
    closeSearch: paletteSwitch.closeSearch,
    pluginCommands: plugin.pluginCommands,
    getCommandDeps: () => commandDeps,
    writeMessage: commandDeps.writeMessage,
  })
  return {
    search,
    commands,
    paletteSwitch,
    runs: {
      runCommand,
      runCommandToEnd,
      runChoice: createChoiceRunner(runCommand, getTarget),
      runWithValues: createValuesRunner(commandDeps, commandDeps.writeMessage),
    },
  }
}
