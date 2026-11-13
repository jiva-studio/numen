/**
 * The when conditions of the plugins' commands in the window: what the window gives them besides the
 * target (the content types the core lists, the settings and the one reader of its conditions), and
 * the palette group a condition puts its command in.
 */
import {
  createWhenReader,
  type JsonObject,
  type WhenCondition,
  type WhenReader,
  type WhenSettings,
} from '@numen/plugin-loader'
import { getPaletteValues, type CommandTarget } from '@/shared/model/command'

export interface WindowWhenContext {
  readonly getContentTypes: () => Readonly<Record<string, string>>
  readonly getConfig: () => WhenSettings
  readonly whenReader: WhenReader
  readonly getNote?: (target: CommandTarget) => PaletteNote | undefined
  readonly forgetNotes?: () => void
}

/** The context of a window that has not listed the content types and has no settings yet. */
export const createEmptyWhenContext = (): WindowWhenContext => ({
  getContentTypes: () => ({}),
  getConfig: () => ({}),
  whenReader: createWhenReader(),
})

/** The type and properties of the note in front of the palette, read from the core. */
export interface PaletteNote {
  readonly type: string
  readonly properties?: JsonObject
}

const hasNoteKindVariable = (condition: WhenCondition): boolean =>
  condition.names.some((name) => name === 'target.noteType' || name === 'target.properties')

/**
 * Whether a command of the condition is available over what is in front of the palette. The target
 * holds no type or properties of its note, so a condition that reads them holds once `getNote` has
 * read the note, and is false while it is read.
 */
export function createWhenAvailability(context: WindowWhenContext) {
  return (when: string | undefined, target: CommandTarget): boolean => {
    const condition = context.whenReader.getCondition(when)
    if (!condition) return false
    const isOverFile = target.isReady && (target.path || target.file) !== ''
    const note =
      isOverFile && context.getNote && hasNoteKindVariable(condition) ? context.getNote(target) : {}
    if (note === undefined) return false
    return condition.evaluate(
      getPaletteValues(target, context.getContentTypes(), context.getConfig(), note),
    )
  }
}

/** Whether the condition reads whether the target is a note: its command is over the note in front. */
export const hasNoteVariable = (condition: WhenCondition | undefined): boolean =>
  condition?.names.includes('target.isNote') ?? false

const hasTargetVariable = (condition: WhenCondition | undefined): boolean =>
  condition?.names.some((name) => name.startsWith('target.')) ?? false

/** The palette group of a command by its condition: over the note, over a file, or over the window. */
export function getWhenGroup(condition: WhenCondition | undefined): 'note' | 'file' | 'window' {
  if (hasNoteVariable(condition)) return 'note'
  return hasTargetVariable(condition) ? 'file' : 'window'
}
