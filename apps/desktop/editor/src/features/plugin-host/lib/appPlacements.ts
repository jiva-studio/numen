/**
 * The entries the application places in slots with commands of its own, kept apart from the entries of
 * the plugins and merged with them before the slots read them.
 */
import type { DeclaredSlotEntry } from '@numen/plugin-loader'

/** The owner of the entries of the application, as the owner of its commands. */
const APP_PLACEMENT_OWNER = 'app'

/** One row that is not the root folder. */
const OVER_ONE_ROW = '!target.isRoot && !target.isMultiple'

export const APP_SLOT_ENTRIES: readonly DeclaredSlotEntry[] = [
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'titleBar.commands',
    command: 'app:settings.open',
    group: 'settings',
    when: 'window.isVaultOpen',
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'fileTree.context',
    command: 'app:file.open',
    group: '0-open',
    order: 10,
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'fileTree.context',
    command: 'app:file.copyPath',
    group: '2-edit',
    order: 20,
    when: OVER_ONE_ROW,
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'fileTree.context',
    command: 'app:version.delete',
    group: '3-text',
    order: 90,
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'fileTree.context',
    command: 'app:file.remove',
    group: '9-remove',
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'tabStrip.context',
    command: 'app:tab.moveToWindow',
    group: '0-window',
    order: 10,
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'tabStrip.context',
    command: 'app:tab.detach',
    group: '0-window',
    order: 20,
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'tabStrip.context',
    command: 'app:tab.link',
    group: '1-link',
    order: 10,
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    slot: 'tabStrip.context',
    command: 'app:tab.unlink',
    group: '1-link',
    order: 20,
  },
]
