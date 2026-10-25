import type { CommandInvocation, CommandTarget } from '../../types'

/** One command as it is carried out, over what it was asked over. */
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
  others: target.others ?? [],
  name,
  kind: target.kind,
  tab: target.tab,
})
