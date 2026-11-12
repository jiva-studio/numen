/**
 * Search scoring, coverage evaluation, and group ranking limits.
 */
import type { IndexCoverage } from '@/shared/model/notices/coverage'

/** Which group is which, and nothing else is one. */
export type SearchGroup = 'names' | 'text' | 'meaning'

/** How many answers each group holds. */
export const GROUP_RESULT_LIMIT = 8

export interface EmptyGroupReasonOptions {
  readonly failureMessage?: string | undefined
  readonly coverage?: (() => IndexCoverage) | undefined
}

/**
 * Why a group holds nothing: the failure of its search, or for the meaning group what the search source
 * says of its index, as it says it.
 */
export function getEmptyGroupReason(id: SearchGroup, options: EmptyGroupReasonOptions): string {
  const { failureMessage, coverage } = options
  if (failureMessage) return failureMessage
  return id === 'meaning' ? (coverage?.().text ?? '') : ''
}
