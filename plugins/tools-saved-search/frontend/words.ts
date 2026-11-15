/** Words of the saved search tab. */
export const WORDS = {
  computing: 'Computing the results…',
  updating: 'Updating…',
  willUpdate: 'Will update',
  upToDate: 'Up to date',
  upToDateWithWarnings: 'Up to date, with warnings',
  computed: 'Computed',
  lastRunFailed: 'Last run failed',
  noMatch: 'The query found nothing',
  noFilterMatch: (filter: string) => `No result matches “${filter}”`,
  allSourcesHidden: 'All sources are hidden',
  clearFilter: 'Clear filter',
  showAllSources: 'Show all sources',
  filterResults: 'Filter results',
  sources: 'Sources',
  searchSources: 'Search sources',
  noSourceMatch: 'No source matches',
  editQuery: 'Edit query',
  queryField: 'Query',
  queryProblems: 'Problems in the query',
  textMethods: [
    {
      name: 'startsWith',
      summary: 'begins with this text',
      description:
        'Keeps the files whose path begins with the text you give, such as the name of a folder.',
      example: 'path.startsWith("books/")',
    },
  ],
  page: 'Page',
  results: 'Results',
  problem: {
    unexpectedAtEnd: (token: string) => `Unexpected “${token}”: the query should end here`,
    unexpected: (token: string) => `Unexpected “${token}” here`,
    missingAtEnd: (token: string) => `Missing “${token}” at the end of the query`,
    missing: (token: string, at: string) => `Missing “${token}” before “${at}”`,
    unreadable: (token: string) => `Cannot read the query at “${token}”`,
    notAllowed: (token: string) => `“${token}” is not allowed in a query`,
    endsEarly: 'The query ends too early',
  },
} as const

/** The words the tab is drawn with. */
export type Words = typeof WORDS
