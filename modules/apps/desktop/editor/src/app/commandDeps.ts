/**
 * Command execution dependencies assembly for palette and shortcut actions.
 */
import type { Ref } from "vue"
import { vaults, type VaultCore } from "./vault"
import { running } from "../shared/artifacts"
import { WORDS as cardWords } from "../entities/deck/words"
import type { CommandDeps, Notes } from "../features/command-palette/deps"
import type { RunSupport } from "../features/command-palette/runs"
import type { MessageWriter } from "../shared/notices/messages"
import type { createFileCreators } from "../entities/tab/makers"
import type { useWindowTabs } from "../entities/tab/windowTabs"
import type { NoteCreator } from "../widgets/note-editor/maker"
import type { VaultRef } from "../features/command-palette/target"

export interface CommandDepsOptions {
  core: VaultCore
  held: ReturnType<typeof useWindowTabs>
  making: NoteCreator
  made: ReturnType<typeof createFileCreators>
  shown: Ref<VaultRef>
  reloads: () => void
  carrying: (path: string) => Promise<void>
  reached: Notes
  opensPreset: (path: string) => Promise<void>
  dressed: { chooses: (item: string) => Promise<void> | void }
  oneName: { chooses: (item: string) => Promise<void> | void }
  hungParts: { chooses: (item: string) => Promise<void> | void; choosesCount: (item: string) => Promise<void> | void }
  recorded: { deleted?: (path: string) => void; onDelete?: (path: string) => void }
  pointed: { deleted?: (path: string) => void; onDelete?: (path: string) => void }
  files: () => { revealPath: (path: string) => void }
  plexes: () => { travel: (path: string) => Promise<void> | void; leaves: (from: string, to: string) => Promise<void> | void }
  agents: () => { askQuestion: (text: string) => Promise<void> | void }
  opening: () => string
  runs: RunSupport
  told: MessageWriter
  searches?: () => void
}

export function createCommandDeps(options: CommandDepsOptions): CommandDeps {
  const {
    core,
    held,
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
    runs,
    told,
  } = options

  return {
    files: {
      makes: (title, from, seat) => making.createWithTitle(title, from, seat),
      renames: (path, title) => core.rename(path, title),
      removes: (path, destroy) => core.remove(path, destroy),
      moves: (from, to) => core.move(from, to),
      makesFolder: (path) => core.createFolder(path),
    },
    runs: {
      getArtifactStates: (path) => running.getArtifactStates(path),
      createArtifact: async (path, of) => {
        const outcome = await running.createArtifact(path, of)
        void carrying(path)
        return outcome
      },
      fetchArtifact: async (path) => {
        const outcome = await running.fetchArtifact(path)
        void carrying(path)
        return outcome
      },
      correctArtifact: async (path) => {
        const outcome = await running.correctArtifact(path)
        void carrying(path)
        return outcome
      },
      deleteTranscript: async (path) => {
        const able = await running.deleteTranscript(path)
        if (able) {
          ;(recorded.onDelete ?? recorded.deleted)?.(path)
          ;(pointed.onDelete ?? pointed.deleted)?.(path)
        }
        void carrying(path)
        return able
      },
      deleteCopy: async (path) => {
        const able = await running.deleteCopy(path)
        if (able) {
          void carrying(path)
        }
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
      reveals: (path) => void files().revealPath(path),
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
      asks: (text) => void agents().askQuestion(text),
      searches: options.searches ?? (() => {}),
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
}
