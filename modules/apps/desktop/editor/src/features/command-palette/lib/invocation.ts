import type { CommandInvocation, CommandTarget } from '../types'

/** One command as it is carried out, over what it was asked over. */
export const invocationOf = (
  id: string,
  at: CommandTarget,
  name = '',
  note: string | null = null,
): CommandInvocation => ({
  id,
  path: at.path,
  vault: at.vault,
  note,
  title: at.title,
  file: at.file,
  others: at.others ?? [],
  name,
  kind: at.kind,
  tab: at.tab,
})
