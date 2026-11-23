import type { PluginContributionKind } from './pluginInfo'
import type { PluginState } from './state'

// Stryker disable StringLiteral: interface copy
const countFiles = (files: number): string => (files === 1 ? '1 file' : `${files} files`)

const PERMISSION_MEANINGS: Readonly<Record<string, string>> = {
  'vault:read': 'Read your notes, files and tabs.',
  'vault:write': 'Change your notes and files, and act for you.',
  'vault:manage': 'Add, rename, remove and open vaults, and add a vault only it writes to.',
  'link:open': 'Open links in your browser.',
  'index:read': 'Read the titles and the search index of your notes.',
  'search:policy': 'Read and change how strict semantic search is.',
  'window:focus': 'Bring a note or a file into view, and list the open tabs.',
  'plugins:read': 'List the running plugins and their health.',
  'logs:read': 'Read the application log.',
  identity: 'Use the identity of this device to sign in to Numen services.',
}

const CONNECTION_PERMISSION = 'connections:use:'

/** What a plugin can do with a permission, in plain words. */
export const describePermission = (permission: string): string => {
  const known = PERMISSION_MEANINGS[permission]
  if (known !== undefined) return known
  if (permission.startsWith(CONNECTION_PERMISSION))
    return `Call ${permission.slice(CONNECTION_PERMISSION.length)} through your connections.`
  return 'Access the app does not describe.'
}

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
  editorExtensions: 'Editor extensions',
  frameOrigins: 'Frame origins',
  connectionTypes: 'Connection types',
  operations: 'Operations',
  licenses: 'Licenses',
  permissions: 'Permissions',
  dependencies: 'Dependencies',
  services: 'Services',
  capabilities: 'Capabilities',
  activationEvents: 'Activation events',
  searchFunctions: 'Search functions',
  tools: 'MCP tools',
}

/** UI copy of the plugin list and the plugin page. */
export const PLUGIN_WORDS = {
  settings: 'Settings',
  /** The bar at the top of a settings page. */
  back: 'Back',
  path: 'Path',
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
  noPlugins: 'No plugins installed.',
  systemSettings: 'System',
  systemSettingsDetail: 'File names and performance',
  appearanceSettings: 'Appearance',
  appearanceSettingsDetail: 'Theme, light or dark, and sizes',
  searchPlugins: 'Search plugins…',
  searchPluginsLabel: 'Search plugins',
  clearSearch: 'Clear search',
  editorSettings: 'Editor',
  editorSettingsDetail: 'Font, line height, line numbers and wrapping',
  searchSettings: 'Search',
  searchSettingsDetail: 'Reranking, limits and results',
  connections: 'Connections',
  connectionsDetail: 'Services and programs for plugins',
  identityAndLicenses: 'Identity and licenses',
  identityAndLicensesDetail: 'Account, this device and license keys',
  /** The sections of a plugin page, in the order of its tabs. */
  pluginTabs: {
    settings: 'Settings',
    slots: 'Slots',
    about: 'About',
    contributions: 'Contributes',
    data: 'Data',
  },
  pluginSections: 'Sections',
  contributionKinds: CONTRIBUTION_KINDS,
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
