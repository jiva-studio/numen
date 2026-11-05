/**
 * The places plugins put their commands in, over the commands this window lists.
 */
import {
  createMenus,
  type MenuPlacement,
  type Menus,
  type PluginPaletteCommand,
} from '@numen/plugin-host'
import type { Command } from '@/features/command-palette'
import type { DeclaredCommand } from '@/entities/plugin'

export interface PluginMenusDeps {
  /** The placements of the enabled plugins; undefined until the first list has loaded. */
  readonly getPlacements: () => readonly MenuPlacement[] | undefined
  /** The commands the plugin host lists: those with a handler and those of a plugin still loading. */
  readonly getPluginCommands: () => readonly PluginPaletteCommand[]
  /** The commands the plugins declare; undefined until the first list has loaded. */
  readonly getDeclaredCommands: () => readonly DeclaredCommand[] | undefined
  /** The commands of the editor. */
  readonly getEditorCommands: () => readonly Command[]
  readonly reportIssue?: (pluginId: string, message: string) => void
}

export function createPluginMenus(deps: PluginMenusDeps): Menus {
  return createMenus({ getPlacements: deps.getPlacements, findCommand: () => undefined })
}
