/** The strings the appearance commands show. */
export const WORDS = {
  /** The command palette: its name, the field's placeholder and the groups it offers. */
  paletteName: 'Appearance',
  palettePlaceholder: 'Choose a theme or mode',
  apply: 'Apply',
  shipping: 'Built-in',
  owned: 'Custom',
  half: 'Mode',
  current: 'Current',
  pinned: 'The theme sets light or dark itself',
  modes: { system: 'System', light: 'Light', dark: 'Dark' },
  /** Shown when the themes cannot be listed. */
  themesUnlisted: 'The themes could not be listed.',
} as const
