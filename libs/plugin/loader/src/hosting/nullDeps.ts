import { ReportedError } from '@numen/result'
import type { NoticeHost } from '@numen/plugin-sdk'
import { createAppInfo } from '../appInfo'
import type {
  HostCommands,
  HostKinds,
  HostPlugins,
  HostSlots,
  HostTabs,
  HostVault,
  HostWindow,
  HostWork,
} from './types'

const stop = { dispose: () => undefined }

const DISCARDING_NOTICES: NoticeHost = { show: () => undefined, setTasks: () => undefined }

/** A window with no services: it opens no web links, shows no notices and reports nothing. */
export const NO_WINDOW: Omit<HostWindow, 'target'> = {
  app: createAppInfo({ getPlatform: () => 'linux' }),
  openExternal: () => Promise.reject(new ReportedError('This window cannot open web links')),
  onError: () => undefined,
  getWorkspace: () => undefined,
  getCommands: () => undefined,
  getWindow: () => undefined,
  getVaultFlags: () => undefined,
  getNotices: () => DISCARDING_NOTICES,
  getPicker: () => undefined,
  getDialogs: () => undefined,
  clearNotices: () => undefined,
}

/** A window with no plugin catalog: every plugin has a frontend and no backend, none fails and none has a setting or a license. */
export const NO_PLUGINS: HostPlugins = {
  hasPluginFrontend: () => true,
  hasPluginBackend: () => false,
  isPluginFailed: () => false,
  startPlugin: () => Promise.resolve(),
  startPluginsOf: () => Promise.resolve(),
  findPluginsOf: () => [],
  getSetting: () => undefined,
  watchSetting: () => stop,
  getEditorSettings: () => undefined,
  watchEditorSettings: () => stop,
  hasLicense: () => false,
  getLicenseKey: () => undefined,
  watchLicense: () => stop,
}

/** A window that declares no commands and runs no backend handler. */
export const NO_COMMANDS: HostCommands = {
  getCommandDeclarations: () => undefined,
  getHandlerDeclarations: () => undefined,
  listBackendHandlerItems: () => Promise.resolve([]),
  runBackendHandler: () => Promise.resolve(),
  reportCommandIssue: () => undefined,
  getContentTypes: () => ({}),
  getWindowValues: () => undefined,
}

/** A window with no slots. */
export const NO_SLOTS: HostSlots = {
  getSlotEntries: () => undefined,
  getSlotPoints: () => undefined,
  getSlotLayout: () => undefined,
  createSlotCommandFinder: () => () => undefined,
  getSlotWhenValues: () => undefined,
  reportSlotIssue: () => undefined,
  reportViewIssue: () => undefined,
}

/** A window that takes no reader or editor, closes no tab and reads and writes no note. */
export const NO_KINDS: Omit<HostKinds, 'registerKind'> = {
  registerExtensionReader: () => () => undefined,
  getReaderExtensions: () => [],
  registerEditor: () => undefined,
  closeTabsForPlugin: () => undefined,
  getNoteBuffer: () => undefined,
  readNote: undefined,
  writeNote: undefined,
}

/** A window that opens no tab: each open answers undefined. */
export const NO_TABS: HostTabs = {
  openTab: () => undefined,
  openTabAdjacent: () => undefined,
  navigateTab: () => undefined,
  closeTab: () => undefined,
  getTabState: () => undefined,
}

/** A window with no vault. */
export const NO_VAULT: HostVault = {
  getNotes: () => undefined,
  getRelations: () => undefined,
  getFiles: () => undefined,
  getVaults: () => undefined,
  getBuffers: () => undefined,
  getTexts: () => undefined,
  resolveLink: (_from, address) => Promise.resolve(address),
}

/** A window with no background work, and that drops the health items. */
export const NO_WORK: HostWork = {
  getTasks: () => undefined,
  getConnections: () => undefined,
  getBackgroundTasks: () => undefined,
  getProcesses: () => undefined,
  getLogs: () => undefined,
  health: { reportHealth: () => Promise.resolve(), clearHealth: () => Promise.resolve() },
}
