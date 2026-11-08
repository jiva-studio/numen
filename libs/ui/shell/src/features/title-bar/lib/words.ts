/** The words the bar writes itself: the names of its own buttons. */
export interface TitleBarWords {
  readonly more: string
  readonly minimise: string
  readonly maximise: string
  readonly restore: string
  readonly close: string
  readonly switcher: string
  readonly search: string
  readonly nothing: string
}

export const TITLE_BAR_WORDS: TitleBarWords = {
  more: 'More actions',
  minimise: 'Minimise',
  maximise: 'Maximise',
  restore: 'Restore',
  close: 'Close',
  switcher: 'Switch',
  search: 'Search',
  nothing: 'No results',
}
