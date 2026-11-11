/**
 * The parts of the window that read the vault: its tabs, the note editors, the display and the
 * settings, and the tabs as the person works with them.
 */
import { measureTabs } from '@/entities/tab'
import { createCommandNotes } from '@/features/command-palette'
import { createWindowSettings } from '@/features/settings-commands'
import { WORDS as words } from '@/shared/ui/words'
import { createTabsPart } from './createTabsPart'
import { createVaultServices } from './createVaultServices'
import { createWindowTabParts } from './createWindowTabParts'
import type { WindowFoundation } from './createWindowFoundation'

/** What the parts that read the vault use of the foundation of the window. */
export type VaultStageDeps = Pick<
  WindowFoundation,
  'log' | 'rest' | 'writeMessage' | 'status' | 'settingsRequests'
>

export function createWindowVault(deps: VaultStageDeps) {
  const { log, rest, writeMessage } = deps
  const tabsPart = createTabsPart({
    writeMessage,
    rest,
    isPluginStarting: deps.status.isPluginStarting,
    openPluginSettings: deps.settingsRequests.emit,
  })
  const { windowTabs, windowId, tabLinks, tabOpeners } = tabsPart
  const vault = createVaultServices({ log, writeMessage, windowTabs, tabLinks, tabOpeners })
  const notes = createCommandNotes(vault.editing.reached, windowTabs)
  return {
    tabsPart,
    vault,
    notes,
    windowSettings: createWindowSettings({
      rest,
      words,
      log,
      onSizeChanged: () => measureTabs(windowTabs),
    }),
    tabs: createWindowTabParts({
      windowTabs,
      windowId,
      tabLinks,
      reports: tabsPart.reports,
      notes,
      tabSync: vault.tabSync,
      writeMessage,
    }),
  }
}

export type WindowVault = ReturnType<typeof createWindowVault>
