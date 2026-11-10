import { createMockNoticeHost, createMockSlotHost } from '@numen/plugin-sdk/testing'
import { createIdleContextHost } from '../hosting/idleContextHost'
import type { PluginContextHost } from '../hosting/types'

/** The ports a test replaces; every other member of a port does nothing. */
export interface ContextHostParts {
  readonly app?: PluginContextHost['app']
  readonly target?: PluginContextHost['target']
  readonly openExternal?: PluginContextHost['openExternal']
  readonly registrations?: Partial<PluginContextHost['registrations']>
  readonly settings?: Partial<PluginContextHost['settings']>
  readonly events?: Partial<PluginContextHost['events']>
  readonly tabs?: Partial<PluginContextHost['tabs']>
  readonly vault?: Partial<PluginContextHost['vault']>
  readonly ui?: Partial<PluginContextHost['ui']>
  readonly work?: Partial<PluginContextHost['work']>
}

/** A context host whose ports do nothing, with the members a test gives in place of them. */
export function createTestContextHost(parts: ContextHostParts = {}): PluginContextHost {
  const idle = createIdleContextHost({
    app: parts.app ?? { name: 'numen', version: '2026.10.1', build: 884, platform: 'linux' },
    target: parts.target ?? 'desktop.editor',
    openExternal: parts.openExternal ?? (() => Promise.resolve()),
    registrations: {
      registerTabKind: () => ({ dispose: () => undefined }),
      registerView: () => ({ dispose: () => undefined }),
      registerSlotView: () => ({ dispose: () => undefined }),
      mountView: () => ({ focus: () => undefined, dispose: () => undefined }),
    },
    ui: { getSlots: () => createMockSlotHost(), getNotices: () => createMockNoticeHost() },
  })
  return {
    ...idle,
    registrations: { ...idle.registrations, ...parts.registrations },
    settings: { ...idle.settings, ...parts.settings },
    events: { ...idle.events, ...parts.events },
    tabs: { ...idle.tabs, ...parts.tabs },
    vault: { ...idle.vault, ...parts.vault },
    ui: { ...idle.ui, ...parts.ui },
    work: { ...idle.work, ...parts.work },
  }
}
