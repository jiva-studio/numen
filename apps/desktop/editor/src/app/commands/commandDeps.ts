import { formatErrorMessage } from '@numen/core-client'
import { openAfterSavingTabs, type TabsHold } from './leaveVault'
import { runInBackground } from '@numen/ui'
import { vaults } from '@/entities/vault'
import type { VaultCore } from '../vault'
import type { CommandDeps, Notes, VaultWriter } from '@/features/command-palette'
import type { RunSupport, VaultRef } from '@/entities/command'
import type { MessageWriter } from '@/shared/notices/messages'
import type { useWindowTabs } from '@/entities/tab'
import { renameNote, type NoteCreator } from '@/entities/note'
import { createFolder, emptyTrash, listFiles, moveFile, removeFile } from '@/entities/file'
import type { Model, SettingEdit } from '@/entities/settings'

export interface CommandDepsOptions {
  core: VaultCore
  windowTabs: ReturnType<typeof useWindowTabs>
  making: NoteCreator
  setVaultName: (vault: VaultRef) => void
  reload: () => void
  tabsHold?: TabsHold
  reached: Notes
  settingPalette?: { choose: (command: string, item: string) => Promise<void> }
  rest: {
    writeSettings: (content: readonly SettingEdit[]) => Promise<void> | void
    getModelsAt?: (targetPath: readonly string[]) => readonly Model[]
  }
  travel: (path: string) => Promise<void> | void
  leavePath: (from: string, to: string) => Promise<void> | void
  getOpeningNote: () => string
  runs: RunSupport
  writeMessage: MessageWriter
  search?: () => void
}

function createFileCommands(making: NoteCreator): VaultWriter {
  return {
    createNote: (title, from, relation) => making.createWithTitle(title, from, relation),
    createUntitled: (folder) => making.createUntitled(folder),
    rename: (path, title) => renameNote(path, title),
    remove: (path) => removeFile(path),
    emptyTrash: () => emptyTrash(),
    move: (from, to) => moveFile(from, to),
    listNames: async (folder) => (await listFiles(folder)).map((one) => one.name),
    createFolder: (path) => createFolder(path),
  }
}

function createNavigationCommands(options: CommandDepsOptions): CommandDeps['navigator'] {
  const { travel, leavePath, windowTabs, getOpeningNote, search, writeMessage } = options
  return {
    travel: async (path) => {
      await travel(path)
    },
    leave: async (from, to) => {
      await leavePath(from, to)
    },
    getOpeningNote,
    openTab: (kind) =>
      runInBackground(windowTabs.openTabOfKind(kind), (error) =>
        writeMessage(formatErrorMessage(error), 'error'),
      ),
    showTab: (kind, path) =>
      runInBackground(windowTabs.navigateTab(kind, path), (error) =>
        writeMessage(formatErrorMessage(error), 'error'),
      ),
    closeTab: (tab) => windowTabs.requestClose(tab),
    // Without a palette to open, searching does nothing.
    search: search ?? (() => undefined),
  }
}

function createSettingsCommands(options: CommandDepsOptions): CommandDeps['settings'] {
  const { settingPalette } = options
  return {
    chooseSetting: async (command, selected) => {
      await settingPalette?.choose(command, selected)
    },
  }
}

function tabMatchesPath(
  windowTab: ReturnType<ReturnType<typeof useWindowTabs>['getTab']>,
  path: string,
): boolean {
  if (!windowTab) return false
  const openPath = windowTab.kind.getOpenTab?.(windowTab.state).path
  if (openPath === path) return true
  const target = windowTab.kind.getTarget?.(windowTab.state)
  return target?.file === path || target?.path === path
}

function findActiveTabAt(
  windowTabs: ReturnType<typeof useWindowTabs>,
  path: string,
): string | null {
  if (!path) return null
  for (const tab of windowTabs.tabs.value) {
    if (tabMatchesPath(windowTabs.getTab(tab.id), path)) return tab.id
  }
  return null
}

/**
 * The open files a command can access: those held by the note editors, and those
 * a tab kind serves.
 */
export function createCommandNotes(
  reached: Notes,
  windowTabs: ReturnType<typeof useWindowTabs>,
): Notes {
  const getTab = (id: string) => windowTabs.getTab(id)
  return {
    ...reached,
    getTabAt: (path: string) => reached.getTabAt(path) ?? findActiveTabAt(windowTabs, path),
    getPath: (id: string) => {
      const path = reached.getPath(id)
      if (path !== id) return path
      const tab = getTab(id)
      return tab?.kind.getOpenTab?.(tab.state).path ?? id
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
      reached.close(id)
      windowTabs.requestClose(id)
    },
  }
}

export function createCommandDeps(options: CommandDepsOptions): CommandDeps {
  const { core, making, setVaultName, reload, reached, windowTabs, runs, writeMessage, tabsHold } =
    options

  return {
    files: createFileCommands(making),
    vaults: {
      ...vaults,
      ...(tabsHold ? { open: openAfterSavingTabs(vaults.open, tabsHold) } : {}),
      setVaultName,
      reload,
    },
    navigator: createNavigationCommands(options),
    settings: createSettingsCommands(options),
    notes: createCommandNotes(reached, windowTabs),
    runSupport: runs,
    copyPath: (path) => {
      const clipboard = navigator.clipboard as Clipboard | undefined
      if (!clipboard) return
      runInBackground(clipboard.writeText(path), (error) =>
        writeMessage(formatErrorMessage(error), 'error'),
      )
    },
    writeMessage,
    emitEvent: (event) => (core as { emitEvent?: (event: string) => void }).emitEvent?.(event),
  }
}
