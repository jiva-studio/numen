/**
 * Command palette, keyboard shortcut dispatch, and command execution.
 */

import { formatErrorMessage } from '@numen/core-client'
import { SLOT, type PluginHost, type SlotHost } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import {
  runInvocation,
  createInvocation,
  useCommandPalette,
  useSearch,
  ANSWER_WORDS,
  type CommandDeps,
  type ResultChoice,
  type SearchMenuDeps,
  type Notes,
  type NoteLookup,
  type PaletteLists,
} from '@/features/command-palette'
import type {
  Command,
  CommandInvocation,
  CommandTarget,
  RunSupport,
  VaultRef,
} from '@/entities/command'
import { createCommandDeps } from './commandDeps'
import { COMMAND } from '@/shared/commandIds'
import { createCommandNoteReader, usePluginCommands } from './usePluginCommands'
import { createItemLists } from './pluginItems'
import { toRunTarget } from './runTarget'
import { createCommandItems } from './commandItems'
import { watchCommandLists } from './watchCommandLists'
import { syncKeybindings } from './syncKeybindings'
import type { TabsHold } from './leaveVault'
import type { DeclaredCommands } from './declaredCommands'
import type { IndexCoverage } from '@/shared/notices/coverage'
import type { MessageLog, MessageWriter } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'
import type { WORDS } from '@/shared/words'
import type { NoteCreator } from '@/entities/note'
import { getFileKinds } from '@/entities/file'
import type { VaultCore } from '../vault'
import { getVaultList } from '../window/vaultList'

type Words = typeof WORDS

import type { Model, PluginInfo, SettingEdit } from '@/entities/settings'
import type { Ref } from 'vue'

export interface CommandsDepsOptions {
  core: VaultCore
  words: Words
  log: MessageLog
  windowTabs: ReturnType<typeof useWindowTabs>
  getTarget: () => CommandTarget
  knows: NoteLookup
  paletteLists: PaletteLists
  runs: RunSupport
  coverage: () => IndexCoverage
  making: NoteCreator
  setVaultName: (vault: VaultRef) => void
  reload: () => void
  tabsHold?: TabsHold
  reached: Notes
  settingPalette?: {
    getCommands: () => readonly Command[]
    choose: (command: string, item: string) => Promise<void>
  }
  rest: {
    writeSettings: (content: readonly SettingEdit[]) => Promise<void> | void
    getModelsAt?: (at: readonly string[]) => readonly Model[]
    plugins?: Ref<readonly PluginInfo[]>
    getCommittedSetting?: (path: readonly string[]) => unknown
  }
  travel: (path: string) => Promise<void> | void
  leavePath: (from: string, to: string) => Promise<void> | void
  getOpeningNote: () => string
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
  getCommandDeps: () => CommandDeps,
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
      await runInvocationToEnd(invocation, getCommandDeps())
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

type Values = Readonly<Record<string, string>>

const getValues = (values: Values | undefined): { values?: Values } =>
  values && Object.keys(values).length > 0 ? { values } : {}

const withValues = (
  invocation: CommandInvocation,
  values: Values | undefined,
): CommandInvocation => ({
  ...invocation,
  ...getValues(values),
})

async function runInvocationToEnd(
  invocation: CommandInvocation,
  deps: CommandDeps,
): Promise<string> {
  const errors: string[] = []
  const writeMessage: MessageWriter = (text, severity) => {
    if (severity === 'error') errors.push(text)
    else deps.writeMessage(text, severity)
  }
  const made = await runInvocation(invocation, { ...deps, writeMessage }, ANSWER_WORDS)
  if (errors[0] !== undefined) throw new Error(errors[0])
  return made
}

/** What the runs of the window go through: the palettes, the plugin host and the deps of a command. */
interface CommandRunnerParts {
  readonly commands: ReturnType<typeof useCommandPalette>
  readonly palette: { setOpen(isOpen: boolean): void }
  readonly pluginHost: CommandsDepsOptions['pluginHost']
  readonly getCommandNote: ReturnType<typeof createCommandNoteReader>
  readonly getCommandDeps: () => CommandDeps
  readonly writeMessage: MessageWriter
}

/** How a command is run: with the text typed for it, and whether its caller awaits it. */
interface RunWay {
  readonly input?: string | undefined
  readonly values?: Values | undefined
  readonly isAwaited: boolean
}

/**
 * The two runs of a command of the window. The run of the palette and the shortcuts writes its
 * error in the window. The run to its end, which a plugin awaits through `context.commands.run`,
 * writes nothing: it settles when the run ends with the path of the note or folder it made, empty
 * where it made none, and rejects with its error.
 */
export function createCommandRunner(parts: CommandRunnerParts) {
  const { commands, palette, pluginHost, getCommandNote } = parts

  /** The run of the command over the target, or the objection to it where it cannot start. */
  const startRun = (id: string, target: CommandTarget, way: RunWay): Promise<unknown> | string => {
    if (pluginHost && (pluginHost.hasCommand(id) || pluginHost.isSharedCommand(id))) {
      commands.setOpen(false)
      return getCommandNote(target).then((note) => {
        const run = { ...getValues(way.values), ...(way.isAwaited ? { isAwaited: true } : {}) }
        return Object.keys(run).length > 0
          ? pluginHost.executeCommand(id, note, run)
          : pluginHost.executeCommand(id, note)
      })
    }
    const { values } = getValues(way.values)
    const invocation = values
      ? commands.startCommand(id, target, way.input, values)
      : way.input === undefined
        ? commands.startCommand(id, target)
        : commands.startCommand(id, target, way.input)
    if (invocation) {
      const deps = parts.getCommandDeps()
      const run = withValues(invocation, way.values)
      return way.isAwaited ? runInvocationToEnd(run, deps) : runInvocation(run, deps, ANSWER_WORDS)
    }
    if (commands.open.value) {
      palette.setOpen(false)
      return Promise.resolve()
    }
    return commands.getObjection(id, target)
  }

  return {
    runCommand: (id: string, target: CommandTarget, input?: string, values?: Values): void => {
      const run = startRun(id, target, { input, values, isAwaited: false })
      if (typeof run === 'string') parts.writeMessage(run, 'error')
      else runInBackground(run, (error) => parts.writeMessage(formatErrorMessage(error), 'error'))
    },
    runCommandToEnd: async (
      id: string,
      target: CommandTarget,
      values?: Values,
    ): Promise<string> => {
      const run = startRun(id, target, { values, isAwaited: true })
      if (typeof run === 'string') throw new Error(run)
      const made = await run
      return typeof made === 'string' ? made : ''
    },
  }
}

function useWindowCommandPalette(
  options: CommandsDepsOptions,
  lists: PaletteLists,
  listExtras: () => readonly Command[],
) {
  const { core, words, getTarget, knows, runs } = options
  return useCommandPalette(
    { ...core, vaults: getVaultList },
    words,
    getTarget,
    knows,
    lists,
    runs,
    undefined,
    listExtras,
  )
}

export function useCommands(options: CommandsDepsOptions) {
  const { core, words, windowTabs, getTarget, coverage, writeMessage } = options
  const { pluginHost, declaredCommands } = options
  const plugins = usePluginCommands(
    options.readCommandNote ?? createCommandNoteReader({ fileKinds: getFileKinds }),
    pluginHost,
    declaredCommands?.whenContext,
    () => declaredCommands?.contributions.handlers.value,
  )
  const { getPaletteEntries, getCommandNote, pluginWays } = plugins

  const palette = useSearch(core, words, {
    coverage,
    menu: createSearchMenu(options.slots),
    getTarget,
  })
  const items = createPaletteItems(options, getCommandNote, () => commandDeps)
  const { lists, resetWhenClosed, removeEmptyCommands } = createItemLists(items, options)
  const listExtras = () => removeEmptyCommands(listExtraCommands(options, getPaletteEntries()))
  const commands = useWindowCommandPalette(options, lists, listExtras)

  watchCommandLists({ ...options, plugins: options.rest.plugins }, commands.open)
  resetWhenClosed(commands.open)

  const commandDeps: CommandDeps = createCommandDeps({
    ...options,
    search: () => {
      commands.setOpen(false)
      palette.setOpen(true)
    },
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
    runInBackground(runInvocation(invocation, commandDeps, ANSWER_WORDS), (error) =>
      writeMessage(formatErrorMessage(error), 'error'),
    )
  }

  /** Runs the command of the application a key is bound to, with the values of the keybinding. */
  const runAppKey = (command: string, values: Values): void => {
    if (command === COMMAND.searchVault) {
      commands.setOpen(false)
      palette.setOpen(!palette.open.value)
    } else if (command === COMMAND.openCommands) {
      palette.setOpen(false)
      commands.setOpen(!commands.open.value)
    } else if (Object.keys(values).length > 0) runKeyWithValues(command, values)
    else runCommand(command, getTarget())
  }

  if (pluginHost && declaredCommands?.contributions.keybindings) {
    syncKeybindings(pluginHost, declaredCommands.contributions.keybindings, options.rest.plugins)
  }

  /** The one handler of keys: the tabs first, then the registry of the keybindings. */
  const onKeyDown = (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (windowTabs.onKeyPress(event)) return event.preventDefault()
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
