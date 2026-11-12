import type { PluginContributionKind } from './pluginInfo'
import type { PluginState } from './state'

// Stryker disable StringLiteral: interface copy
const countFiles = (files: number): string => (files === 1 ? '1 file' : `${files} files`)

const PLUGIN_STATES: Readonly<Record<PluginState, string>> = {
  enabled: 'Running',
  disabled: 'Off',
  stopped: 'Stopped',
  error: 'Error',
  warning: 'Warning',
  available: 'Not installed',
}

const CONTRIBUTION_KINDS: Readonly<Record<PluginContributionKind, string>> = {
  commands: 'Commands',
  keybindings: 'Keybindings',
  tabKinds: 'Tab kinds',
  readers: 'Readers',
  handlers: 'Handlers',
  triggers: 'Triggers',
  textKinds: 'Text kinds',
  noteTypes: 'Note types',
  themes: 'Themes',
  slots: 'Slots',
  listeners: 'Listeners',
  viewCapabilities: 'View capabilities',
  dropHandlers: 'Drop handlers',
  frameOrigins: 'Frame origins',
  connectionTypes: 'Connection types',
  operations: 'Operations',
  permissions: 'Permissions',
  dependencies: 'Dependencies',
  services: 'Services',
  capabilities: 'Capabilities',
  activationEvents: 'Activation events',
}

/** UI copy of the plugin list and the plugin page. */
export const PLUGIN_WORDS = {
  settings: 'Settings',
  breadcrumb: 'Breadcrumb',
  /** Controls of a plugin. */
  pluginShowDetails: 'Show details',
  pluginHideDetails: 'Hide details',
  pluginRetry: 'Retry',
  pluginDisable: 'Disable',
  pluginEnable: 'Enable',
  pluginInstall: 'Install',
  pluginActions: 'Plugin actions',
  pluginSwitch: (name: string) => `Enabled ${name}`,
  pluginVersion: (version: string) => `v${version}`,
  /** The state of a plugin, on its row and on its page. */
  pluginState: PLUGIN_STATES,
  /** The accessible name of a plugin row: its name, and its state where it is not running. */
  pluginRow: (name: string, state: PluginState) =>
    state === 'enabled' ? name : `${name}, ${PLUGIN_STATES[state]}`,
  /** Plugin list and plugin page. */
  backToPlugins: 'All plugins',
  noPlugins: 'No plugins installed.',
  systemSettings: 'System',
  systemSettingsDetail: 'File names and performance',
  appearanceSettings: 'Appearance',
  appearanceSettingsDetail: 'Theme, light or dark, and sizes',
  searchPlugins: 'Search plugins…',
  searchPluginsLabel: 'Search plugins',
  clearSearch: 'Clear search',
  connections: 'Connections',
  connectionsDetail: 'Services and programs for plugins',
  /** The sections of a plugin page, in the order of its tabs. */
  pluginTabs: { settings: 'Settings', about: 'About', contributions: 'Contributes', data: 'Data' },
  pluginSections: 'Sections',
  contributionKinds: CONTRIBUTION_KINDS,
  noSettings: 'Nothing to set up.',
  noContributions: 'Contributes nothing.',
  noPluginsMatch: (query: string) => `No plugins match “${query}”`,
  noPluginsHint: 'Search by name, description or category, such as agents or readers.',
  /** The folder a plugin keeps in the shown vault, and its deletion. */
  pluginData: 'Plugin data',
  pluginDataDetail: (files: number) => `${countFiles(files)} in the plugin's folder of this vault.`,
  pluginDataDelete: 'Delete data',
  pluginDataAsking: (name: string, files: number) =>
    `Delete ${countFiles(files)} of ${name}? Its texts leave search. This cannot be undone.`,
  pluginDataConfirm: 'Delete',
  pluginDataKeep: 'Keep',
}
