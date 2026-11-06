/**
 * The slot entries of the window as the plugin host reads them: those the enabled plugins declare
 * and those of the application, and how the command of an entry is drawn over a target.
 */
import {
  APP_COMMAND_OWNER,
  getCommandOwner,
  type DeclaredSlotEntry,
  type PluginPaletteCommand,
  type SlotCommandLook,
  type CommandTarget,
} from '@numen/plugin-host'
import type { Command } from '@/features/command-palette'
import { canRunOn, type DeclaredCommand, type DeclaredSlot } from '@/entities/plugin'
import { getNamedIcon } from '@/shared/icons'
import { getPaletteId } from './declaredCommands'
import { APP_SLOT_ENTRIES } from './appPlacements'

/** The entries of the plugins with those of the application; undefined until the plugins have been listed. */
export function readSlotEntries(
  declared: readonly DeclaredSlot[] | undefined,
): readonly DeclaredSlotEntry[] | undefined {
  if (declared === undefined) return undefined
  const fromPlugins = declared.map(({ iconName, ...entry }): DeclaredSlotEntry => ({
    ...entry,
    ...(iconName ? { icon: getNamedIcon(iconName) } : {}),
  }))
  return [...fromPlugins, ...APP_SLOT_ENTRIES]
}

export interface SlotCommandFinderDeps {
  /** The commands the plugin host lists: those with a handler and those of a plugin still loading. */
  readonly getPluginCommands: () => readonly PluginPaletteCommand[]
  /** The commands the plugins declare; undefined until the first list has loaded. */
  readonly getDeclaredCommands: () => readonly DeclaredCommand[] | undefined
  /** The commands of the editor. */
  readonly getEditorCommands: () => readonly Pick<Command, 'id' | 'text' | 'iconName'>[]
}

/** Whether a command of a plugin runs over the target: a command over a note needs one, a command over files a file of its types. */
const isAvailableOver = (
  scope: string | undefined,
  declared: DeclaredCommand | undefined,
  target: CommandTarget | undefined,
): boolean => {
  if (scope === 'note' && (target === undefined || target.isFolder)) return false
  if (declared === undefined || declared.contentTypes.length === 0) return true
  return target !== undefined && !target.isFolder && canRunOn(declared, target.path)
}

/** How the command of the full id is drawn over the target, and undefined while it is not available there. */
export function createSlotCommandFinder(
  deps: SlotCommandFinderDeps,
): (command: string, target: CommandTarget | undefined) => SlotCommandLook | undefined {
  const findEditorLook = (command: string): SlotCommandLook | undefined => {
    const found = deps.getEditorCommands().find((one) => one.id === command)
    if (!found) return undefined
    return {
      title: found.text,
      ...(found.iconName ? { icon: getNamedIcon(found.iconName) } : {}),
      shortcut: '',
    }
  }

  const findPluginLook = (command: string, target: CommandTarget | undefined) => {
    const hostCommand = deps.getPluginCommands().find((one) => one.id === command)
    const declared = deps.getDeclaredCommands()?.find((one) => getPaletteId(one) === command)
    const look = hostCommand ?? declared
    if (!look || !isAvailableOver(look.scope, declared, target)) return undefined
    return {
      title: look.title,
      ...(look.icon ? { icon: getNamedIcon(look.icon) } : {}),
      shortcut: look.shortcut ?? '',
    }
  }

  return (command, target) =>
    getCommandOwner(command) === APP_COMMAND_OWNER
      ? findEditorLook(command)
      : findPluginLook(command, target)
}
