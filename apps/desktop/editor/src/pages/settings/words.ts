import { PLUGIN_WORDS } from '@numen/plugin-manager'

/** UI copy of the settings tab: group headings, setting titles and descriptions. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  ...PLUGIN_WORDS,
  /** Shown in place of the settings file path until the path is known. */
  file: 'numen.json in the configuration folder',
  /** Group headings, in display order. */
  window: 'Window',
  naming: 'Note names',
  performance: 'Performance',
  /** Window settings. */
  theme: 'Theme',
  themeDetail: 'Window stylesheet.',
  shipped: 'Built-in',
  owned: 'Custom',
  mode: 'Light or dark',
  modeDetail: 'Colour scheme of the theme.',
  pinned: 'Set by the theme.',
  system: 'System',
  light: 'Light',
  dark: 'Dark',
  interfaceScale: 'Interface size',
  interfaceScaleDetail: 'Scale of controls and interface text. 1 is the default.',
  textScale: 'Reading size',
  textScaleDetail: 'Scale of note and book text. 1 is the default.',
  syncing: 'Sync title and filename',
  syncingDetail: 'Renaming one also renames the other.',
  /** Workload and performance calibration. */
  performanceProfile: 'Performance profile',
  eco: 'Eco',
  ecoDetail: 'Minimal power, 1 worker thread.',
  balanced: 'Balanced',
  balancedDetail: 'Standard workload, 50% CPU capacity.',
  maximum: 'Maximum',
  maximumDetail: 'All CPU cores and memory.',
  /** Accessible name of the search field in a long model list. */
  searchModels: 'Search models',
  /** Accessible name of the info button next to a setting title. */
  aboutSetting: (name: string) => `About ${name}`,
  /** Plugin secrets. */
  keys: 'Key',
  keyStored: 'Set',
  keyMissing: 'Not set',
  keyDetail: 'Stored in the system keychain, not in the settings file.',
  keyPlaceholder: 'Paste a key',
  keyDelete: 'Delete the key',
  /** The label of the action that opens a plugin's settings. */
  configure: 'Configure',
  /** A plugin's setting of type file or folder. */
  pathNone: 'None',
  pathChoose: 'Choose…',
  pathAdd: 'Add…',
  pathClear: 'Clear',
}
