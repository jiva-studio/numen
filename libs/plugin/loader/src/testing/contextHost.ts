import type { Disposable } from '@numen/plugin-sdk'
import { createMockNoticeHost, createMockSlotHost } from '@numen/plugin-sdk/testing'
import type { PluginContextHost } from '../hosting'

/** The parts a test replaces; every other member of a part does nothing. */
export interface ContextHostParts {
  readonly info?: Partial<PluginContextHost['info']>
  readonly registration?: Partial<PluginContextHost['registration']>
  readonly views?: Partial<PluginContextHost['views']>
  readonly settings?: Partial<PluginContextHost['settings']>
  readonly events?: Partial<PluginContextHost['events']>
  readonly tabs?: Partial<PluginContextHost['tabs']>
  readonly vault?: Partial<PluginContextHost['vault']>
  readonly window?: Partial<PluginContextHost['window']>
}

const NOTHING: Disposable = { dispose: () => undefined }

const IDLE: PluginContextHost = {
  info: {
    app: { name: 'numen', version: '2026.10.1', build: 884, platform: 'linux' },
    target: 'desktop.editor',
    openExternal: () => Promise.resolve(),
  },
  registration: {
    registerReader: () => NOTHING,
    registerSettingsView: () => NOTHING,
    registerCommand: () => NOTHING,
    registerTabKind: () => NOTHING,
  },
  views: {
    registerDropHandler: () => NOTHING,
    drops: { canDrop: () => false, drop: () => Promise.resolve([]) },
    registerView: () => NOTHING,
    registerSlotView: () => NOTHING,
    mountView: () => ({ focus: () => undefined, dispose: () => undefined }),
  },
  settings: { getSetting: () => undefined, onSettingChange: () => NOTHING },
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
  window: {
    getWorkspace: () => undefined,
    getCommands: () => undefined,
    isCommandListed: () => false,
    getWindow: () => undefined,
    getTasks: () => undefined,
    getBackgroundTasks: () => undefined,
    getHealth: () => undefined,
    getConnections: () => undefined,
    getSlots: () => createMockSlotHost(),
    getNotices: () => createMockNoticeHost(),
  },
}

/** A context host whose parts do nothing, with the members a test gives in place of them. */
export function createTestContextHost(parts: ContextHostParts = {}): PluginContextHost {
  return {
    info: { ...IDLE.info, ...parts.info },
    registration: { ...IDLE.registration, ...parts.registration },
    views: { ...IDLE.views, ...parts.views },
    settings: { ...IDLE.settings, ...parts.settings },
    events: { ...IDLE.events, ...parts.events },
    tabs: { ...IDLE.tabs, ...parts.tabs },
    vault: { ...IDLE.vault, ...parts.vault },
    window: { ...IDLE.window, ...parts.window },
  }
}
