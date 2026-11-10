/**
 * The slots of the window as the plugin host serves them: the entries the plugins and the application
 * place, how their commands are drawn, and the when values their conditions and the conditions of
 * the commands of the plugins read.
 */
import {
  getWhenValues as buildWhenValues,
  type CommandTarget,
  type PluginHost,
  type PluginHostDeps,
  type WhenValues,
} from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import {
  type DeclaredCommands,
  createSlotCommandFinder,
  readSlotEntries,
  getPaletteId,
} from '@/features/plugin-commands'

export interface WindowSlotsDeps {
  readonly declaredCommands: Pick<
    DeclaredCommands,
    'slots' | 'commands' | 'contributions' | 'whenContext'
  >
  readonly hostHolder: { readonly current?: PluginHost }
  readonly writeIssue: (pluginId: string, message: string, severity: 'warning' | 'error') => void
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
    | 'whenReader'
  >
>

export function createWindowSlots(deps: WindowSlotsDeps) {
  /** The window variables of when, and the settings as config; set once the window can say what is in front. */
  const windowValues: { current?: () => WhenValues } = {}
  const { whenContext } = deps.declaredCommands
  const getWindowValues = (): WhenValues => ({
    config: whenContext.getConfig(),
    ...windowValues.current?.(),
  })
  const getWhenValues = (target: CommandTarget | undefined): WhenValues =>
    buildWhenValues({
      window: getWindowValues(),
      target,
      note: { type: target?.noteType, properties: target?.properties },
      contentTypes: whenContext.getContentTypes(),
    })
  const findCommand = createSlotCommandFinder({
    getWhenValues,
    whenReader: whenContext.whenReader,
    getPluginCommands: () => deps.hostHolder.current?.commands.getPaletteCommands() ?? [],
    getDeclaredCommands: () => deps.declaredCommands.commands.value,
    getHandlerDeclarations: () => deps.declaredCommands.contributions.handlers.value,
    getEditorCommands: () => deps.getAppCommands?.() ?? [],
    hasPluginItems: (command) =>
      (deps.hostHolder.current?.commands.hasCommandItems(command) ?? false) ||
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
    whenReader: whenContext.whenReader,
  }
  return { hostDeps, windowValues }
}
