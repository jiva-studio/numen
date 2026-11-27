// Stryker disable StringLiteral: interface copy
/** Words of the saved search tab. */
export const WORDS = {
  computing: 'Computing',
  updating: 'Updating',
  stale: 'Stale',
  upToDate: 'Up to date',
  withWarnings: 'With warnings',
  failed: 'Failed',
  computed: 'Computed',
  failedAt: 'Failed',
  error: 'Error',
  warning: 'Warning',
  warnings: 'Warnings',
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
  moreProblems: (count: number) => `and ${count} more`,
  formatQuery: 'Format',
  formatQueryHint: 'Format (Shift+Alt+F)',
  textMethods: [
    {
      // Stryker restore next-line StringLiteral
      name: 'startsWith',
      summary: 'begins with this text',
      description:
        'Keeps the files with a path beginning with the given text, such as a folder name.',
      example: 'path.startsWith("books/")',
    },
  ],
  page: 'Page',
  results: 'Results',
} as const

/** The words the tab is drawn with. */
export type Words = typeof WORDS
