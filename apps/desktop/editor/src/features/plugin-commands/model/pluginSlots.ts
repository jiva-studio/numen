/**
 * The slot entries of the window as the plugin host reads them: those the enabled plugins declare
 * and those of the application, and how the command of an entry is drawn over a target.
 */
import {
  APP_COMMAND_OWNER,
  isSharedCommand,
  SHARED_COMMANDS,
  getCommandOwner,
  type DeclaredSlotEntry,
  type HandlerDeclaration,
  type PluginPaletteCommand,
  type SlotCommandLook,
  type CommandTarget,
  type WhenReader,
  type WhenValues,
} from '@numen/plugin-loader'
import type { Component } from 'vue'
import { type Command } from '@/entities/command'
import { findIcon } from '@/shared/model/command'
import type { DeclaredCommand, DeclaredSlot } from '@/entities/plugin'
import { getNamedIcon } from '@/shared/ui/icons'
import { getPaletteId } from './declaredCommands'
import { APP_SLOT_ENTRIES } from '../lib/appPlacements'

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
  readonly getPluginCommands: () => readonly PluginPaletteCommand[]
  readonly getDeclaredCommands: () => readonly DeclaredCommand[] | undefined
  readonly getEditorCommands: () => readonly Pick<
    Command,
    'id' | 'text' | 'iconName' | 'when' | 'listItems'
  >[]
  readonly getHandlerDeclarations?: () => readonly HandlerDeclaration[] | undefined
  readonly hasPluginItems?: (command: string) => boolean
  readonly getWhenValues: (target: CommandTarget | undefined) => WhenValues
  readonly whenReader: WhenReader
}

function getEditorIcon(iconName: string | undefined, command: string): { icon?: Component } {
  if (iconName) return { icon: getNamedIcon(iconName) }
  const icon = findIcon(command)
  return icon ? { icon } : {}
}

const isOverFile = (target: CommandTarget | undefined): boolean =>
  target !== undefined && !target.isFolder && target.path !== ''

/** How the command of the full id is drawn over the target, and undefined while it is not available there. */
export function createSlotCommandFinder(
  deps: SlotCommandFinderDeps,
): (command: string, target: CommandTarget | undefined) => SlotCommandLook | undefined {
  const isAvailableOver = (when: string | undefined, target: CommandTarget | undefined) =>
    deps.whenReader.getCondition(when)?.evaluate(deps.getWhenValues(target)) ?? false

  const findEditorLook = (command: string, target: CommandTarget | undefined) => {
    const found = deps.getEditorCommands().find((one) => one.id === command)
    if (!found || !isAvailableOver(found.when, target)) return undefined
    return {
      title: found.text,
      ...getEditorIcon(found.iconName, command),
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
      hasItems,
    }
  }

  /**
   * The values of the target a handler is judged by before a menu is drawn: the properties of the note
   * and the variables of a version are unknown until the menu reads them, and so is the type of a note
   * the place has not read.
   */
  const getKnownValues = (target: CommandTarget | undefined): WhenValues => {
    const { ['target.properties']: _properties, ...known } = deps.getWhenValues(target)
    if (known['target.isNote'] !== true || target?.noteType !== undefined) return known
    const { ['target.noteType']: _noteType, ...typeless } = known
    return typeless
  }

  /**
   * A shared command some enabled plugin handles: its handlers are its items, read when a menu opens.
   * It is not drawn where every handler is false over what the window knows of the target.
   */
  const findSharedLook = (command: string, target: CommandTarget | undefined) => {
    const shared = SHARED_COMMANDS.find((one) => one.id === command)
    if (!shared) return undefined
    const partialValues = getKnownValues(target)
    const isPossible = deps
      .getHandlerDeclarations?.()
      ?.some(
        (one) =>
          one.command === command &&
          deps.whenReader.getCondition(one.when)?.evaluatePartial(partialValues) !== 'false',
      )
    const icon = findIcon(command)
    return isPossible
      ? { title: shared.title, ...(icon ? { icon } : {}), hasItems: true }
      : undefined
  }

  return (command, target) => {
    if (isSharedCommand(command)) return findSharedLook(command, target)
    return getCommandOwner(command) === APP_COMMAND_OWNER
      ? findEditorLook(command, target)
      : findPluginLook(command, target)
  }
}
