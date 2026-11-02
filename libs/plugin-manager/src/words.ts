/** A count of files, in the singular for one. */
const countFiles = (files: number): string => (files === 1 ? '1 file' : `${files} files`)

/** UI copy of the plugin list and the plugin page. */
// Stryker disable StringLiteral: interface copy
export const PLUGIN_WORDS = {
  settings: 'Settings',
  /** AI consent of a plugin. */
  consentAllowed: 'Allow model use',
  consentDetail: {
    none: '',
    pending: 'Not set. Requests fail until you choose.',
    requested: 'Requested. Requests fail until you choose.',
    allowed: 'Allowed. Requests go to the agent in its settings.',
    denied: 'Denied. Requests return an error.',
  },
  /** Controls of a plugin. */
  pluginDetails: 'Details',
  pluginRetry: 'Retry',
  pluginDisable: 'Disable',
  pluginEnable: 'Enable',
  pluginInstall: 'Install',
  /** State badge of a plugin, on its row and on its page. */
  pluginState: {
    enabled: 'Enabled',
    disabled: 'Disabled',
    stopped: 'Stopped',
    available: 'Not installed',
  },
  /** Plugin list and plugin page. */
  backToPlugins: 'All plugins',
  pluginEnabled: 'Enabled',
  pluginAvailable: 'This plugin is not installed.',
  pluginCatalogSource: (source: string) => `Offered by ${source}.`,
  noPlugins: 'No plugins installed.',
  systemSettings: 'System',
  systemSettingsDetail: 'Theme, scale and filename sync.',
  searchPlugins: 'Search plugins…',
  noPluginsFound: 'No plugins found.',
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
