/**
 * The commands of the window: the commands the features declare and the entries the plugins
 * declare, collected and passed to the palette, the shortcut dispatch, and the run of a command.
 */
import type { PluginHost } from '@numen/plugin-loader'
import {
  createAppItemRunner,
  createChoiceRunner,
  createCommandDeps,
  createCommandPalette,
  createCommandRunner,
  createPaletteSwitch,
  createSearchRequests,
  createValuesRunner,
  PALETTE_WORDS,
  useSearch,
  type NoteLookup,
  type PaletteLists,
  type SearchDeps,
} from '@/features/command-palette'
import {
  createCommandItems,
  createItemLists,
  createPaletteCommandList,
  createPluginCommandPort,
  createSearchMenu,
  usePluginCommands,
  watchCommandLists,
  type createCommandNoteReader,
  type DeclaredCommands,
} from '@/features/plugin-commands'
import { createKeyHandler } from '@/features/hotkeys'
import { getVaultList } from '@/features/window-display'
import type { Command } from '@/entities/command'
import type { CommandRegistry, CommandTarget, Notes, RunSupport } from '@/shared/model/command'
import type { SettingsStore } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import type { IndexCoverage } from '@/shared/model/notices/coverage'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { WORDS } from '@/shared/ui/words'

export interface CommandsDepsOptions {
  core: SearchDeps
  words: typeof WORDS
  windowTabs: WindowTabs
  getTarget: () => CommandTarget
  knows: NoteLookup
  paletteLists: PaletteLists
  runs: RunSupport
  registry: CommandRegistry
  coverage: () => IndexCoverage
  reached: Notes
  settingPalette: {
    getCommands: () => readonly Command[]
    choose: (command: string, item: string) => Promise<void>
  }
  settings: Pick<SettingsStore, 'read' | 'state'>
  writeMessage: MessageWriter
  pluginHost: PluginHost
  getAppCommands: () => readonly Command[]
  declaredCommands: DeclaredCommands
  readCommandNote: ReturnType<typeof createCommandNoteReader>
  emitEvent?: ((event: string) => void) | undefined
}

function createPalettes(
  options: CommandsDepsOptions,
  plugin: Pick<ReturnType<typeof usePluginCommands>, 'getPaletteEntries' | 'getCommandNote'>,
) {
  const { core, words, windowTabs, getTarget, writeMessage, pluginHost, declaredCommands } = options
  const { settings, settingPalette } = options
  const palette = useSearch(
    core,
    { ...words, ...PALETTE_WORDS },
    { coverage: options.coverage, menu: createSearchMenu(pluginHost.slots), getTarget },
  )
  const searchRequests = createSearchRequests()
  const commandDeps = createCommandDeps({
    windowTabs,
    reached: options.reached,
    settingPalette,
    runs: options.runs,
    registry: options.registry,
    writeMessage,
    search: searchRequests.request,
    emitEvent: options.emitEvent,
  })
  const paletteCommands = createPaletteCommandList({
    getAppCommands: options.getAppCommands,
    getSettingCommands: settingPalette.getCommands,
    declaredCommands,
    getPluginEntries: plugin.getPaletteEntries,
  })
  const items = createCommandItems({
    getAppCommands: paletteCommands.listAppCommands,
    pluginCommands: pluginHost.commands,
    getTarget,
    runApp: createAppItemRunner(commandDeps),
    getCommandNote: plugin.getCommandNote,
  })
  const { lists, resetWhenClosed, removeEmptyCommands } = createItemLists(items, options)
  const commands = createCommandPalette({
    core: { ...core, vaults: getVaultList },
    words: { ...words, ...PALETTE_WORDS },
    at: getTarget,
    knows: options.knows,
    lists,
    runs: options.runs,
    getCommands: () => removeEmptyCommands(paletteCommands.listCommands()),
  })
  watchCommandLists(
    {
      declaredCommands,
      writeMessage,
      plugins: settings.state.plugins,
      getCommittedSetting: settings.read.getCommittedSetting,
    },
    commands.palette.open,
  )
  resetWhenClosed(commands.palette.open)
  const paletteSwitch = createPaletteSwitch(commands, palette)
  searchRequests.connect(paletteSwitch)
  return { palette, commands, commandDeps, items, paletteSwitch }
}

export function createCommands(options: CommandsDepsOptions) {
  const { windowTabs, getTarget, writeMessage, pluginHost, declaredCommands, settings } = options
  const plugin = usePluginCommands(options.registry, options.readCommandNote, pluginHost, {
    context: declaredCommands.whenContext,
    getHandlers: () => declaredCommands.contributions.handlers.value,
  })
  const { palette, commands, commandDeps, items, paletteSwitch } = createPalettes(options, plugin)
  const pluginCommands = createPluginCommandPort(pluginHost, plugin.getCommandNote)
  const { runCommand, runCommandToEnd } = createCommandRunner({
    commands,
    closeSearch: paletteSwitch.closeSearch,
    pluginCommands,
    getCommandDeps: () => commandDeps,
    writeMessage,
  })
  const onKeyDown = createKeyHandler({
    windowTabs,
    keybindings: pluginHost.keybindings,
    declaredKeybindings: declaredCommands.contributions.keybindings,
    plugins: settings.state.plugins,
    handleShortcut: (event, runApp) => pluginCommands.handleShortcut(event, getTarget, runApp),
    toggleSearch: paletteSwitch.toggleSearch,
    toggleCommands: paletteSwitch.toggleCommands,
    runCommand,
    runWithValues: createValuesRunner(commandDeps, writeMessage),
    getTarget,
  })

  return {
    palette,
    commands,
    commandDeps,
    pluginRuns: { items, runCommandToEnd },
    runCommand,
    runChoice: createChoiceRunner(runCommand, getTarget),
    onKeyDown,
    pluginWays: plugin.pluginWays,
  }
}
