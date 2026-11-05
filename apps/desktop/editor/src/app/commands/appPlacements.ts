/**
 * The places the application puts commands of its own in, kept apart from the placements of the
 * plugins and merged with them before the menus read them.
 */
import type { MenuPlacement } from '@numen/plugin-host'

/** The owner of the placements of the application, as the owner of its commands. */
export const APP_PLACEMENT_OWNER = 'app'

export const APP_PLACEMENTS: MenuPlacement[] = []
