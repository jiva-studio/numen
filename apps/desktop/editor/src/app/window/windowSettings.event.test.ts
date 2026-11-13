/**
 * What the window does when a setting changes: one event reaches every reader of the setting,
 * whoever wrote it, and nothing reads the settings again on a timer.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { THEME_MODE, type ThemeModeCode } from '@numen/core-client'
import { createTestTabs } from '@/testing/tabs'
import { createConnectionStore } from '@/entities/connection'
import { createSettingsStore, SETTINGS_WORDS, type SettingEdit } from '@/entities/settings'
import { installSettingsCore, type SettingsCore } from '@/testing/settingsWire'
import type { MessageLog } from '@/shared/model/notices/messages'
import { WORDS } from '@/shared/ui/words'
import { createTestPluginSupport } from '@/testing/pluginSupport'
import { createWindowSettingsPart } from '@/testing/windowSettings'

const file = { current: {} as Record<string, unknown> }

const createFeed = () => {
  const pending: ((batch: readonly SettingEdit[]) => void)[] = []
  const queued: (readonly SettingEdit[])[] = []
  let signal: AbortSignal | undefined
  return {
    open: (given: AbortSignal): AsyncIterable<readonly SettingEdit[]> => {
      signal = given
      return {
        [Symbol.asyncIterator]: () => ({
          next: async () => {
            const batch =
              queued.shift() ??
              (await new Promise<readonly SettingEdit[]>((resolve) => pending.push(resolve)))
            return { done: false, value: batch }
          },
        }),
      }
    },
    push: (batch: readonly SettingEdit[]) => {
      const resolve = pending.shift()
      if (resolve) resolve(batch)
      else queued.push(batch)
    },
    isAborted: () => signal?.aborted ?? false,
  }
}

const settle = async () => {
  for (let step = 0; step < 50; step++) await Promise.resolve()
  await nextTick()
}

const PLUGIN = 'md.numen.eventful'

/** Puts back the transport that was installed before the core of the test. */
let restoreCore: (() => void) | undefined

const create = () => {
  file.current = { app: { appearance: { mode: 'light' }, naming: { syncTitleAndFilename: true } } }
  const feed = createFeed()
  const core = {
    getSettings: vi.fn(async () => ({
      written: JSON.stringify(file.current),
      path: '/config/numen.json',
      plugins: [],
      app: [],
    })),
    updateSettings: vi.fn(async () => undefined),
    watchSettings: vi.fn((signal: AbortSignal) => feed.open(signal)),
  } as unknown as SettingsCore
  restoreCore = installSettingsCore(core, { theme: THEME })
  const log = { getWriter: () => vi.fn() } as unknown as MessageLog
  const windowTabs = createTestTabs()
  const connections = createConnectionStore(log.getWriter('connections'))
  const settings = createWindowSettingsPart({
    rest: createSettingsStore(SETTINGS_WORDS, log.getWriter('configured')),
    connections,
    words: WORDS,
    log,
    windowTabs,
    onSizeChanged: vi.fn(),
    getSettingsViews: () => new Map(),
    getPlatform: () => 'linux',
  })
  return { settings, core, feed, windowTabs, connections }
}

const change = (feed: ReturnType<typeof createFeed>, path: readonly string[], value: unknown) => {
  let target = file.current
  for (const step of path.slice(0, -1)) {
    target[step] = { ...((target[step] as Record<string, unknown> | undefined) ?? {}) }
    target = target[step] as Record<string, unknown>
  }
  target[path[path.length - 1] as string] = value
  feed.push([{ at: path, value: JSON.stringify(value) }])
}

const MODES: Record<string, ThemeModeCode> = {
  system: THEME_MODE.SYSTEM,
  light: THEME_MODE.LIGHT,
  dark: THEME_MODE.DARK,
}

/** What the core answers about the themes: the mode the file holds. */
const THEME = {
  listThemes: () => {
    const app = (file.current['app'] ?? {}) as Record<string, Record<string, unknown>>
    const mode = app['appearance']?.['mode'] as string | undefined
    return {
      themes: [],
      applied: 'preset:numen',
      mode: MODES[mode ?? 'system'] ?? THEME_MODE.SYSTEM,
      interfaceScale: 1,
      textScale: 1,
      interfaceScaleBounds: { min: 0.8, max: 1.4 },
      textScaleBounds: { min: 0.8, max: 1.4 },
    }
  },
  readTheme: () => ({ css: '' }),
  async *watchThemes() {
    yield* []
  },
}

describe('a setting that changes', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    restoreCore?.()
    restoreCore = undefined
    vi.useRealTimers()
  })

  it('shows its new value in the settings of the window when the event arrives', async () => {
    const { settings, feed } = create()
    await settings.start()
    expect(settings.rest.read.getSetting(['app', 'appearance', 'mode'])).toBe('light')

    change(feed, ['app', 'appearance', 'mode'], 'dark')
    await vi.advanceTimersByTimeAsync(0)
    await settle()

    expect(settings.rest.read.getSetting(['app', 'appearance', 'mode'])).toBe('dark')
    settings.close()
  })

  it('shows the new value of a plugin setting when the event arrives', async () => {
    const { settings, feed } = create()
    await settings.start()

    change(feed, [PLUGIN, 'mode'], 'b')
    await vi.advanceTimersByTimeAsync(0)
    await settle()

    expect(settings.rest.read.getSetting([PLUGIN, 'mode'])).toBe('b')
    settings.close()
  })

  it('draws the mode in the window when the event arrives', async () => {
    const { settings, feed } = create()
    await settings.start()
    await vi.waitFor(() => expect(settings.appearance.state.mode.value).toBe('light'))

    change(feed, ['app', 'appearance', 'mode'], 'dark')

    await vi.waitFor(() => expect(settings.appearance.state.mode.value).toBe('dark'))
    settings.close()
  })

  it('shows the naming setting in the palette when the event arrives', async () => {
    const { settings, feed } = create()
    await settings.start()
    await settle()
    expect(settings.rest.read.getSetting(['app', 'naming', 'syncTitleAndFilename'])).toBe(true)

    change(feed, ['app', 'naming', 'syncTitleAndFilename'], false)
    await vi.advanceTimersByTimeAsync(0)
    await settle()

    expect(settings.rest.read.getSetting(['app', 'naming', 'syncTitleAndFilename'])).toBe(false)
    settings.close()
  })

  it('calls the handler a plugin registered for the setting once with the new value', async () => {
    const { settings, feed, windowTabs } = create()
    const support = createTestPluginSupport({
      windowTabs,
      tabOpeners: {
        registerExtensionReader: vi.fn(() => () => undefined),
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      rest: settings.rest,
      getServices: () => ({}),
    })
    await settings.start()
    const handler = vi.fn()
    support.pluginHost.lifecycle.createContext(PLUGIN).settings.onChange('mode', handler)

    change(feed, [PLUGIN, 'mode'], 'b')
    await vi.advanceTimersByTimeAsync(0)
    await settle()

    expect(handler).toHaveBeenCalledExactlyOnceWith('b')
    settings.close()
  })

  it('does not call the handler of a plugin for a setting of another plugin', async () => {
    const { settings, feed, windowTabs } = create()
    const support = createTestPluginSupport({
      windowTabs,
      tabOpeners: {
        registerExtensionReader: vi.fn(() => () => undefined),
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      rest: settings.rest,
      getServices: () => ({}),
    })
    await settings.start()
    const handler = vi.fn()
    support.pluginHost.lifecycle.createContext(PLUGIN).settings.onChange('mode', handler)

    change(feed, ['md.numen.other', 'mode'], 'b')
    await vi.advanceTimersByTimeAsync(0)
    await settle()

    expect(handler).not.toHaveBeenCalled()
    settings.close()
  })

  it('tells the connections of a plugin of a change of the connections, the detected ones or an enabled plugin', async () => {
    const { settings, feed, windowTabs, connections } = create()
    const support = createTestPluginSupport({
      windowTabs,
      tabOpeners: {
        registerExtensionReader: vi.fn(() => () => undefined),
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      rest: settings.rest,
      getServices: () => ({}),
      subscribeConnectionChanges: connections.changes.subscribe,
    })
    await settings.start()
    const handler = vi.fn()
    support.pluginHost.lifecycle.createContext(PLUGIN).work.connections!.onChange!(handler)

    change(feed, ['app', 'appearance', 'mode'], 'dark')
    await vi.advanceTimersByTimeAsync(0)
    await settle()
    expect(handler).not.toHaveBeenCalled()

    const pushes = [
      () => feed.push([{ at: ['detectedConnections'], value: '["c-gemini"]' }]),
      () => change(feed, ['connections'], { 'c-orchard': { type: 'acp' } }),
      () => change(feed, ['md.numen.agents.acp', 'enabled'], false),
    ]
    for (const push of pushes) {
      push()
      await vi.advanceTimersByTimeAsync(0)
      await settle()
    }

    expect(handler).toHaveBeenCalledTimes(3)
    settings.close()
  })

  it('is not read again at 1.5, 3.5, 7, 11 or 16 seconds after the window wrote it', async () => {
    const { settings, core } = create()
    await settings.start()
    await settle()
    const { installation } = settings.configured.state
    const reads = vi.mocked(core.getSettings).mock.calls.length

    await (installation.settings.write([
      { at: ['app', 'appearance', 'mode'], value: '"dark"' },
    ]) as unknown as Promise<void>)
    for (const second of [1.5, 3.5, 7, 11, 16, 20]) {
      await vi.advanceTimersByTimeAsync(second * 1000)
      await settle()
    }

    expect(vi.mocked(core.getSettings).mock.calls.length).toBeLessThanOrEqual(reads + 1)
    settings.close()
  })

  it('is read no more than once after the window started it, however long the window stays open', async () => {
    const { settings, core } = create()
    await settings.start()
    await settle()
    const reads = vi.mocked(core.getSettings).mock.calls.length

    await vi.advanceTimersByTimeAsync(60_000)
    await settle()

    expect(vi.mocked(core.getSettings).mock.calls.length).toBe(reads)
    settings.close()
  })

  it('is ignored by a window that closed', async () => {
    const { settings, feed } = create()
    await settings.start()
    settings.close()

    change(feed, ['app', 'appearance', 'mode'], 'dark')
    await vi.advanceTimersByTimeAsync(0)
    await settle()

    expect(feed.isAborted()).toBe(true)
    expect(settings.rest.read.getSetting(['app', 'appearance', 'mode'])).toBe('light')
  })

  it('does not change anything when the event that opens the stream holds no change', async () => {
    const { settings, feed } = create()
    await settings.start()

    feed.push([])
    await vi.advanceTimersByTimeAsync(0)
    await settle()

    expect(settings.rest.read.getSetting(['app', 'appearance', 'mode'])).toBe('light')
    settings.close()
  })
})
