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
  noFilterMatch: 'No result matches the filter.',
  filterResults: 'Filter results',
  sources: 'Sources',
  searchSources: 'Search sources',
  noSourceMatch: 'No source matches',
  editQuery: 'Edit query',
  queryField: 'Query',
  page: 'Page',
  results: 'Results',
} as const

/** The words the tab is drawn with. */
export type Words = typeof WORDS
