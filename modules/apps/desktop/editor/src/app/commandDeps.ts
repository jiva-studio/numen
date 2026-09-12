/**
 * Command execution dependencies assembly for palette and shortcut actions.
 */
import type { Ref } from 'vue'
import { vaults, type VaultCore } from './vault'
import { running } from '@/shared/artifacts'
import { WORDS as cardWords } from '@/entities/deck'
import type { CommandDeps, Notes, RunSupport, VaultRef } from '@/features/command-palette'
import type { MessageWriter } from '@/shared/notices/messages'
import type { createFileCreators } from '@/entities/tab'
import type { useWindowTabs } from '@/entities/tab'
import type { NoteCreator } from '@/pages/note-editor'

export interface CommandDepsOptions {
  core: VaultCore
  held: ReturnType<typeof useWindowTabs>
  making: NoteCreator
  made: ReturnType<typeof createFileCreators>
  shown: Ref<VaultRef>
  reload: () => void
  loadArtifactStates: (path: string) => Promise<void>
  reached: Notes
  openPreset: (path: string) => Promise<void>
  dressed: { chooseItem: (item: string) => Promise<void> | void }
  oneName: { choose: (item: string) => Promise<void> | void }
  hungParts: { choose: (item: string) => Promise<void> | void; chooseCount: (item: string) => Promise<void> | void }
  recorded: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  pointed: { reloadTranscript?: (path: string) => void; onDelete?: (path: string) => void }
  files: () => { revealPath: (path: string) => void }
  plexes: () => { travel: (path: string) => Promise<void> | void; leavePath: (from: string, to: string) => Promise<void> | void }
  agents: () => { askQuestion: (text: string) => Promise<void> | void }
  getOpeningNote: () => string
  runs: RunSupport
  writeMessage: MessageWriter
  search?: () => void
}

export function createCommandDeps(options: CommandDepsOptions): CommandDeps {
  const {
    core,
    held,
    making,
    made,
    shown,
    reload,
    loadArtifactStates,
    reached,
    openPreset,
    dressed,
    oneName,
    hungParts,
    recorded,
    pointed,
    files,
    plexes,
    agents,
    getOpeningNote,
    runs,
    writeMessage,
  } = options

  return {
    files: {
      createNote: (title, from, seat) => making.createWithTitle(title, from, seat),
      rename: (path, title) => core.rename(path, title),
      remove: (path, destroy) => core.remove(path, destroy),
      move: (from, to) => core.move(from, to),
      createFolder: (path) => core.createFolder(path),
    },
    runs: {
      getArtifactStates: (path) => running.getArtifactStates(path),
      createArtifact: async (path, of) => {
        const outcome = await running.createArtifact(path, of)
        void loadArtifactStates(path)
        return outcome
      },
      fetchArtifact: async (path) => {
        const outcome = await running.fetchArtifact(path)
        void loadArtifactStates(path)
        return outcome
      },
      correctArtifact: async (path) => {
        const outcome = await running.correctArtifact(path)
        void loadArtifactStates(path)
        return outcome
      },
      deleteTranscript: async (path) => {
        const able = await running.deleteTranscript(path)
        if (able) {
          ;(recorded.onDelete ?? recorded.reloadTranscript)?.(path)
          ;(pointed.onDelete ?? pointed.reloadTranscript)?.(path)
        }
        void loadArtifactStates(path)
        return able
      },
      deleteCopy: async (path) => {
        const able = await running.deleteCopy(path)
        if (able) {
          void loadArtifactStates(path)
        }
        return able
      },
    },
    makers: {
      ...made,
      createStencil: (folder, name) => made.createStencil(folder, name, [cardWords.newField]),
    },
    vaults: {
      ...vaults,
      showVault: (vault) => (shown.value = vault),
      reload,
    },
    goes: {
      revealPath: (path) => void files().revealPath(path),
      travel: async (path) => {
        await plexes().travel(path)
      },
      leave: async (from, to) => {
        await plexes().leavePath(from, to)
      },
      getOpeningNote,
      openTab: (kind) => void held.openTabOfKind(kind),
      openPreset: (path) => openPreset(path),
      closeTab: (tab) => held.requestClose(tab),
      ask: (text) => void agents().askQuestion(text),
      search: options.search ?? (() => {}),
    },
    settings: {
      chooseAppearance: async (chosen) => {
        await dressed.chooseItem(chosen)
      },
      chooseSync: async (chosen) => {
        await oneName.choose(chosen)
      },
      chooseHanging: async (chosen) => {
        await hungParts.choose(chosen)
      },
      chooseParts: async (chosen) => {
        await hungParts.chooseCount(chosen)
      },
    },
    notes: reached,
    runSupport: runs,
    copyPath: (path) => void navigator.clipboard?.writeText(path),
    writeMessage,
  }
}
