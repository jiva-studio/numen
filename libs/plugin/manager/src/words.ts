const countFiles = (files: number): string => (files === 1 ? '1 file' : `${files} files`)

/** UI copy of the plugin list and the plugin page. */
// Stryker disable StringLiteral: interface copy
export const PLUGIN_WORDS = {
  settings: 'Settings',
  /** Controls of a plugin. */
  pluginShowDetails: 'Show details',
  pluginHideDetails: 'Hide details',
  pluginRetry: 'Retry',
  pluginDisable: 'Disable',
  pluginEnable: 'Enable',
  pluginInstall: 'Install',
  /** State badge of a plugin, on its row and on its page. */
  pluginState: {
    enabled: 'Enabled',
    disabled: 'Off',
    stopped: 'Stopped',
    error: 'Error',
    warning: 'Warning',
    available: 'Not installed',
  },
  /** Plugin list and plugin page. */
  backToPlugins: 'All plugins',
  pluginEnabled: 'Enabled',
  pluginAvailable: 'This plugin is not installed.',
  pluginCatalogSource: (source: string) => `Offered by ${source}.`,
  noPlugins: 'No plugins installed.',
  systemSettings: 'System',
  systemSettingsDetail: 'Appearance and files',
  searchPlugins: 'Search plugins…',
  clearSearch: 'Clear search',
  filterAll: 'All',
  plugins: 'Plugins',
  connections: 'Connections',
  connectionsDetail: 'Services and programs for plugins',
  categoryFilter: 'Categories',
  pluginRunning: 'Running',
  pluginStoppedBanner: (name: string) => `${name} stopped.`,
  pluginOpen: 'Open',
  /** The sections of a plugin page. */
  pluginTabs: { settings: 'Settings', contributions: 'Contributes', about: 'About', data: 'Data' },
  pluginSections: 'Sections',
  aboutPlugin: 'Plugin',
  pluginVersion: 'Version',
  pluginIdentifier: 'Identifier',
  noSettings: 'Nothing to set up.',
  noContributions: 'Contributes nothing.',
  noPluginsFound: 'No plugins found.',
  noPluginsMatch: (query: string) => `No plugins match “${query}”`,
  noPluginsHint: 'Search by name, description or category, such as agents or readers.',
  readMore: 'Read more',
  showLess: 'Show less',
  /** The folder a plugin keeps in the shown vault, and its deletion. */
  pluginData: 'Plugin data',
  pluginDataDetail: (files: number) => `${countFiles(files)} in the plugin's folder of this vault.`,
  pluginDataDelete: 'Delete data',
  pluginDataAsking: (name: string, files: number) =>
    `Delete ${countFiles(files)} of ${name}? Its texts leave search. This cannot be undone.`,
  pluginDataConfirm: 'Delete',
  pluginDataKeep: 'Keep',
}
