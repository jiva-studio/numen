import { runInBackground } from '@numen/ui'
import type { CommandDeps, SearchDeps } from '@/features/command-palette'
import type { Notes, RunSupport } from '@/entities/command'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import type { WindowTabs } from '@/entities/tab'

export interface CommandDepsOptions {
  core: SearchDeps
  windowTabs: WindowTabs
  reached: Notes
  settingPalette?: { choose: (command: string, item: string) => Promise<void> }
  runs: RunSupport
  writeMessage: MessageWriter
  search?: () => void
}

function createNavigationCommands(options: CommandDepsOptions): CommandDeps['navigator'] {
  const { windowTabs, search, writeMessage } = options
  return {
    openTab: (kind) =>
      runInBackground(windowTabs.openTabOfKind(kind), (error) =>
        writeMessage(formatError(error), 'error'),
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

function tabMatchesPath(windowTab: ReturnType<WindowTabs['getTab']>, path: string): boolean {
  if (!windowTab) return false
  const openPath = windowTab.kind.getOpenTab?.(windowTab.state).path
  if (openPath === path) return true
  const target = windowTab.kind.getTarget?.(windowTab.state)
  return target?.file === path || target?.path === path
}

function findActiveTabAt(windowTabs: WindowTabs, path: string): string | null {
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
export function createCommandNotes(reached: Notes, windowTabs: WindowTabs): Notes {
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
  const { core, windowTabs, reached, runs, writeMessage } = options

  return {
    navigator: createNavigationCommands(options),
    settings: createSettingsCommands(options),
    notes: createCommandNotes(reached, windowTabs),
    runSupport: runs,
    writeMessage,
    emitEvent: (event) => (core as { emitEvent?: (event: string) => void }).emitEvent?.(event),
  }
}
