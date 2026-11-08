/**
 * The when conditions of the plugins' commands in the window: what the window gives them besides the
 * target (the content types the core lists and the settings), and the palette group a condition
 * puts its command in.
 */
import { parseWhen, type WhenCondition, type WhenSettings } from '@numen/plugin-host'
import type { CommandTarget } from '@/features/command-palette'
import { getPaletteValues } from '@/shared/paletteValues'

export interface WindowWhenContext {
  readonly getContentTypes: () => Readonly<Record<string, string>>
  readonly getConfig: () => WhenSettings
}

/** The context of a window that has not listed the content types and has no settings yet. */
export const EMPTY_WHEN_CONTEXT: WindowWhenContext = {
  getContentTypes: () => ({}),
  getConfig: () => ({}),
}

const conditions = new Map<string, WhenCondition | undefined>()

/** The condition of the text, read once; undefined where it cannot be read. */
export function getCondition(when: string | undefined): WhenCondition | undefined {
  const text = when ?? ''
  if (!conditions.has(text)) {
    const parsed = parseWhen(text)
    conditions.set(text, parsed.ok ? parsed.value : undefined)
  }
  return conditions.get(text)
}

/** Whether a command of the condition is available over what is in front of the palette. */
export function createWhenAvailability(context: WindowWhenContext) {
  return (when: string | undefined, target: CommandTarget): boolean =>
    getCondition(when)?.evaluate(
      getPaletteValues(target, context.getContentTypes(), context.getConfig()),
    ) ?? false
}

/** Whether the condition reads whether the target is a note: its command is over the note in front. */
export const hasNoteVariable = (when: string | undefined): boolean =>
  getCondition(when)?.names.includes('target.isNote') ?? false

/** The palette group of a command by its condition: over the note, over a file, or over the window. */
export function getWhenGroup(when: string | undefined): 'note' | 'file' | 'window' {
  if (hasNoteVariable(when)) return 'note'
  return getCondition(when)?.names.some((name) => name.startsWith('target.')) ? 'file' : 'window'
}
