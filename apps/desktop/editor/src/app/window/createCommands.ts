/**
 * The commands of the window: the commands the features declare and the entries the plugins
 * declare, collected and passed to the palette, the shortcut dispatch, and the run of a command.
 */

import { SLOT, type PluginHost, type SlotHost } from '@numen/plugin-loader'
import {
  createInvocation,
  createCommandPalette,
  useSearch,
  PALETTE_WORDS,
  type SearchMenuDeps,
  type NoteLookup,
  type PaletteLists,
  type SearchDeps,
} from '@/features/command-palette'
import {
  type Command,
  type CommandTarget,
  type Notes,
  type RunSupport,
  type CommandDeps,
  type CommandRegistry,
} from '@/entities/command'
import { createCommandDeps } from '../tabs/commandDeps'
import { createKeyHandler } from './createKeyHandler'
import {
  createChoiceRunner,
  createCommandRunner,
  getValues,
  runInvocationToEnd,
  withValues,
} from './createCommandRunner'
import {
  createCommandNoteReader,
  usePluginCommands,
  createItemLists,
  createCommandItems,
  watchCommandLists,
  type DeclaredCommands,
} from '@/features/plugin-commands'
import type { IndexCoverage } from '@/shared/model/notices/coverage'
import type { MessageLog, MessageWriter } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'
import type { WORDS } from '@/shared/ui/words'
import { getFileKinds } from '@/entities/file'
import { getVaultList } from './vaultList'

type Words = typeof WORDS

import type { SettingsStore } from '@/entities/settings'
import { createEmitter } from './createEmitter'

export interface CommandsDepsOptions {
  core: SearchDeps
  words: Words
  log: MessageLog
  windowTabs: WindowTabs
  getTarget: () => CommandTarget
  knows: NoteLookup
  paletteLists: PaletteLists
  runs: RunSupport
  registry: CommandRegistry
  coverage: () => IndexCoverage
  reached: Notes
  settingPalette?: {
    getCommands: () => readonly Command[]
    choose: (command: string, item: string) => Promise<void>
  }
  settings: Pick<SettingsStore, 'read' | 'state'>
  writeMessage: MessageWriter
  pluginHost?: PluginHost | undefined
  slots?: SlotHost
  getAppCommands?: () => readonly Command[]
  declaredCommands?: DeclaredCommands
  readCommandNote?: ReturnType<typeof createCommandNoteReader>
}

const createSearchMenu = (slots: SlotHost | undefined): SearchMenuDeps => ({
  getEntries: (target) => slots?.getEntries(SLOT.paletteSearchResult, target) ?? [],
})

const listExtraCommands = (
  options: CommandsDepsOptions,
  pluginEntries: readonly Command[],
): readonly Command[] => [
  ...(options.getAppCommands?.() ?? []),
  ...(options.settingPalette?.getCommands() ?? []),
  ...(options.declaredCommands?.paletteCommands.value ?? []),
  ...pluginEntries,
]

function createPaletteItems(
  options: Pick<
    CommandsDepsOptions,
    'getAppCommands' | 'declaredCommands' | 'pluginHost' | 'getTarget'
  >,
  getCommandNote: ReturnType<typeof createCommandNoteReader>,
  commandDeps: CommandDeps,
) {
  const { pluginHost } = options
  return createCommandItems({
    getAppCommands: () => [
      ...(options.getAppCommands?.() ?? []),
      ...(options.declaredCommands?.paletteCommands.value ?? []),
    ],
    pluginCommands: pluginHost?.commands,
    getTarget: options.getTarget,
    runApp: async (id, target, itemId, values) => {
      const invocation = withValues(createInvocation(id, target, itemId), values)
      await runInvocationToEnd(invocation, commandDeps)
    },
    runPlugin: async (id, target, itemId, values) => {
      const note = await getCommandNote(target)
      await pluginHost?.commands.executeCommand(id, note, {
        itemId,
        ...getValues(values),
        isAwaited: true,
      })
    },
    readTarget: async (target) => {
      const { type, properties } = await getCommandNote(target)
      return { ...target, noteType: type, ...(properties ? { properties } : {}) }
    },
  })
}

function createWindowCommandPalette(
  options: CommandsDepsOptions,
  lists: PaletteLists,
  listExtras: () => readonly Command[],
) {
  const { core, words, getTarget, knows, runs } = options
  return createCommandPalette({
    core: { ...core, vaults: getVaultList },
    words: { ...words, ...PALETTE_WORDS },
    at: getTarget,
    knows,
    lists,
    runs,
    getCommands: listExtras,
  })
}

function usePluginPalette(options: CommandsDepsOptions) {
  const { pluginHost, declaredCommands } = options
  return usePluginCommands(
    options.registry,
    options.readCommandNote ?? createCommandNoteReader({ fileKinds: getFileKinds }),
    pluginHost,
    {
      ...(declaredCommands ? { context: declaredCommands.whenContext } : {}),
      getHandlers: () => declaredCommands?.contributions.handlers.value,
    },
  )
}

export function createCommands(options: CommandsDepsOptions) {
  const { core, words, windowTabs, getTarget, coverage, writeMessage } = options
  const { pluginHost, declaredCommands, settings } = options
  const { getPaletteEntries, getCommandNote, pluginWays } = usePluginPalette(options)
  const palette = useSearch(
    core,
    { ...words, ...PALETTE_WORDS },
    {
      coverage,
      menu: createSearchMenu(options.slots),
      getTarget,
    },
  )
  const searchRequests = createEmitter<void>()
  const commandDeps: CommandDeps = createCommandDeps({
    core,
    windowTabs,
    reached: options.reached,
    ...(options.settingPalette ? { settingPalette: options.settingPalette } : {}),
    runs: options.runs,
    registry: options.registry,
    writeMessage,
    search: () => searchRequests.emit(),
  })
  const items = createPaletteItems(options, getCommandNote, commandDeps)
  const { lists, resetWhenClosed, removeEmptyCommands } = createItemLists(items, options)
  const listExtras = () => removeEmptyCommands(listExtraCommands(options, getPaletteEntries()))
  const commands = createWindowCommandPalette(options, lists, listExtras)

  if (declaredCommands) {
    watchCommandLists(
      {
        declaredCommands,
        writeMessage,
        plugins: settings.state.plugins,
        getCommittedSetting: settings.read.getCommittedSetting,
      },
      commands.palette.open,
    )
  }
  resetWhenClosed(commands.palette.open)
  searchRequests.subscribe(() => {
    commands.palette.setOpen(false)
    palette.setOpen(true)
  })

  const { runCommand, runCommandToEnd } = createCommandRunner({
    commands,
    palette,
    pluginHost,
    getCommandNote,
    getCommandDeps: () => commandDeps,
    writeMessage,
  })

  const onKeyDown = createKeyHandler({
    windowTabs,
    pluginHost,
    declaredCommands,
    plugins: settings.state.plugins,
    palette,
    commands,
    commandDeps,
    runCommand,
    getTarget,
    getCommandNote,
    writeMessage,
  })

  return {
    palette,
    commands,
    commandDeps,
    pluginRuns: { items, runCommandToEnd },
    runCommand,
    runChoice: createChoiceRunner(runCommand, getTarget),
    onKeyDown,
    pluginWays,
  }
}
