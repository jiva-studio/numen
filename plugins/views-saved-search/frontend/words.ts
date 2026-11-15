/** Words of the saved search tab. */
export const WORDS = {
  computing: 'Computing the results…',
  updating: 'Updating…',
  willUpdate: 'Will update',
  computed: 'Computed',
  computedLower: 'computed',
  separator: ' · ',
  lastRunFailed: 'Last run failed',
  noMatch: 'No passage matches the query.',
  rerankLabel: 'Rerank:',
  editQuery: 'Edit query',
  queryField: 'Query',
  rerankField: 'Rerank',
  limitField: 'Limit',
  save: 'Save',
  page: 'Page',
  of: 'of',
  results: 'Results',
} as const

/** The words the tab is drawn with. */
export type Words = typeof WORDS
