import { createMockNoticeHost, createMockSlotHost } from '@numen/plugin-sdk/testing'
import { createIdleContextHost, type PluginContextHost } from '../hosting'

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

/** A context host whose parts do nothing, with the members a test gives in place of them. */
export function createTestContextHost(parts: ContextHostParts = {}): PluginContextHost {
  const idle = createIdleContextHost({
    info: {
      app: { name: 'numen', version: '2026.10.1', build: 884, platform: 'linux' },
      target: 'desktop.editor',
      openExternal: () => Promise.resolve(),
    },
    registration: { registerTabKind: () => ({ dispose: () => undefined }) },
    views: {
      registerView: () => ({ dispose: () => undefined }),
      registerSlotView: () => ({ dispose: () => undefined }),
      mountView: () => ({ focus: () => undefined, dispose: () => undefined }),
    },
    window: { getSlots: () => createMockSlotHost(), getNotices: () => createMockNoticeHost() },
  })
  return {
    info: { ...idle.info, ...parts.info },
    registration: { ...idle.registration, ...parts.registration },
    views: { ...idle.views, ...parts.views },
    settings: { ...idle.settings, ...parts.settings },
    events: { ...idle.events, ...parts.events },
    tabs: { ...idle.tabs, ...parts.tabs },
    vault: { ...idle.vault, ...parts.vault },
    window: { ...idle.window, ...parts.window },
  }
}
