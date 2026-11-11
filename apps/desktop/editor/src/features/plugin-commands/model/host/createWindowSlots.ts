/**
 * The slots of the window as the plugin host serves them: the entries the plugins and the application
 * place, how their commands are drawn, and the when values their conditions and the conditions of
 * the commands of the plugins read.
 */
import {
  getWhenValues as buildWhenValues,
  type CommandTarget,
  type PluginHostDeps,
  type WhenValues,
} from '@numen/plugin-loader'
import { getPaletteId, type DeclaredCommands } from '../declaredCommands'
import { createSlotCommandFinder, readSlotEntries } from '../pluginSlots'
import type { WindowFactsSource } from './createWindowFacts'

export interface WindowSlotsDeps {
  readonly declaredCommands: Pick<
    DeclaredCommands,
    'slots' | 'commands' | 'contributions' | 'whenContext'
  >
  readonly facts: WindowFactsSource
  readonly writeIssue: (pluginId: string, message: string, severity: 'warning' | 'error') => void
}

type WindowHostDeps = Required<
  Pick<
    PluginHostDeps,
    | 'getSlotEntries'
    | 'createSlotCommandFinder'
    | 'reportSlotIssue'
    | 'getSlotWhenValues'
    | 'getWindowValues'
    | 'getContentTypes'
    | 'whenReader'
  >
>

export function createWindowSlots(deps: WindowSlotsDeps) {
  const { whenContext } = deps.declaredCommands
  /** The window variables of when, and the settings as config. */
  const getWindowValues = (): WhenValues => ({
    config: whenContext.getConfig(),
    ...deps.facts.getValues(),
  })
  const getWhenValues = (target: CommandTarget | undefined): WhenValues =>
    buildWhenValues({
      window: getWindowValues(),
      target,
      note: { type: target?.noteType, properties: target?.properties },
      contentTypes: whenContext.getContentTypes(),
    })
  const createFinder: WindowHostDeps['createSlotCommandFinder'] = (own) =>
    createSlotCommandFinder({
      getWhenValues,
      whenReader: whenContext.whenReader,
      getPluginCommands: own.getPluginCommands,
      getDeclaredCommands: () => deps.declaredCommands.commands.value,
      getHandlerDeclarations: () => deps.declaredCommands.contributions.handlers.value,
      getEditorCommands: deps.facts.getAppCommands,
      hasPluginItems: (command) =>
        own.hasPluginItems(command) ||
        (deps.declaredCommands.commands.value?.some(
          (one) => one.hasItems && getPaletteId(one) === command,
        ) ??
          false),
    })
  const hostDeps: WindowHostDeps = {
    getSlotEntries: () => readSlotEntries(deps.declaredCommands.slots.value),
    createSlotCommandFinder: createFinder,
    reportSlotIssue: deps.writeIssue,
    getSlotWhenValues: getWhenValues,
    getWindowValues,
    getContentTypes: whenContext.getContentTypes,
    whenReader: whenContext.whenReader,
  }
  return { hostDeps }
}
