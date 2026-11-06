/**
 * The slots of the window as the plugin host serves them: the entries the plugins and the application
 * place, how their commands are drawn, and the when values their conditions and the conditions of
 * the commands of the plugins read.
 */
import {
  getTargetValues,
  type CommandTarget,
  type PluginHost,
  type PluginHostDeps,
  type WhenValues,
} from '@numen/plugin-host'
import { createCommands, type Command } from '@/features/command-palette'
import { WORDS as words } from '@/shared/words'
import type { DeclaredCommands } from '../commands/declaredCommands'
import { createSlotCommandFinder, readSlotEntries } from '../commands/pluginSlots'
import { getPaletteId } from '../commands/declaredCommands'

export interface WindowSlotsDeps {
  readonly declaredCommands: Pick<
    DeclaredCommands,
    'slots' | 'commands' | 'handlers' | 'whenContext'
  >
  readonly hostHolder: { readonly current?: PluginHost }
  /** Writes what is wrong with the slot entries of a plugin to its messages. */
  readonly writeIssue: (pluginId: string, message: string, severity: 'warning' | 'error') => void
  /** The commands the window makes as it runs, such as those over a tab. */
  readonly getAppCommands?: () => readonly Command[]
}

type WindowHostDeps = Required<
  Pick<
    PluginHostDeps,
    | 'getSlotEntries'
    | 'findSlotCommand'
    | 'reportSlotIssue'
    | 'getSlotWhenValues'
    | 'getWindowValues'
    | 'getContentTypes'
  >
>

export function createWindowSlots(deps: WindowSlotsDeps) {
  let editorCommands: readonly Command[] | undefined
  /** The window variables of when, and the settings as config; set once the window can say what is in front. */
  const windowValues: { current?: () => WhenValues } = {}
  const { whenContext } = deps.declaredCommands
  const getWindowValues = (): WhenValues => ({
    config: whenContext.getConfig(),
    ...windowValues.current?.(),
  })
  const getWhenValues = (target: CommandTarget | undefined): WhenValues => ({
    ...getWindowValues(),
    ...getTargetValues(target, whenContext.getContentTypes(), {
      type: target?.noteType,
      properties: target?.properties,
    }),
  })
  const findCommand = createSlotCommandFinder({
    getWhenValues,
    getPluginCommands: () => deps.hostHolder.current?.getPaletteCommands() ?? [],
    getDeclaredCommands: () => deps.declaredCommands.commands.value,
    getHandlerDeclarations: () => deps.declaredCommands.handlers.value,
    getEditorCommands: () => [
      ...(editorCommands ??= createCommands(words)),
      ...(deps.getAppCommands?.() ?? []),
    ],
    hasPluginItems: (command) =>
      (deps.hostHolder.current?.hasCommandItems(command) ?? false) ||
      (deps.declaredCommands.commands.value?.some(
        (one) => one.hasItems && getPaletteId(one) === command,
      ) ??
        false),
  })
  const hostDeps: WindowHostDeps = {
    getSlotEntries: () => readSlotEntries(deps.declaredCommands.slots.value),
    findSlotCommand: findCommand,
    reportSlotIssue: deps.writeIssue,
    getSlotWhenValues: getWhenValues,
    getWindowValues,
    getContentTypes: whenContext.getContentTypes,
  }
  return { hostDeps, windowValues }
}
