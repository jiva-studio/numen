/**
 * The places the application puts commands of its own in, kept apart from the placements of the
 * plugins and merged with them before the menus read them.
 */
import type { MenuPlacement } from '@numen/plugin-host'

/** The owner of the placements of the application, as the owner of its commands. */
export const APP_PLACEMENT_OWNER = 'app'

export const APP_PLACEMENTS: MenuPlacement[] = [
  {
    pluginId: APP_PLACEMENT_OWNER,
    location: 'titleBar.commands',
    command: 'app:settings.open',
    group: 'settings',
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    location: 'fileTree.context',
    command: 'app:note.open',
    group: '0-open',
    order: 10,
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    location: 'fileTree.context',
    command: 'app:note.copyPath',
    group: '2-edit',
    order: 20,
  },
  {
    pluginId: APP_PLACEMENT_OWNER,
    location: 'fileTree.context',
    command: 'app:note.trash',
    group: '9-remove',
  },
]
