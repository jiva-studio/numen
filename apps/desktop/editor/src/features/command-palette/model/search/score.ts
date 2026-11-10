/**
 * Search scoring, coverage evaluation, and group ranking limits.
 */
import { isTextSearchOnly, type IndexCoverage } from '@/shared/notices/coverage'
import type { Words } from './search'

/** Which group is which, and nothing else is one. */
export type SearchGroup = 'names' | 'text' | 'meaning'

/** How many answers each group holds. */
export const GROUP_RESULT_LIMIT = 8

export interface EmptyGroupReasonOptions {
  readonly failureMessage?: string | undefined
  readonly words: Words
  readonly coverage?: (() => IndexCoverage) | undefined
}

/**
 * Why a group holds nothing, based on coverage score and indexing state.
 */
export function getEmptyGroupReason(id: SearchGroup, options: EmptyGroupReasonOptions): string {
  const { failureMessage, words, coverage } = options
  if (failureMessage) return failureMessage
  const readItem = id === 'meaning' ? coverage?.() : undefined
  if (!readItem) return ''
  if (isTextSearchOnly(readItem)) return words.wordsOnly
  return readItem.embedded === 0 ? words.notEmbedded : ''
}
