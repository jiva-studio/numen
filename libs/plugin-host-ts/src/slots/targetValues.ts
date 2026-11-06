/** The values a slot gives the keys of the when clauses of its entries, from the target it is over. */
import type { CommandTarget } from '@numen/plugin-sdk'
import type { WhenValues } from '../when/evaluate'

/** The keys of the target; all are false where the place has no target, as in a bar. */
export function getTargetWhenValues(target: CommandTarget | undefined): WhenValues {
  const isFolder = target?.isFolder ?? false
  return {
    // The root folder of the vault is the folder of the empty path.
    isRoot: isFolder && target?.path === '',
    isFolder,
    isMultiple: (target?.others.length ?? 0) > 0,
  }
}
