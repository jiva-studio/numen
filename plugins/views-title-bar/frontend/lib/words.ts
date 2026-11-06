/** The words the bar writes itself: the names of its own buttons. */
export interface TitleBarWords {
  readonly more: string
  readonly minimize: string
  readonly maximize: string
  readonly restore: string
  readonly close: string
  /** What the list the title opens is called. */
  readonly switcher: string
  /** What the search field of that list says while empty. */
  readonly search: string
  /** What that list says when the search finds nothing. */
  readonly nothing: string
}

export const TITLE_BAR_WORDS: TitleBarWords = {
  more: 'More commands',
  minimize: 'Minimize',
  maximize: 'Maximize',
  restore: 'Restore',
  close: 'Close',
  switcher: 'Switch vault',
  search: 'Search vaults',
  nothing: 'No vault found',
}
