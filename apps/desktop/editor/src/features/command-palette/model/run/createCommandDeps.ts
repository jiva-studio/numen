import { type CommandDeps } from '@/shared/model/command'
import { runInBackground } from '@numen/ui'
import type { CommandRegistry, Notes, RunSupport, SettingsRequest } from '@/shared/model/command'
import { createCommandNotes } from '../notes/createCommandNotes'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { SETTINGS, type WindowTabs } from '@/entities/tab'

/** How the window searches and opens its settings. Without openSettings, a settings tab opens. */
export interface NavigationOptions {
  search: () => void
  openSettings?: ((request: SettingsRequest) => void) | undefined
}

export interface CommandDepsOptions {
  windowTabs: Pick<WindowTabs, 'tabs' | 'panes'>
  reached: Notes
  settingPalette: { choose: (command: string, item: string) => Promise<void> }
  runs: RunSupport
  registry: Pick<CommandRegistry, 'get'>
  writeMessage: MessageWriter
  navigation: NavigationOptions
  emitEvent?: ((event: string) => void) | undefined
}

function createNavigationCommands(
  options: Pick<CommandDepsOptions, 'windowTabs' | 'writeMessage' | 'navigation'>,
): CommandDeps['navigator'] {
  const { windowTabs, writeMessage, navigation } = options
  const openTab = (kind: string) =>
    runInBackground(windowTabs.tabs.openTabOfKind(kind), (error) =>
      writeMessage(formatError(error), 'error'),
    )
  return {
    openTab,
    openSettings: (request) =>
      navigation.openSettings ? navigation.openSettings(request) : openTab(SETTINGS),
    closeTab: (tab) => windowTabs.panes.requestClose(tab),
    search: navigation.search,
  }
}

function createSettingsCommands(
  settingPalette: CommandDepsOptions['settingPalette'],
): CommandDeps['settings'] {
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
    settings: createSettingsCommands(options.settingPalette),
    notes: createCommandNotes(reached, windowTabs),
    runSupport: runs,
    registry,
    writeMessage,
    emitEvent: options.emitEvent,
  }
}
