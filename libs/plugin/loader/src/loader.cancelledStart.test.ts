/** What a start cancelled at the limit registers through the services of its context. The plugin is invented. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Disposable, PluginContext } from '@numen/plugin-sdk'
import type { LoaderHost } from './host'
import { createPluginLoader } from './loader'

const HARBOR = 'org.example.harbor'

afterEach(() => {
  vi.useRealTimers()
})

function createRecordingContext(handed: Map<string, Disposable>): PluginContext {
  const hand = (name: string) => () => {
    const one = { dispose: vi.fn() }
    handed.set(name, one)
    return one
  }
  return {
    plugin: { id: HARBOR, subscriptions: [] },
    vault: {
      vaults: { watch: hand('vaults.watch') },
      buffers: { track: hand('buffers.track'), addFlushHandler: hand('buffers.addFlushHandler') },
    },
    ui: {
      commands: { getIds: () => [], watch: hand('commands.watch'), run: vi.fn() },
      window: { watch: hand('window.watch') },
      mountView: () => ({ focus: vi.fn(), ...hand('mountView')() }),
    },
  } as unknown as PluginContext
}

describe('a start cancelled at the limit', () => {
  it('takes back what it registered through the services of its context, before and after the limit', async () => {
    vi.useFakeTimers()
    const handed = new Map<string, Disposable>()
    const pushed = { dispose: vi.fn() }
    const host = {
      createContext: () => createRecordingContext(handed),
      disposePlugin: vi.fn<(pluginId: string) => void>(),
      disposeAll: vi.fn(),
      getSetting: <T>() => undefined as T,
    } satisfies LoaderHost
    const loader = createPluginLoader({ host })
    loader.registerPluginModule(HARBOR, async () => ({
      activate: async (context: PluginContext) => {
        context.ui.commands?.watch(() => undefined)
        context.vault.buffers?.track({} as never)
        await new Promise((resolve) => setTimeout(resolve, 30_000 + 1_000))
        context.vault.vaults?.watch(() => undefined)
        context.ui.window?.watch(() => undefined)
        context.vault.buffers?.addFlushHandler(async () => undefined)
        context.ui.mountView('note-editor', {} as HTMLElement, { path: 'a.md' })
        context.plugin.subscriptions.push(pushed)
      },
    }))

    await loader.start()
    const started = loader.status.startPlugin(HARBOR)
    await vi.advanceTimersByTimeAsync(30_000 + 2_000)
    await started

    expect(loader.status.getPluginFailure(HARBOR)).toBe('the plugin did not start within 30 s')
    expect(
      [...handed]
        .filter(([, one]) => vi.mocked(one.dispose).mock.calls.length !== 1)
        .map(([name]) => name),
    ).toStrictEqual([])
    expect([...handed.keys()].sort()).toStrictEqual([
      'buffers.addFlushHandler',
      'buffers.track',
      'commands.watch',
      'mountView',
      'vaults.watch',
      'window.watch',
    ])
    expect(pushed.dispose).toHaveBeenCalledTimes(1)
  })
})
