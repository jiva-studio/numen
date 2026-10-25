/**
 * Command palette, keyboard shortcut dispatch, and command execution.
 */

import { watch } from 'vue'
import { formatErrorMessage } from '@numen/wire'
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
import type { DeclaredCommands } from './declaredCommands'
import type { IndexCoverage } from '@/shared/notices/coverage'
import type { MessageLog, MessageWriter } from '@/shared/notices/messages'
import type { createFileCreators } from '@/entities/tab'
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
  made: ReturnType<typeof createFileCreators>
  setVaultName: (vault: VaultRef) => void
  reload: () => void
  loadArtifactStates: (path: string) => Promise<void>
  reached: Notes
  openPreset: (path: string) => Promise<void>
  appearance: { chooseItem: (item: string) => Promise<void> | void }
  oneName: { choose: (item: string) => Promise<void> | void }
  nesting: {
    choose: (item: string) => Promise<void> | void
    chooseCount: (item: string) => Promise<void> | void
  }
  rest: {
    writeSettings: (content: readonly SettingEdit[]) => Promise<void> | void
    getModelsAt?: (at: readonly string[]) => readonly import('@/entities/settings').Model[]
    plugins?: import('vue').Ref<readonly import('@/entities/settings').PluginInfo[]>
  }
  recorded: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  pointed?: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  files: () => { revealPath: (path: string) => void }
  /** A note put in front of the person in the plex they were last in. */
  travel: (path: string) => Promise<void> | void
  /** Every tab standing on a path moved to another. */
  leavePath: (from: string, to: string) => Promise<void> | void
  agents: () => { askQuestion: (text: string) => Promise<void> | void }
  getOpeningNote: () => string
  writeMessage: MessageWriter
  pluginHost?: import('@/entities/plugin').PluginHost | undefined
  /** The commands the enabled plugins declare, read again each time the palette opens. */
  declaredCommands?: DeclaredCommands
}

export function useCommands(options: CommandsDepsOptions) {
  const { core, words, windowTabs, getTarget, knows, kept, runs, coverage, writeMessage } = options

  const tabRuns: RunSupport = {
    canRun(run: string) {
      if (run === 'plex' || run === 'files' || run === 'agent') {
        if (!windowTabs.hasKind(run)) return false
      }
      return runs.canRun(run)
    },
    cannotRun(run: string) {
      runs.cannotRun(run)
    },
  }

  const palette = useSearch(core, words, { coverage })
  const commands = useCommandPalette(
    core,
    words,
    getTarget,
    knows,
    kept,
    tabRuns,
    undefined,
    () => [
      ...createPluginSettingCommands(options.rest.plugins?.value ?? []),
      ...(options.declaredCommands?.paletteCommands.value ?? []),
      ...(options.pluginHost?.getPaletteCommands().map((cmd) => ({
        id: cmd.id,
        text: cmd.title,
        group: 'window' as const,
        isOffered: () => cmd.isOffered(),
      })) ?? []),
    ],
  )

  const { declaredCommands } = options
  if (declaredCommands) {
    watch(commands.open, (isOpen) => {
      if (!isOpen) return
      declaredCommands
        .load()
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    })
  }

  const commandDeps: CommandDeps = createCommandDeps({
    ...options,
    search: () => {
      commands.setOpen(false)
      palette.setOpen(true)
    },
  })

  const runCommand = (id: string, target: CommandTarget) => {
    if (options.pluginHost?.hasCommand(id)) {
      commands.setOpen(false)
      void options.pluginHost.executeCommand(id)
      return
    }
    const invocation = commands.startCommand(id, target)
    if (invocation) return void runInvocation(invocation, commandDeps, ANSWER_WORDS)
    if (commands.open.value) return void palette.setOpen(false)
    writeMessage(commands.getObjection(id, target), 'error')
  }

  const onKeyDown = (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (windowTabs.onKeyPress(event)) return event.preventDefault()
    if (options.pluginHost?.handleShortcut(event)) return event.preventDefault()
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
  }
}
