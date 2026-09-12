/**
 * Command palette, keyboard shortcut dispatch, and command execution.
 */
import type { Ref } from 'vue'
import {
  isChord,
  commandFor,
  runInvocation,
  useCommandPalette,
  useSearch,
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
  where: () => CommandTarget
  knows: NoteLookup
  kept: PaletteLists
  runs: RunSupport
  coverage: () => IndexCoverage
  making: NoteCreator
  made: ReturnType<typeof createFileCreators>
  shown: Ref<VaultRef>
  reloads: () => void
  loadArtifactStates: (path: string) => Promise<void>
  reached: Notes
  opensPreset: (path: string) => Promise<void>
  dressed: { chooseItem: (item: string) => Promise<void> | void }
  oneName: { choose: (item: string) => Promise<void> | void }
  hungParts: { choose: (item: string) => Promise<void> | void; chooseCount: (item: string) => Promise<void> | void }
  recorded: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  pointed: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  files: () => { revealPath: (path: string) => void }
  plexes: () => { travel: (path: string) => Promise<void> | void; leavePath: (from: string, to: string) => Promise<void> | void }
  agents: () => { askQuestion: (text: string) => Promise<void> | void }
  opening: () => string
  told: MessageWriter
}

export function useCommands(options: CommandsDepsOptions) {
  const {
    core,
    words,
    held,
    where,
    knows,
    kept,
    runs,
    coverage,
    told,
  } = options

  const palette = useSearch(core, words, { coverage })
  const commands = useCommandPalette(core, words, where, knows, kept, runs)

  const doing: CommandDeps = createCommandDeps({
    ...options,
    search: () => {
      ;commands.setOpen(false)
      ;palette.setOpen(true)
    },
  })

  const runCommand = (id: string, at: CommandTarget) => {
    const invocation = commands.startCommand(id, at)
    if (invocation) return void runInvocation(invocation, doing, words)
    if (commands.open.value) return void palette.setOpen(false)
    told(commands.getObjection(id, at), 'error')
  }

  const onKeyDown = (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (held.onKeyPress(event)) return event.preventDefault()
    if (!isChord(event)) return
    const command = commandFor(event.key.toLowerCase(), event.shiftKey)
    if (!command) return
    event.preventDefault()
    runCommand(command, where())
  }
  const asked = onKeyDown

  return {
    palette,
    commands,
    doing,
    runCommand,
    asked,
    onKeyDown,
  }
}
