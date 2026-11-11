import { type CommandDeps } from '@/entities/command'
import { runInBackground } from '@numen/ui'
import type { CommandRegistry, Notes, RunSupport } from '@/entities/command'
import { createCommandNotes } from '../notes/createCommandNotes'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { WindowTabs } from '@/entities/tab'

export interface CommandDepsOptions {
  windowTabs: Pick<WindowTabs, 'tabs' | 'panes'>
  reached: Notes
  settingPalette: { choose: (command: string, item: string) => Promise<void> }
  runs: RunSupport
  registry: Pick<CommandRegistry, 'get'>
  writeMessage: MessageWriter
  search: () => void
  /** Receives the event a command run announces, for the plugins that activate on it. */
  emitEvent?: ((event: string) => void) | undefined
}

function createNavigationCommands(options: CommandDepsOptions): CommandDeps['navigator'] {
  const { windowTabs, search, writeMessage } = options
  return {
    openTab: (kind) =>
      runInBackground(windowTabs.tabs.openTabOfKind(kind), (error) =>
        writeMessage(formatError(error), 'error'),
      ),
    closeTab: (tab) => windowTabs.panes.requestClose(tab),
    search,
  }
}

function createSettingsCommands(options: CommandDepsOptions): CommandDeps['settings'] {
  const { settingPalette } = options
  return {
    chooseSetting: async (command, selected) => {
      await settingPalette.choose(command, selected)
    },
  }
}

export function createCommandDeps(options: CommandDepsOptions): CommandDeps {
  const { windowTabs, reached, runs, registry, writeMessage } = options

  return {
    navigator: createNavigationCommands(options),
    settings: createSettingsCommands(options),
    notes: createCommandNotes(reached, windowTabs),
    runSupport: runs,
    registry,
    writeMessage,
    emitEvent: options.emitEvent,
  }
}
