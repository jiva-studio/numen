import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { PluginContext } from '@numen/plugin-sdk'
import { activateEntry, type ActivateContext } from './activation'
import { startInterfacePlugins } from './interfaceStart'
import { INTERFACE_WAIT_MS, PLUGIN_START_LIMIT_MS } from './limits'
import { waitWithinInterfaceLimit } from './interfaceWait'
import { createStartLimit } from './startLimit'
import { startLoader } from './startLoader'
import { createPluginStyles } from '../styles'

const track = (wait: Promise<void>) => {
  const state = { isEnded: false }
  wait.then(
    () => {
      state.isEnded = true
    },
    () => undefined,
  )
  return state
}

describe('the wait for the plugins of the window slots', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('ends at INTERFACE_WAIT_MS when the work does not end', async () => {
    const wait = track(waitWithinInterfaceLimit(new Promise(() => undefined)))

    await vi.advanceTimersByTimeAsync(INTERFACE_WAIT_MS - 1)
    expect(wait.isEnded).toBe(false)
    await vi.advanceTimersByTimeAsync(1)
    expect(wait.isEnded).toBe(true)
  })

  it('ends when the work ends first, and leaves no timer behind', async () => {
    const wait = track(waitWithinInterfaceLimit(Promise.resolve()))

    await vi.advanceTimersByTimeAsync(0)

    expect(wait.isEnded).toBe(true)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('ends without failing when the work fails', async () => {
    await expect(
      waitWithinInterfaceLimit(Promise.reject(new Error('stopped'))),
    ).resolves.toBeUndefined()
  })

  it('never ends once the signal aborts, and leaves no timer behind', async () => {
    const aborting = new AbortController()
    const wait = track(waitWithinInterfaceLimit(new Promise(() => undefined), aborting.signal))

    aborting.abort()
    await vi.advanceTimersByTimeAsync(INTERFACE_WAIT_MS)

    expect(wait.isEnded).toBe(false)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('does not start on a signal that has already aborted', async () => {
    const aborting = new AbortController()
    aborting.abort()
    const wait = track(waitWithinInterfaceLimit(Promise.resolve(), aborting.signal))

    await vi.advanceTimersByTimeAsync(INTERFACE_WAIT_MS)

    expect(wait.isEnded).toBe(false)
  })

  it('tracks start limits and cleans up deadline timers in startInterfacePlugins and startLoader', async () => {
    const limit = createStartLimit()
    expect(limit.isPassed()).toBe(false)
    await vi.advanceTimersByTimeAsync(PLUGIN_START_LIMIT_MS)
    expect(limit.isPassed()).toBe(true)
    await expect(limit.expired).resolves.toBe('expired')
    limit.stop()

    const startedIds: string[] = []
    const fast = await startInterfacePlugins({
      ids: ['ui-1', 'non-ui', 'err-ui'],
      isInterface: async (id) => {
        if (id === 'err-ui') throw new Error('manifest err')
        return id === 'ui-1'
      },
      load: async (id) => `mod:${id}`,
      start: async (id) => {
        startedIds.push(id)
      },
    })
    await fast.started
    expect(startedIds).toStrictEqual(['ui-1'])
    expect(vi.getTimerCount()).toBe(0)

    const lateOrder: string[] = []
    const resolvers = new Map<string, () => void>()
    const slowPromise = startInterfacePlugins({
      ids: ['slow-1', 'slow-2', 'slow-3'],
      isInterface: async () => true,
      load: async (id) => id,
      start: (id) => {
        lateOrder.push(id)
        return new Promise<void>((resolve) => {
          resolvers.set(id, resolve)
        })
      },
    })
    await vi.advanceTimersByTimeAsync(0)
    expect(lateOrder).toStrictEqual(['slow-1'])

    await vi.advanceTimersByTimeAsync(INTERFACE_WAIT_MS)
    const slowResult = await slowPromise
    expect(lateOrder).toStrictEqual(['slow-1', 'slow-2', 'slow-3'])
    expect(vi.getTimerCount()).toBe(0)
    for (const resolve of resolvers.values()) resolve()
    await slowResult.started

    const listened: string[] = []
    const checkedInterface: string[] = []
    const activate = vi.fn(async () => true)
    const onInterfaceStarted = vi.fn()
    await startLoader(
      {
        pluginIds: ['enabled-ui', 'disabled-ui'],
        listen: (id) => listened.push(id),
        isEnabled: (id) => id === 'enabled-ui',
        isInterface: async (id) => {
          checkedInterface.push(id)
          return true
        },
        load: async () => undefined,
        onInterfaceStarted,
      },
      activate,
    )
    expect(listened).toStrictEqual(['enabled-ui', 'disabled-ui'])
    expect(checkedInterface).toStrictEqual(['enabled-ui'])
    expect(activate).not.toHaveBeenCalled()
    expect(onInterfaceStarted).toHaveBeenCalledOnce()
    expect(vi.getTimerCount()).toBe(0)
  })

  it('handles overlapping starts, pre-existing active/failure entries, and expired starts in activateEntry', async () => {
    const createCtx = (): ActivateContext => ({
      baseUrl: '/plugin',
      fetchFn: vi.fn() as never,
      importFn: vi.fn(),
      manifests: new Map(),
      styles: createPluginStyles(),
      loaders: new Map(),
      host: {
        createContext: vi.fn((id: string) => ({
          plugin: {
            identity: { id, target: 'desktop.editor', subscriptions: [] },
            register: {},
            settings: {},
          },
          host: { vault: {}, tabs: {}, ui: {}, work: {}, events: {} },
        })) as never,
        disposePlugin: vi.fn(),
        disposeAll: vi.fn(),
        markPluginStarting: vi.fn(),
        markPluginStarted: vi.fn(),
      },
      activePlugins: { value: new Map() },
      failures: { value: new Map([['p', 'previous error']]) },
      reportError: vi.fn(),
      starts: new Map(),
    })

    const activeCtx = createCtx()
    activeCtx.activePlugins.value = new Map([['p', { pluginId: 'p' }]])
    const stopSpy = vi.fn()
    expect(
      await activateEntry(activeCtx, 'p', {
        limit: { expired: new Promise(() => undefined), isPassed: () => false, stop: stopSpy },
      }),
    ).toBe(true)
    expect(stopSpy).toHaveBeenCalledOnce()

    const ctx = createCtx()
    let failureDuringActivate = true
    let lateSubDisposed = false
    let capturedSubs: { push: (...d: { dispose: () => void }[]) => number } | undefined
    const ok = await activateEntry(ctx, 'p', {
      module: {
        activate: (plugin: PluginContext) => {
          failureDuringActivate = ctx.failures.value.has('p')
          capturedSubs = plugin.identity.subscriptions
        },
      },
    })
    expect(ok).toBe(true)
    expect(failureDuringActivate).toBe(false)
    expect(ctx.failures.value.has('p')).toBe(false)
    capturedSubs?.push({
      dispose: () => {
        lateSubDisposed = true
      },
    })
    expect(lateSubDisposed).toBe(false)

    const overlapCtx = createCtx()
    let resolveFirst: (() => void) | undefined
    const firstRun = activateEntry(overlapCtx, 'p', {
      module: {
        activate: () =>
          new Promise<void>((r) => {
            resolveFirst = r
          }),
      },
    })
    await vi.advanceTimersByTimeAsync(0)
    expect(overlapCtx.host.markPluginStarting).toHaveBeenCalledTimes(1)

    const secondRun = activateEntry(overlapCtx, 'p', {
      module: { activate: () => new Promise(() => undefined) },
    })
    await vi.advanceTimersByTimeAsync(0)
    expect(overlapCtx.starts.get('p')).toBe(2)

    resolveFirst?.()
    expect(await firstRun).toBe(true)
    expect(overlapCtx.host.markPluginStarted).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(PLUGIN_START_LIMIT_MS)
    expect(await secondRun).toBe(false)
    expect(overlapCtx.failures.value.get('p')).toBe('the plugin did not start within 30 s')

    const expiredResolveCtx = createCtx()
    let rejectLoader: ((err: Error) => void) | undefined
    expiredResolveCtx.loaders.set(
      'p',
      () =>
        new Promise((_, reject) => {
          rejectLoader = reject
        }),
    )
    const expiredThrowRun = activateEntry(expiredResolveCtx, 'p', {})
    await vi.advanceTimersByTimeAsync(PLUGIN_START_LIMIT_MS)
    expect(await expiredThrowRun).toBe(false)
    vi.mocked(expiredResolveCtx.host.disposePlugin).mockClear()
    expiredResolveCtx.starts.set('p', 99)
    rejectLoader?.(new Error('late loader failure'))
    await vi.advanceTimersByTimeAsync(0)
    expect(expiredResolveCtx.host.disposePlugin).not.toHaveBeenCalled()

    const passedAfterResolve = createCtx()
    expect(
      await activateEntry(passedAfterResolve, 'p', {
        module: { activate: vi.fn() },
        limit: { expired: new Promise(() => undefined), isPassed: () => true, stop: vi.fn() },
      }),
    ).toBe(false)
    expect(passedAfterResolve.failures.value.get('p')).toBe('the plugin did not start within 30 s')

    let passedInActivate = false
    const passedAfterActivate = createCtx()
    expect(
      await activateEntry(passedAfterActivate, 'p', {
        module: {
          activate: () => {
            passedInActivate = true
          },
        },
        limit: {
          expired: new Promise(() => undefined),
          isPassed: () => passedInActivate,
          stop: vi.fn(),
        },
      }),
    ).toBe(false)
    expect(passedAfterActivate.host.disposePlugin).toHaveBeenCalledWith('p')
    expect(passedAfterActivate.failures.value.get('p')).toBe('the plugin did not start within 30 s')
  })
})
