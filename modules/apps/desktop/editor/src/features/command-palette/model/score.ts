/**
 * Search scoring, coverage evaluation, and group ranking limits.
 */
import { isTextSearchOnly, type IndexCoverage } from '@/shared/notices/coverage'
import type { Words } from './search'

/** Which group is which, and nothing else is one. */
export type SearchGroup = 'names' | 'text' | 'meaning'

/** How many answers each group holds. */
export const GROUP_RESULT_LIMIT = 8

/** How long a keystroke waits before anything is asked. */
export const SEARCH_DEBOUNCE_MS = 120

/**
 * Why a group holds nothing, based on coverage score and indexing state.
 */
export function getEmptyGroupReason(
  id: SearchGroup,
  failureMessage: string | undefined,
  words: Words,
  coverage?: () => IndexCoverage,
): string {
  if (failureMessage) return failureMessage
  const read = id === 'meaning' ? coverage?.() : undefined
  if (!read) return ''
  if (isTextSearchOnly(read)) return words.wordsOnly
  return read.embedded === 0 ? words.notEmbedded : ''
}
