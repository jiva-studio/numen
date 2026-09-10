/**
 * Command palette, keyboard shortcut dispatch, and command execution.
 */
import type { Ref } from 'vue'
import { chorded, commandFor } from '../shared/command/chords'
import { does } from '../shared/command/handlers'
import { commandPalette } from '../shared/command/palette'
import { search } from '../shared/command/search'
import type { CommandTarget, VaultRef } from '../shared/command/target'
import { vaults } from './vault'
import { running } from '../shared/artifacts'
import { WORDS as cardWords } from '../shared/flashcards/words'
import type { NoteLookup, PaletteLists } from '../shared/command/lists'
import type { CommandDeps, Notes } from '../shared/command/deps'
import type { RunSupport } from '../shared/command/runs'
import type { IndexCoverage } from '../shared/notices/coverage'
import type { MessageLog, MessageWriter } from '../shared/notices/messages'
import type { fileMakers } from '../shared/tabs/makers'
import type { windowTabs } from '../shared/tabs/windowTabs'
import { WORDS } from '../shared/words'
import type { noteMaker } from '../note-tab/maker'
import type { VaultCore } from './vault'

type Words = typeof WORDS

export interface CommandsDepsOptions {
  core: VaultCore
  words: Words
  log: MessageLog
  held: ReturnType<typeof windowTabs>
  where: () => CommandTarget
  knows: NoteLookup
  kept: PaletteLists
  runs: RunSupport
  coverage: () => IndexCoverage
  making: ReturnType<typeof noteMaker>
  made: ReturnType<typeof fileMakers>
  shown: Ref<VaultRef>
  reloads: () => void
  carrying: (path: string) => Promise<void>
  reached: Notes
  opensPreset: (path: string) => Promise<void>
  dressed: { chooses: (item: string) => Promise<void> | void }
  oneName: { chooses: (item: string) => Promise<void> | void }
  hungParts: { chooses: (item: string) => Promise<void> | void; choosesCount: (item: string) => Promise<void> | void }
  recorded: { deleted: (path: string) => void }
  pointed: { deleted: (path: string) => void }
  files: () => { reveals: (path: string) => void }
  plexes: () => { travel: (path: string) => Promise<void> | void; leaves: (from: string, to: string) => Promise<void> | void }
  agents: () => { asks: (text: string) => void }
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
    making,
    made,
    shown,
    reloads,
    carrying,
    reached,
    opensPreset,
    dressed,
    oneName,
    hungParts,
    recorded,
    pointed,
    files,
    plexes,
    agents,
    opening,
    told,
  } = options

  const palette = search(core, words, { coverage })
  const commands = commandPalette(core, words, where, knows, kept, runs)

  const doing: CommandDeps = {
    files: {
      makes: (title, from, seat) => making.calls(title, from, seat),
      renames: (path, title) => core.rename(path, title),
      removes: (path, destroy) => core.remove(path, destroy),
      moves: (from, to) => core.move(from, to),
      makesFolder: (path) => core.makeFolder(path),
    },
    runs: {
      carries: (path) => running.carries(path),
      makes: async (path, of) => {
        const outcome = await running.makes(path, of)
        void carrying(path)
        return outcome
      },
      fetches: async (path) => {
        const outcome = await running.fetches(path)
        void carrying(path)
        return outcome
      },
      corrects: async (path) => {
        const outcome = await running.corrects(path)
        void carrying(path)
        return outcome
      },
      deletesTranscript: async (path) => {
        const able = await running.deletesTranscript(path)
        if (able) {
          recorded.deleted(path)
          pointed.deleted(path)
        }
        void carrying(path)
        return able
      },
      deletesCopy: async (path) => {
        const able = await running.deletesCopy(path)
        void carrying(path)
        return able
      },
    },
    makers: {
      ...made,
      stencils: (folder, name) => made.stencils(folder, name, [cardWords.newField]),
    },
    vaults: {
      ...vaults,
      calls: (vault) => (shown.value = vault),
      reloads,
    },
    goes: {
      reveals: (path) => void files().reveals(path),
      travel: async (path) => {
        await plexes().travel(path)
      },
      leaves: async (from, to) => {
        await plexes().leaves(from, to)
      },
      opening,
      opens: (kind) => void held.opens(kind),
      preset: (path) => opensPreset(path),
      closes: (tab) => held.drops(tab),
      asks: (text) => void agents().asks(text),
      searches: () => {
        commands.shows(false)
        palette.shows(true)
      },
    },
    settings: {
      appearance: async (chosen) => {
        await dressed.chooses(chosen)
      },
      syncing: async (chosen) => {
        await oneName.chooses(chosen)
      },
      hanging: async (chosen) => {
        await hungParts.chooses(chosen)
      },
      parts: async (chosen) => {
        await hungParts.choosesCount(chosen)
      },
    },
    notes: reached,
    runSupport: runs,
    copies: (path) => void navigator.clipboard?.writeText(path),
    says: told,
  }

  const carries = (id: string, at: CommandTarget) => {
    const invocation = commands.asks(id, at)
    if (invocation) return void does(invocation, doing, words)
    if (commands.open.value) return palette.shows(false)
    told(commands.refused(id, at), 'refusal')
  }

  const asked = (event: KeyboardEvent) => {
    if (event.defaultPrevented) return
    if (held.presses(event)) return event.preventDefault()
    if (!chorded(event)) return
    const command = commandFor(event.key.toLowerCase(), event.shiftKey)
    if (!command) return
    event.preventDefault()
    carries(command, where())
  }

  return {
    palette,
    commands,
    doing,
    carries,
    asked,
  }
}
