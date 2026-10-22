import { vaults, type VaultCore } from './vault'
import { running } from '@/entities/artifact'
import type { CommandDeps, Notes, RunSupport, VaultRef } from '@/features/command-palette'
import type { MessageWriter } from '@/shared/notices/messages'
import type { createFileCreators } from '@/entities/tab'
import type { useWindowTabs } from '@/entities/tab'
import type { NoteCreator } from '@/entities/note'
import { write, type SettingEdit } from '@/entities/settings'

export interface CommandDepsOptions {
  core: VaultCore
  windowTabs: ReturnType<typeof useWindowTabs>
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
    getModelsAt?: (targetPath: readonly string[]) => readonly import('@/entities/settings').Model[]
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
  runs: RunSupport
  writeMessage: MessageWriter
  search?: () => void
}

function createFileCommands(core: VaultCore, making: NoteCreator): CommandDeps['files'] {
  return {
    createNote: (title, from, relation) => making.createWithTitle(title, from, relation),
    rename: (path, title) => core.rename(path, title),
    remove: (path, isPermanent) => core.remove(path, isPermanent),
    move: (from, to) => core.move(from, to),
    createFolder: (path) => core.createFolder(path),
  }
}

function createRunCommands(options: CommandDepsOptions): CommandDeps['runs'] {
  const { loadArtifactStates, recorded, pointed } = options
  return {
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
        ;(pointed?.onDelete ?? pointed?.reloadTranscript)?.(path)
      }
      void loadArtifactStates(path)
      return able
    },
    deleteCopy: async (path) => {
      const able = await running.deleteCopy(path)
      if (able) void loadArtifactStates(path)
      return able
    },
  }
}

function createNavigationCommands(options: CommandDepsOptions): CommandDeps['goes'] {
  const { files, travel, leavePath, agents, windowTabs, openPreset, getOpeningNote, search } = options
  return {
    revealPath: (path) => void files().revealPath(path),
    travel: async (path) => {
      await travel(path)
    },
    leave: async (from, to) => {
      await leavePath(from, to)
    },
    getOpeningNote,
    openTab: (kind) => void windowTabs.openTabOfKind(kind),
    openPreset: (path) => openPreset(path),
    closeTab: (tab) => windowTabs.requestClose(tab),
    ask: (text) => void agents().askQuestion(text),
    search: search ?? (() => {}),
  }
}

function createSettingsCommands(options: CommandDepsOptions): CommandDeps['settings'] {
  const { appearance, oneName, nesting, rest } = options
  return {
    chooseAppearance: async (selected) => {
      await appearance.chooseItem(selected)
    },
    chooseSync: async (selected) => {
      await oneName.choose(selected)
    },
    chooseNesting: async (selected) => {
      await nesting.choose(selected)
    },
    chooseHeadings: async (selected) => {
      await nesting.chooseCount(selected)
    },
    choosePerformance: async (selected) => {
      await rest.writeSettings([{ at: ['performance', 'profile'], value: write(selected) }])
    },
    chooseIndexingModel: async (selected) => {
      const models = rest.getModelsAt?.(['indexing', 'model', 'name']) ?? []
      const model = models.find((one) => one.name === selected)
      if (model) {
        await rest.writeSettings(model.writes)
      } else {
        await rest.writeSettings([{ at: ['indexing', 'model', 'name'], value: write(selected) }])
      }
    },
    choosePluginSetting: async (pluginId: string, key: string, selected: unknown) => {
      await rest.writeSettings([{ at: ['plugins', pluginId, key], value: write(selected) }])
    },
  }
}

function tabMatchesPath(
  windowTab: ReturnType<ReturnType<typeof useWindowTabs>['getTab']>,
  path: string,
): boolean {
  if (!windowTab) return false
  const openPath = windowTab.kind.getOpenTab?.(windowTab.state)?.path
  if (openPath === path) return true
  const target = windowTab.kind.getTarget?.(windowTab.state)
  return target?.file === path || target?.path === path
}

function findActiveTabAt(windowTabs: ReturnType<typeof useWindowTabs>, path: string): string | null {
  if (!path || !windowTabs?.tabs?.value || typeof windowTabs.getTab !== 'function') return null
  for (const tab of windowTabs.tabs.value) {
    if (tabMatchesPath(windowTabs.getTab(tab.id), path)) return tab.id
  }
  return null
}

/**
 * The open files a command reaches: those the note editors hold, and those a
 * kind of tab answers for.
 */
export function createCommandNotes(reached: Notes, windowTabs: ReturnType<typeof useWindowTabs>): Notes {
  const getTab = (id: string) => windowTabs?.getTab?.(id) ?? null
  return {
    ...reached,
    getTabAt: (path: string) => reached?.getTabAt?.(path) ?? findActiveTabAt(windowTabs, path),
    getPath: (id: string) => {
      const path = reached.getPath(id)
      if (path !== id) return path
      const tab = getTab(id)
      return tab?.kind.getOpenTab?.(tab.state)?.path ?? id
    },
    asking: (id: string) => {
      const tab = getTab(id)
      return reached.asking(id) || (tab?.kind.isAsking?.(tab.state) ?? false)
    },
    flushPending: async (id: string) => {
      await reached.flushPending(id)
      const tab = getTab(id)
      await tab?.kind.flushPending?.(tab.state)
    },
    close: (id: string) => {
      reached?.close?.(id)
      windowTabs?.requestClose?.(id)
    },
  }
}

export function createCommandDeps(options: CommandDepsOptions): CommandDeps {
  const { core, making, made, setVaultName, reload, reached, windowTabs, runs, writeMessage } = options

  return {
    files: createFileCommands(core, making),
    runs: createRunCommands(options),
    makers: made,
    vaults: {
      ...vaults,
      setVaultName,
      reload,
    },
    navigator: createNavigationCommands(options),
    settings: createSettingsCommands(options),
    notes: createCommandNotes(reached, windowTabs),
    runSupport: runs,
    copyPath: (path) => void navigator.clipboard?.writeText(path),
    writeMessage,
    emitEvent: (event) => (core as { emitEvent?: (event: string) => void }).emitEvent?.(event),
  }
}
