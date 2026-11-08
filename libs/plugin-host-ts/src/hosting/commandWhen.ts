/** Whether the host runs a command of a plugin over a note: the when condition of its declaration. */
import type { CommandNote } from '@numen/plugin-sdk'
import { getTargetValues } from '../when/targetValues'
import { parseWhen, type WhenCondition } from '../when/when'
import type { CommandDeclaration, PluginHostDeps } from './types'

export type CommandWhenDeps = Pick<PluginHostDeps, 'getContentTypes' | 'getWindowValues'>

const conditions = new Map<string, WhenCondition | undefined>()

const getCondition = (text: string): WhenCondition | undefined => {
  if (!conditions.has(text)) {
    const parsed = parseWhen(text)
    conditions.set(text, parsed.ok ? parsed.value : undefined)
  }
  return conditions.get(text)
}

/** Whether the command runs over the note, the note in front or none: its condition holds there. */
export function isCommandAvailable(
  declaration: Pick<CommandDeclaration, 'when'>,
  note: CommandNote,
  deps: CommandWhenDeps,
): boolean {
  const condition = getCondition(declaration.when ?? '')
  if (!condition) return false
  const target = note.path === '' && !note.isFolder ? undefined : note
  return condition.evaluate({
    ...deps.getWindowValues?.(),
    ...getTargetValues(target, deps.getContentTypes?.() ?? {}, {
      type: note.type,
      properties: note.properties,
    }),
  })
}
