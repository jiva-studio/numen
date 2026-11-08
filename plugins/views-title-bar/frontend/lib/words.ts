/** The words the bar writes itself: the names of its own buttons. */
export interface TitleBarWords {
  readonly toolbar: string
  readonly more: string
  readonly minimize: string
  readonly maximize: string
  readonly restore: string
  readonly close: string
  readonly switcher: string
  readonly search: string
  readonly nothing: string
  readonly createVault: string
  readonly loadingText: string
}

export const TITLE_BAR_WORDS: TitleBarWords = {
  toolbar: 'Title bar',
  more: 'More commands',
  minimize: 'Minimize',
  maximize: 'Maximize',
  restore: 'Restore',
  close: 'Close',
  switcher: 'Switch vault',
  search: 'Search vaults',
  nothing: 'No vault found',
  createVault: 'Create vault',
  loadingText: 'Loading…',
}
