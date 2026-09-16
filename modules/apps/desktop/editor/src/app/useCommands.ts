/**
 * Command palette, keyboard shortcut dispatch, and command execution.
 */

import {
  isChord,
  commandFor,
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
import type { IndexCoverage } from '@/shared/notices/coverage'
import type { MessageLog, MessageWriter } from '@/shared/notices/messages'
import type { createFileCreators } from '@/entities/tab'
import type { useWindowTabs } from '@/entities/tab'
import { WORDS } from '@/shared/words'
import type { NoteCreator } from '@/pages/note-editor'
import type { VaultCore } from './vault'

type Words = typeof WORDS

export interface CommandsDepsOptions {
  core: VaultCore
  words: Words
  log: MessageLog
  held: ReturnType<typeof useWindowTabs>
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
  dressed: { chooseItem: (item: string) => Promise<void> | void }
  oneName: { choose: (item: string) => Promise<void> | void }
  hungParts: {
    choose: (item: string) => Promise<void> | void
    chooseCount: (item: string) => Promise<void> | void
  }
  recorded: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  pointed: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  files: () => { revealPath: (path: string) => void }
  plexes: () => {
    travel: (path: string) => Promise<void> | void
    leavePath: (from: string, to: string) => Promise<void> | void
  }
  agents: () => { askQuestion: (text: string) => Promise<void> | void }
  getOpeningNote: () => string
  writeMessage: MessageWriter
}

export function useCommands(options: CommandsDepsOptions) {
  const { core, words, held, getTarget, knows, kept, runs, coverage, writeMessage } = options

  const palette = useSearch(core, words, { coverage })
  const commands = useCommandPalette(core, words, getTarget, knows, kept, runs)

  const commandDeps: CommandDeps = createCommandDeps({
    ...options,
    search: () => {
      commands.setOpen(false)
      palette.setOpen(true)
    },
  })

  const runCommand = (id: string, at: CommandTarget) => {
    const invocation = commands.startCommand(id, at)
    if (invocation) return void runInvocation(invocation, commandDeps, ANSWER_WORDS)
    if (commands.open.value) return void palette.setOpen(false)
    writeMessage(commands.getObjection(id, at), 'error')
  }

  const onKeyDown = (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (held.onKeyPress(event)) return event.preventDefault()
    if (!isChord(event)) return
    const command = commandFor(event.key.toLowerCase(), event.shiftKey)
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
