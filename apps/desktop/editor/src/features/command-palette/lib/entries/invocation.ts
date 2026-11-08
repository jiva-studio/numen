import type { CommandInvocation, CommandTarget } from '../../types'

/** One command as executed, with the target it was invoked on. */
export const createInvocation = (
  id: string,
  target: CommandTarget,
  ...[name = '', note = null]: [string?, (string | null)?]
): CommandInvocation => ({
  id,
  path: target.path,
  vault: target.vault,
  note,
  title: target.title,
  file: target.file,
  ...(target.line === undefined ? {} : { line: target.line }),
  ...(target.selector === undefined ? {} : { selector: target.selector }),
  others: target.others ?? [],
  name,
  kind: target.kind,
  tab: target.tab,
  ...(target.isTargetNamed ? { isTargetNamed: true } : {}),
})
