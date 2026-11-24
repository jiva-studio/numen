import type { Disposable } from '@numen/plugin-sdk'
import { createMockNoticeHost, createMockSlotHost } from '@numen/plugin-sdk/testing'
import type { PluginHost } from '../hosting'

type PluginContextHost = PluginHost['contextHost']

/** The parts a test replaces; every other member of a part does nothing. */
export interface ContextHostParts {
  readonly registration?: Partial<PluginContextHost['registration']>
  readonly views?: Partial<PluginContextHost['views']>
  readonly settings?: Partial<PluginContextHost['settings']>
  readonly licenses?: Partial<PluginContextHost['settings']>
  readonly events?: Partial<PluginContextHost['events']>
  readonly tabs?: Partial<PluginContextHost['tabs']>
  readonly vault?: Partial<PluginContextHost['vault']>
  readonly ui?: Partial<PluginContextHost['ui']>
  readonly work?: Partial<PluginContextHost['work']>
}

const NOTHING: Disposable = { dispose: () => undefined }

const IDLE: PluginContextHost = {
  registration: {
    registerReader: () => NOTHING,
    registerSettingsView: () => NOTHING,
    registerCommand: () => NOTHING,
    registerTabKind: () => NOTHING,
  },
  views: {
    registerDropHandler: () => NOTHING,
    drops: { canDrop: () => false, drop: () => Promise.resolve([]) },
    registerEditorExtension: () => NOTHING,
    editorExtensions: { get: () => [], onChange: () => NOTHING, fail: () => undefined },
    registerView: () => NOTHING,
    registerSlotView: () => NOTHING,
    mountView: () => ({ focus: () => undefined, dispose: () => undefined }),
  },
  settings: {
    getSetting: () => undefined,
    watchSetting: () => NOTHING,
    getEditorSettings: () => undefined,
    watchEditorSettings: () => NOTHING,
    hasLicense: () => false,
    getLicenseKey: () => undefined,
    watchLicense: () => NOTHING,
  },
  events: {
    onVaultChange: () => NOTHING,
    onFileSelect: () => NOTHING,
    emitVaultChange: () => undefined,
    emitFileSelect: () => undefined,
  },
  tabs: { openTab: () => undefined, navigateTab: () => undefined, closeTab: () => undefined },
  vault: {
    getNotes: () => undefined,
    getRelations: () => undefined,
    getFiles: () => undefined,
    getBuffers: () => undefined,
    getVaults: () => undefined,
    getTexts: () => undefined,
  },
  ui: {
    app: { name: 'numen', version: '2026.10.1', build: 884, platform: 'linux' },
    target: 'desktop.editor',
    openExternal: () => Promise.resolve(),
    getWorkspace: () => undefined,
    getCommands: () => undefined,
    commandAccess: { isListed: () => false },
    getWindow: () => undefined,
    forPlugin: { getVaultFlags: () => undefined, getNotices: () => createMockNoticeHost() },
    getSlots: () => createMockSlotHost(),
    prompts: { getPicker: () => undefined, getDialogs: () => undefined },
  },
  work: {
    getTasks: () => undefined,
    getBackgroundTasks: () => undefined,
    getProcesses: () => undefined,
    getLogs: () => undefined,
    getHealth: () => undefined,
    getConnections: () => undefined,
  },
}

/** A context host whose parts do nothing, with the members a test gives in place of them. */
export function createTestContextHost(parts: ContextHostParts = {}): PluginContextHost {
  return {
    registration: { ...IDLE.registration, ...parts.registration },
    views: { ...IDLE.views, ...parts.views },
    settings: { ...IDLE.settings, ...parts.settings, ...parts.licenses },
    events: { ...IDLE.events, ...parts.events },
    tabs: { ...IDLE.tabs, ...parts.tabs },
    vault: { ...IDLE.vault, ...parts.vault },
    ui: { ...IDLE.ui, ...parts.ui },
    work: { ...IDLE.work, ...parts.work },
  }
}
