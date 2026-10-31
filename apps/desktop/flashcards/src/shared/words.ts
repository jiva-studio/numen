/** The strings displayed by the window. */
export const WORDS = {
  /** What the list of vaults is called. */
  vaults: 'Vaults',
  /** Shown while the installation has not yet reported its vaults. */
  listing: 'Loading vaults…',
  /** Shown when no plugin of the window provides a view. */
  empty: 'No plugin views.',
  /** Shown when the agent's availability cannot be determined. */
  agentUnreachable: 'Agent unreachable.',
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
