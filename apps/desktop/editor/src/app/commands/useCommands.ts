/**
 * Command palette, keyboard shortcut dispatch, and command execution.
 */

import { formatErrorMessage } from '@numen/core-client'
import { SLOT, type SlotHost } from '@numen/plugin-host'
import { runInBackground } from '@numen/ui'
import {
  isChord,
  findChordCommand,
  runInvocation,
  createInvocation,
  useCommandPalette,
  useSearch,
  ANSWER_WORDS,
  type Command,
  type CommandDeps,
  type CommandTarget,
  type ResultChoice,
  type SearchMenuDeps,
  type Notes,
  type NoteLookup,
  type PaletteLists,
  type RunSupport,
  type VaultRef,
} from '@/features/command-palette'
import { createCommandDeps } from './commandDeps'
import { createCommandNoteReader, usePluginCommands } from './usePluginCommands'
import { createItemLists } from './pluginItems'
import { createCommandItems } from './commandItems'
import { watchCommandLists } from './watchCommandLists'
import type { TabsHold } from './leaveVault'
import type { DeclaredCommands } from './declaredCommands'
import type { FileMenuCommands } from './fileMenuCommands'
import type { IndexCoverage } from '@/shared/notices/coverage'
import type { MessageLog, MessageWriter } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'
import { WORDS } from '@/shared/words'
import type { NoteCreator } from '@/entities/note'
import type { VaultCore } from '../vault'

type Words = typeof WORDS

import type { SettingEdit } from '@/entities/settings'

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
  /** The adapter that writes the row chosen in the step of the command of a setting. */
  settingPalette?: {
    getCommands: () => readonly Command[]
    choose: (command: string, item: string) => Promise<void>
  }
  rest: {
    writeSettings: (content: readonly SettingEdit[]) => Promise<void> | void
    getModelsAt?: (at: readonly string[]) => readonly import('@/entities/settings').Model[]
    plugins?: import('vue').Ref<readonly import('@/entities/settings').PluginInfo[]>
    getCommittedSetting?: (path: readonly string[]) => unknown
  }
  /** Shows a note in the plugin that displays notes. */
  travel: (path: string) => Promise<void> | void
  /** Every tab standing on a path moved to another. */
  leavePath: (from: string, to: string) => Promise<void> | void
  getOpeningNote: () => string
  writeMessage: MessageWriter
  pluginHost?: import('@numen/plugin-host').PluginHost | undefined
  /** The slots of the window, of which the palette draws the menu of a search result. */
  slots?: SlotHost
  /** The commands of the window made as it runs, such as those over a tab. */
  getAppCommands?: () => readonly Command[]
  /** The commands the enabled plugins declare, read as the window starts and each time the palette opens. */
  declaredCommands?: DeclaredCommands
  /** The menu the text versions of the file in front make, read each time the palette opens. */
  fileMenuCommands?: FileMenuCommands
}

/** The menu of a search result: the entries of its slot over the note of the result. */
const createSearchMenu = (slots: SlotHost | undefined): SearchMenuDeps => ({
  getEntries: (target) => slots?.getEntries(SLOT.paletteSearchResult, target) ?? [],
})

/** What a chosen result is over, replacing the tab and the note of the window. */
const toResultTarget = (place: NonNullable<ResultChoice['over']>) => ({
  tab: '',
  kind: null,
  path: place.path,
  file: place.file,
  title: place.title,
  ...(place.line === undefined ? {} : { line: place.line }),
  ...(place.selector === undefined ? {} : { selector: place.selector }),
})

/** The commands the palette lists beside its own table, as the window and its plugins make them. */
const listExtraCommands = (
  options: CommandsDepsOptions,
  pluginEntries: readonly Command[],
): readonly Command[] => [
  ...(options.getAppCommands?.() ?? []),
  ...(options.settingPalette?.getCommands() ?? []),
  ...(options.declaredCommands?.paletteCommands.value ?? []),
  ...(options.fileMenuCommands?.paletteCommands.value ?? []),
  ...pluginEntries,
]

/** The choices of the commands of the palette, of the window and of the plugins, and their runs. */
export function createPaletteItems(
  options: Pick<CommandsDepsOptions, 'getAppCommands' | 'pluginHost' | 'getTarget'>,
  getCommandNote: ReturnType<typeof createCommandNoteReader>,
  getCommandDeps: () => CommandDeps,
) {
  const { pluginHost } = options
  return createCommandItems({
    getAppCommands: () => options.getAppCommands?.() ?? [],
    pluginHost,
    getTarget: options.getTarget,
    runApp: async (id, target, choice) => {
      await runInvocation(createInvocation(id, target, choice), getCommandDeps(), ANSWER_WORDS)
    },
    runPlugin: async (id, target, choice) => {
      await pluginHost?.executeCommand(id, await getCommandNote(target), choice)
    },
  })
}

export function useCommands(options: CommandsDepsOptions) {
  const { core, words, windowTabs, getTarget, knows, runs, coverage, writeMessage } = options
  const { pluginHost } = options

  const { getPaletteEntries, getCommandNote, getNoteCommands, pluginWays } = usePluginCommands(
    core,
    pluginHost,
  )

  const palette = useSearch(core, words, {
    coverage,
    menu: createSearchMenu(options.slots),
    getTarget,
  })
  const choices = createPaletteItems(options, getCommandNote, () => commandDeps)
  const { lists, resetWhenClosed } = createItemLists(options.paletteLists, choices, getTarget)
  const extra = () => listExtraCommands(options, getPaletteEntries())
  const commands = useCommandPalette(core, words, getTarget, knows, lists, runs, undefined, extra)

  watchCommandLists({ ...options, plugins: options.rest.plugins }, commands.open, getTarget)
  resetWhenClosed(commands.open)

  const commandDeps: CommandDeps = createCommandDeps({
    ...options,
    search: () => {
      commands.setOpen(false)
      palette.setOpen(true)
    },
  })

  const reportError = (error: unknown) => writeMessage(formatErrorMessage(error), 'error')

  const runCommand = (id: string, target: CommandTarget, input?: string) => {
    if (pluginHost?.hasCommand(id)) {
      commands.setOpen(false)
      runInBackground(
        getCommandNote(target).then((note) => pluginHost.executeCommand(id, note)),
        reportError,
      )
      return
    }
    const invocation =
      input === undefined
        ? commands.startCommand(id, target)
        : commands.startCommand(id, target, input)
    if (invocation) {
      runInBackground(runInvocation(invocation, commandDeps, ANSWER_WORDS), reportError)
      return
    }
    if (commands.open.value) {
      palette.setOpen(false)
      return
    }
    writeMessage(commands.getObjection(id, target), 'error')
  }

  /** Runs the command of a chosen action over what it was chosen on, with the text typed for it. */
  const runChoice = (choice: ResultChoice): void =>
    runCommand(
      choice.command,
      choice.over ? { ...getTarget(), ...toResultTarget(choice.over) } : getTarget(),
      choice.input || undefined,
    )

  const onKeyDown = (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (windowTabs.onKeyPress(event)) return event.preventDefault()
    const getNote = () => getCommandNote(getTarget())
    if (pluginHost?.handleShortcut(event, getNote)) return event.preventDefault()
    if (!isChord(event)) return
    const command = findChordCommand(event.key.toLowerCase(), event.shiftKey)
    if (!command) return
    event.preventDefault()
    runCommand(command, getTarget())
  }

  return {
    palette,
    commands,
    commandDeps,
    runCommand,
    runChoice,
    onKeyDown,
    pluginWays,
    getNoteCommands,
  }
}
