/**
 * The palettes of the window, the run of a command and the keys: the commands of the features, the
 * settings and the plugins, listed in the palette of the commands and the search.
 */
import {
  createCommandDeps,
  createNoteLookup,
  createSearchRequests,
  PALETTE_WORDS,
  runSupport,
  searchCore,
  useSearch,
} from '@/features/command-palette'
import { createCommandNoteReader, createPluginCommandPort, usePluginCommands } from '@/features/plugin-commands'
import { createSearchMenu } from '@/features/plugin-host'
import { WORDS as words } from '@/shared/ui/words'
import {
  createCommandRuns,
  createCommandSources,
  createPaletteCommands,
  createPaletteSwitching,
} from './createCommands'
import type { WindowFeatures } from './createWindowFeatures'
import type { WindowFoundation } from './createWindowFoundation'
import type { WindowPlugins } from './createWindowPlugins'
import type { WindowVault } from './createWindowVault'

/** What the palettes of the window use of the other parts of it. */
export interface WindowPalettesDeps {
  readonly foundation: Pick<WindowFoundation, 'rest' | 'writeMessage' | 'registry'>
  readonly parts: Pick<WindowVault, 'tabsPart' | 'vault' | 'windowSettings'>
  readonly features: WindowFeatures
  readonly plugins: Pick<WindowPlugins, 'support' | 'declaredCommands'>
}

export function createWindowPalettes(deps: WindowPalettesDeps) {
  const { rest, writeMessage, registry } = deps.foundation
  const { windowTabs } = deps.parts.tabsPart
  const { editing, vaults } = deps.parts.vault
  const { settingPalette, paletteLists } = deps.parts.windowSettings
  const { getTarget, itemLists } = deps.features
  const { declaredCommands } = deps.plugins
  const { pluginHost } = deps.plugins.support

  const plugin = usePluginCommands(registry, createCommandNoteReader(itemLists), pluginHost, {
    context: declaredCommands.whenContext,
    getHandlers: () => declaredCommands.contributions.handlers.value,
  })
  const runs = runSupport()
  const searchRequests = createSearchRequests()
  const commandDeps = createCommandDeps({
    windowTabs,
    reached: editing.reached,
    settingPalette,
    runs,
    registry,
    writeMessage,
    search: searchRequests.request,
  })
  const search = useSearch(
    searchCore,
    { ...words, ...PALETTE_WORDS },
    { coverage: vaults.coverage, menu: createSearchMenu(pluginHost.slots), getTarget },
  )
  const sources = createCommandSources({
    appCommands: deps.features.featureCommands.commands,
    settingPalette,
    declaredCommands,
    pluginHost,
    plugin,
    getTarget,
    commandDeps,
  })
  const paletteCommands = createPaletteCommands({
    core: searchCore,
    words,
    knows: createNoteLookup(editing, windowTabs),
    getTarget,
    paletteLists,
    runs,
    sources,
  })
  deps.parts.vault.renames.subscribe((renamed) => paletteCommands.navigation.applyRenames(renamed))
  const paletteSwitch = createPaletteSwitching({
    declaredCommands,
    writeMessage,
    settings: rest,
    commands: paletteCommands,
    search,
    requests: searchRequests,
  })
  const pluginCommands = createPluginCommandPort(pluginHost, plugin.getCommandNote)
  const commandRuns = createCommandRuns({
    commands: paletteCommands,
    paletteSwitch,
    pluginCommands,
    commandDeps,
    writeMessage,
    getTarget,
  })
  return {
    search,
    paletteCommands,
    commandDeps,
    plugin,
    sources,
    paletteSwitch,
    pluginCommands,
    commandRuns,
  }
}

export type WindowPalettes = ReturnType<typeof createWindowPalettes>
