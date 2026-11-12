import type { CommandInvocation, CommandTarget, InvocationTarget } from '@/shared/model/command'

const getInvocationTarget = (target: CommandTarget): InvocationTarget => ({
  path: target.path,
  vault: target.vault,
  title: target.title,
  file: target.file,
  ...(target.line === undefined ? {} : { line: target.line }),
  ...(target.selector === undefined ? {} : { selector: target.selector }),
  others: target.others ?? [],
  kind: target.kind,
  tab: target.tab,
  ...(target.isFolder ? { isFolder: true } : {}),
  ...(target.isTargetNamed ? { isTargetNamed: true } : {}),
})

/** One command as executed, with the target it was invoked on. */
export const createInvocation = (
  id: string,
  target: CommandTarget,
  ...[name = '', note = null]: [string?, (string | null)?]
): CommandInvocation => ({ id, target: getInvocationTarget(target), note, name })
