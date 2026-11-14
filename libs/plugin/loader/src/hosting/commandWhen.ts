/** Whether the host runs a command of a plugin over a note: the when condition of its declaration. */
import type { CommandNote } from '@numen/plugin-sdk'
import { type WhenReader, getWhenValues } from '../when'
import type { CommandDeclaration, HostCommands } from './types'

export type CommandWhenDeps = Pick<HostCommands, 'getContentTypes' | 'getWindowValues'> & {
  readonly whenReader: WhenReader
}

/** Whether the command runs over the note, the note in front or none: its condition holds there. */
export function isCommandAvailable(
  declaration: Pick<CommandDeclaration, 'when'>,
  note: CommandNote,
  deps: CommandWhenDeps,
): boolean {
  const condition = deps.whenReader.getCondition(declaration.when)
  if (!condition) return false
  return condition.evaluate(
    getWhenValues({
      window: deps.getWindowValues?.(),
      target: note.path === '' && !note.isFolder ? undefined : note,
      note: { type: note.type, properties: note.properties },
      contentTypes: deps.getContentTypes?.() ?? {},
    }),
  )
}
