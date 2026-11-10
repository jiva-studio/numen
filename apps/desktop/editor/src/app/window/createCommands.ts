/**
 * The commands of the window: the commands the features declare and the entries the plugins
 * declare, collected and passed to the palette, the shortcut dispatch, and the run of a command.
 */

import { SLOT, type PluginHost, type SlotHost } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import {
  runInvocation,
  createInvocation,
  createCommandPalette,
  useSearch,
  PALETTE_WORDS,
  type ResultChoice,
  type SearchMenuDeps,
  type NoteLookup,
  type PaletteLists,
  type SearchDeps,
} from '@/features/command-palette'
import {
  COMMAND,
  type Command,
  type CommandInvocation,
  type CommandTarget,
  type Notes,
  type RunSupport,
  type CommandDeps,
} from '@/entities/command'
import { formatError } from '@/shared/api/errors'
import { createCommandDeps } from '../tabs/commandDeps'
import {
  createCommandRunner,
  getValues,
  runInvocationToEnd,
  withValues,
  type Values,
} from './createCommandRunner'
import {
  createCommandNoteReader,
  usePluginCommands,
  createItemLists,
  toRunTarget,
  createCommandItems,
  watchCommandLists,
  type DeclaredCommands,
} from '@/features/plugin-commands'
import { syncKeybindings } from '@/features/hotkeys'
import type { IndexCoverage } from '@/shared/notices/coverage'
import type { MessageLog, MessageWriter } from '@/shared/notices/messages'
import type { WindowTabs } from '@/entities/tab'
import type { WORDS } from '@/shared/words'
import { getFileKinds } from '@/entities/file'
import { getVaultList } from './vaultList'

type Words = typeof WORDS

import type { SettingsStore } from '@/entities/settings'
import { createEmitter } from '@/shared/lib/emitter'

export interface CommandsDepsOptions {
  core: SearchDeps
  words: Words
  log: MessageLog
  windowTabs: WindowTabs
  getTarget: () => CommandTarget
  knows: NoteLookup
  paletteLists: PaletteLists
  runs: RunSupport
  coverage: () => IndexCoverage
  reached: Notes
  settingPalette?: {
    getCommands: () => readonly Command[]
    choose: (command: string, item: string) => Promise<void>
  }
  /** The plugins of the installation and the settings that switch them on. */
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

const toResultTarget = (place: NonNullable<ResultChoice['over']>) => ({
  tab: '',
  kind: null,
  path: place.path,
  file: place.file,
  title: place.title,
  ...(place.line === undefined ? {} : { line: place.line }),
  ...(place.selector === undefined ? {} : { selector: place.selector }),
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
    pluginHost,
    getTarget: options.getTarget,
    runApp: async (id, target, itemId, values) => {
      const invocation = withValues(createInvocation(id, target, itemId), values)
      await runInvocationToEnd(invocation, commandDeps)
    },
    runPlugin: async (id, target, itemId, values) => {
      const note = await getCommandNote(target)
      await pluginHost?.executeCommand(id, note, { itemId, ...getValues(values), isAwaited: true })
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

export function createCommands(options: CommandsDepsOptions) {
  const { core, words, windowTabs, getTarget, coverage, writeMessage } = options
  const { pluginHost, declaredCommands, settings } = options
  const plugins = usePluginCommands(
    options.readCommandNote ?? createCommandNoteReader({ fileKinds: getFileKinds }),
    pluginHost,
    declaredCommands?.whenContext,
    () => declaredCommands?.contributions.handlers.value,
  )
  const { getPaletteEntries, getCommandNote, pluginWays } = plugins

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

  /** Runs the command of a chosen action over what it was chosen on, with the text typed for it. */
  const runChoice = (choice: ResultChoice): void =>
    runCommand(
      choice.command,
      choice.over ? { ...getTarget(), ...toResultTarget(choice.over) } : getTarget(),
      choice.input || undefined,
      choice.values,
    )

  /**
   * Runs a command of the application over the front target with the values of its keybinding. The
   * command is not asked whether it is offered: the reveal of a note is offered only as the rows that
   * carry its values, and the condition of the key was judged when the key matched.
   */
  const runKeyWithValues = (command: string, values: Values): void => {
    const invocation = withValues(createInvocation(command, getTarget()), values)
    runInBackground(runInvocation(invocation, commandDeps), (error) =>
      writeMessage(formatError(error), 'error'),
    )
  }

  /** Runs the command of the application a key is bound to, with the values of the keybinding. */
  const runAppKey = (command: string, values: Values): void => {
    if (command === COMMAND.searchVault) {
      commands.palette.setOpen(false)
      palette.setOpen(!palette.open.value)
    } else if (command === COMMAND.openCommands) {
      palette.setOpen(false)
      commands.palette.setOpen(!commands.palette.open.value)
    } else if (Object.keys(values).length > 0) runKeyWithValues(command, values)
    else runCommand(command, getTarget())
  }

  if (pluginHost && declaredCommands?.contributions.keybindings) {
    syncKeybindings(pluginHost, declaredCommands.contributions.keybindings, settings.state.plugins)
  }

  /** The one handler of keys: the tabs first, then the registry of the keybindings. */
  const onKeyDown = (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (windowTabs.panes.onKeyPress(event)) return event.preventDefault()
    const getNote = () => getCommandNote(getTarget())
    const keyContext = { target: toRunTarget(getTarget()), runApp: runAppKey }
    if (pluginHost?.handleShortcut(event, getNote, keyContext)) event.preventDefault()
  }

  return {
    palette,
    commands,
    commandDeps,
    pluginRuns: { items, runCommandToEnd },
    runCommand,
    runChoice,
    onKeyDown,
    pluginWays,
  }
}
