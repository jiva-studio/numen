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
  type WhenValues,
} from '@numen/plugin-host'
import type { Command } from '@/features/command-palette'
import type { DeclaredCommand, DeclaredSlot } from '@/entities/plugin'
import { getNamedIcon } from '@/shared/icons'
import { getPaletteId } from './declaredCommands'
import { APP_SLOT_ENTRIES } from './appPlacements'
import { getCondition } from './whenContext'

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
  readonly getEditorCommands: () => readonly Pick<
    Command,
    'id' | 'text' | 'iconName' | 'when' | 'listItems'
  >[]
  /** Whether the plugin command of the full id lists items; none where absent. */
  readonly hasPluginItems?: (command: string) => boolean
  /** The values the when conditions of the commands read over the target of the slot. */
  readonly getWhenValues: (target: CommandTarget | undefined) => WhenValues
}

/** Whether the target is one file, which the items of a command are listed over. */
const isOverFile = (target: CommandTarget | undefined): boolean =>
  target !== undefined && !target.isFolder && target.path !== ''

/** How the command of the full id is drawn over the target, and undefined while it is not available there. */
export function createSlotCommandFinder(
  deps: SlotCommandFinderDeps,
): (command: string, target: CommandTarget | undefined) => SlotCommandLook | undefined {
  const isAvailableOver = (when: string | undefined, target: CommandTarget | undefined) =>
    getCondition(when)?.evaluate(deps.getWhenValues(target)) ?? false

  const findEditorLook = (command: string, target: CommandTarget | undefined) => {
    const found = deps.getEditorCommands().find((one) => one.id === command)
    if (!found || !isAvailableOver(found.when, target)) return undefined
    return {
      title: found.text,
      ...(found.iconName ? { icon: getNamedIcon(found.iconName) } : {}),
      shortcut: '',
      hasItems: found.listItems !== undefined,
    }
  }

  const findPluginLook = (command: string, target: CommandTarget | undefined) => {
    const hostCommand = deps.getPluginCommands().find((one) => one.id === command)
    const declared = deps.getDeclaredCommands()?.find((one) => getPaletteId(one) === command)
    const look = hostCommand ?? declared
    const hasItems = deps.hasPluginItems?.(command) ?? false
    // The versions a command with items runs over are the core's to list, over any file.
    const isAvailable = hasItems ? isOverFile(target) : isAvailableOver(look?.when, target)
    if (!look || !isAvailable) return undefined
    return {
      title: look.title,
      ...(look.icon ? { icon: getNamedIcon(look.icon) } : {}),
      shortcut: look.shortcut ?? '',
      hasItems,
    }
  }

  return (command, target) =>
    getCommandOwner(command) === APP_COMMAND_OWNER
      ? findEditorLook(command, target)
      : findPluginLook(command, target)
}
