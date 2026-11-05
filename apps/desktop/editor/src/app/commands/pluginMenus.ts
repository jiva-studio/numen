/**
 * The places plugins put their commands in, over the commands this window lists.
 */
import {
  APP_COMMAND_OWNER,
  createMenus,
  getCommandOwner,
  type CommandLook,
  type MenuPlacement,
  type Menus,
  type PluginPaletteCommand,
} from '@numen/plugin-host'
import type { Command } from '@/features/command-palette'
import type { DeclaredCommand } from '@/entities/plugin'
import { getPaletteId } from './declaredCommands'

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
  const findPluginLook = (fullId: string): CommandLook | undefined => {
    const listed = deps.getPluginCommands().find((one) => one.id === fullId)
    if (listed) return listed
    const declared = deps.getDeclaredCommands()?.find((one) => getPaletteId(one) === fullId)
    if (!declared) return undefined
    return {
      title: declared.title,
      ...(declared.icon ? { icon: declared.icon } : {}),
      ...(declared.scope ? { scope: declared.scope } : {}),
      ...(declared.shortcut ? { shortcut: declared.shortcut } : {}),
    }
  }

  const findEditorLook = (fullId: string): CommandLook | undefined => {
    const found = deps.getEditorCommands().find((one) => one.id === fullId)
    if (!found) return undefined
    return { title: found.text, ...(found.iconName ? { icon: found.iconName } : {}) }
  }

  return createMenus({
    getPlacements: deps.getPlacements,
    findCommand: (fullId) =>
      getCommandOwner(fullId) === APP_COMMAND_OWNER
        ? findEditorLook(fullId)
        : findPluginLook(fullId),
    ...(deps.reportIssue ? { reportIssue: deps.reportIssue } : {}),
  })
}
