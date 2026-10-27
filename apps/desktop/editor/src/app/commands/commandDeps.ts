import { vaults, type VaultCore } from '../vault'
import type {
  CommandDeps,
  Notes,
  RunSupport,
  VaultRef,
  VaultWriter,
} from '@/features/command-palette'
import type { MessageWriter } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'
import type { NoteCreator } from '@/entities/note'
import { write, type SettingEdit } from '@/entities/settings'

export interface CommandDepsOptions {
  core: VaultCore
  windowTabs: ReturnType<typeof useWindowTabs>
  making: NoteCreator
  setVaultName: (vault: VaultRef) => void
  reload: () => void
  reached: Notes
  appearance: { chooseItem: (item: string) => Promise<void> | void }
  oneName: { choose: (item: string) => Promise<void> | void }
  rest: {
    writeSettings: (content: readonly SettingEdit[]) => Promise<void> | void
    getModelsAt?: (targetPath: readonly string[]) => readonly import('@/entities/settings').Model[]
  }
  /** A note put in front of the person, where the plugin that shows notes puts it. */
  travel: (path: string) => Promise<void> | void
  /** Every tab standing on a path moved to another. */
  leavePath: (from: string, to: string) => Promise<void> | void
  getOpeningNote: () => string
  runs: RunSupport
  writeMessage: MessageWriter
  search?: () => void
}

function createFileCommands(core: VaultCore, making: NoteCreator): VaultWriter {
  return {
    createNote: (title, from, relation) => making.createWithTitle(title, from, relation),
    rename: (path, title) => core.rename(path, title),
    remove: (path, isPermanent) => core.remove(path, isPermanent),
    move: (from, to) => core.move(from, to),
    createFolder: (path) => core.createFolder(path),
  }
}

function createNavigationCommands(options: CommandDepsOptions): CommandDeps['navigator'] {
  const { travel, leavePath, windowTabs, getOpeningNote, search } = options
  return {
    travel: async (path) => {
      await travel(path)
    },
    leave: async (from, to) => {
      await leavePath(from, to)
    },
    getOpeningNote,
    openTab: (kind) => void windowTabs.openTabOfKind(kind),
    closeTab: (tab) => windowTabs.requestClose(tab),
    search: search ?? (() => {}),
  }
}

function createSettingsCommands(options: CommandDepsOptions): CommandDeps['settings'] {
  const { appearance, oneName, rest } = options
  return {
    chooseAppearance: async (selected) => {
      await appearance.chooseItem(selected)
    },
    chooseSync: async (selected) => {
      await oneName.choose(selected)
    },
    choosePerformance: async (selected) => {
      await rest.writeSettings([{ at: ['performance', 'profile'], value: write(selected) }])
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

function findActiveTabAt(
  windowTabs: ReturnType<typeof useWindowTabs>,
  path: string,
): string | null {
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
export function createCommandNotes(
  reached: Notes,
  windowTabs: ReturnType<typeof useWindowTabs>,
): Notes {
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
  const { core, making, setVaultName, reload, reached, windowTabs, runs, writeMessage } = options

  return {
    files: createFileCommands(core, making),
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
