/**
 * Command palette, keyboard shortcut dispatch, and command execution.
 */

import { formatErrorMessage } from '@numen/core-client'
import { runInBackground } from '@numen/ui'
import {
  isChord,
  findChordCommand,
  runInvocation,
  useCommandPalette,
  useSearch,
  ANSWER_WORDS,
  type CommandDeps,
  type CommandTarget,
  type Notes,
  type NoteLookup,
  type PaletteLists,
  type RunSupport,
  type VaultRef,
} from '@/features/command-palette'
import { createCommandDeps } from './commandDeps'
import { usePluginCommands } from './usePluginCommands'
import { watchCommandLists } from './watchCommandLists'
import type { TabsHold } from './leaveVault'
import type { DeclaredCommands } from './declaredCommands'
import type { FileMenuCommands } from './fileMenuCommands'
import type { IndexCoverage } from '@/shared/notices/coverage'
import type { MessageLog, MessageWriter } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'
import { WORDS } from '@/shared/words'
import { createPluginSettingCommands } from '../window/useSettings'
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
  kept: PaletteLists
  runs: RunSupport
  coverage: () => IndexCoverage
  making: NoteCreator
  setVaultName: (vault: VaultRef) => void
  reload: () => void
  tabsHold?: TabsHold
  reached: Notes
  appearance: { chooseItem: (item: string) => Promise<void> | void }
  oneName: { choose: (item: string) => Promise<void> | void }
  rest: {
    writeSettings: (content: readonly SettingEdit[]) => Promise<void> | void
    getModelsAt?: (at: readonly string[]) => readonly import('@/entities/settings').Model[]
    plugins?: import('vue').Ref<readonly import('@/entities/settings').PluginInfo[]>
  }
  /** Shows a note in the plugin that displays notes. */
  travel: (path: string) => Promise<void> | void
  /** Every tab standing on a path moved to another. */
  leavePath: (from: string, to: string) => Promise<void> | void
  getOpeningNote: () => string
  writeMessage: MessageWriter
  pluginHost?: import('@numen/plugin-host').PluginHost | undefined
  /** The commands the enabled plugins declare, read as the window starts and each time the palette opens. */
  declaredCommands?: DeclaredCommands
  /** The menu the text versions of the file in front make, read each time the palette opens. */
  fileMenuCommands?: FileMenuCommands
}

export function useCommands(options: CommandsDepsOptions) {
  const { core, words, windowTabs, getTarget, knows, kept, runs, coverage, writeMessage } = options
  const { pluginHost } = options

  const {
    getPaletteEntries,
    findNoteDestination,
    getCommandNote,
    showNote,
    getNoteCommands,
    pluginWays,
  } = usePluginCommands(core, pluginHost)

  const palette = useSearch(core, words, {
    coverage,
    getTravel: () => findNoteDestination()?.title ?? '',
  })
  const commands = useCommandPalette(core, words, getTarget, knows, kept, runs, undefined, () => [
    ...createPluginSettingCommands(options.rest.plugins?.value ?? []),
    ...(options.declaredCommands?.paletteCommands.value ?? []),
    ...(options.fileMenuCommands?.paletteCommands.value ?? []),
    ...getPaletteEntries(),
  ])

  watchCommandLists({ ...options, plugins: options.rest.plugins }, commands.open, getTarget)

  const commandDeps: CommandDeps = createCommandDeps({
    ...options,
    search: () => {
      commands.setOpen(false)
      palette.setOpen(true)
    },
  })

  const reportError = (error: unknown) => writeMessage(formatErrorMessage(error), 'error')

  const runCommand = (id: string, target: CommandTarget) => {
    if (pluginHost?.hasCommand(id)) {
      commands.setOpen(false)
      runInBackground(
        getCommandNote(target.path, target.title).then((note) =>
          pluginHost.executeCommand(id, note),
        ),
        reportError,
      )
      return
    }
    const invocation = commands.startCommand(id, target)
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

  const onKeyDown = (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (windowTabs.onKeyPress(event)) return event.preventDefault()
    const getNote = () => getCommandNote(getTarget().path, getTarget().title)
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
    onKeyDown,
    pluginWays,
    getNoteCommands,
    showNote,
  }
}
