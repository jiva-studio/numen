/**
 * The slots of the window as the plugin host serves them: the entries the plugins and the application
 * place, how their commands are drawn, and the when values their conditions and the conditions of
 * the commands of the plugins read.
 */
import {
  getWhenValues as buildWhenValues,
  type CommandTarget,
  type HandlerDeclaration,
  type HostCommands,
  type HostSlots,
  type WhenReader,
  type WhenSettings,
  type WhenValues,
} from '@numen/plugin-loader'
import type { Ref } from 'vue'
import { getPaletteId, type DeclaredCommand, type DeclaredSlot } from '@/entities/plugin'
import { createSlotCommandFinder, readSlotEntries } from './pluginSlots'
import type { WindowFactsSource } from './createWindowFacts'

export interface WindowSlotsDeps {
  readonly declaredCommands: {
    readonly slots: Readonly<Ref<readonly DeclaredSlot[] | undefined>>
    readonly commands: Readonly<Ref<readonly DeclaredCommand[] | undefined>>
    readonly contributions: {
      readonly handlers: Readonly<Ref<readonly HandlerDeclaration[] | undefined>>
    }
    readonly whenContext: {
      readonly getContentTypes: () => Readonly<Record<string, string>>
      readonly getConfig: () => WhenSettings
      readonly whenReader: WhenReader
    }
  }
  readonly facts: WindowFactsSource
  readonly writeIssue: (pluginId: string, message: string, severity: 'warning' | 'error') => void
}

/** What the host reads of the slots of the window, and of the when values of its commands. */
interface WindowSlotParts {
  readonly slots: Required<
    Pick<
      HostSlots,
      'getSlotEntries' | 'createSlotCommandFinder' | 'reportSlotIssue' | 'getSlotWhenValues'
    >
  >
  readonly commands: Required<
    Pick<HostCommands, 'getWindowValues' | 'getContentTypes' | 'whenReader'>
  >
}

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
  const createFinder: WindowSlotParts['slots']['createSlotCommandFinder'] = (own) =>
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
  const hostDeps: WindowSlotParts = {
    slots: {
      getSlotEntries: () => readSlotEntries(deps.declaredCommands.slots.value),
      createSlotCommandFinder: createFinder,
      reportSlotIssue: deps.writeIssue,
      getSlotWhenValues: getWhenValues,
    },
    commands: {
      getWindowValues,
      getContentTypes: whenContext.getContentTypes,
      whenReader: whenContext.whenReader,
    },
  }
  return { hostDeps }
}
