/**
 * The entries the application places in slots with commands of its own, kept apart from the entries of
 * the plugins and merged with them before the slots read them.
 */
import type { DeclaredSlotEntry } from '@numen/plugin-host'

/** The owner of the entries of the application, as the owner of its commands. */
export const APP_PLACEMENT_OWNER = 'app'

export const APP_SLOT_ENTRIES: readonly DeclaredSlotEntry[] = [
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'titleBar.commands',
    command: 'app:settings.open',
    group: 'settings',
  },
]
