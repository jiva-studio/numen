/** What a start cancelled at the limit registers through the host services. The plugin is invented. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Disposable, HostServices, PluginContext } from '@numen/plugin-sdk'
import type { LoaderHost, PluginParts } from './host'
import { createTestPluginLoader } from './testing/pluginLoader'

const HARBOR = 'org.example.harbor'

afterEach(() => {
  vi.useRealTimers()
})

function createRecordingParts(handed: Map<string, Disposable>): PluginParts {
  const hand = (name: string) => () => {
    const one = { dispose: vi.fn() }
    handed.set(name, one)
    return one
  }
  return {
    plugin: { identity: { id: HARBOR, subscriptions: [] } },
    host: {
      vault: {
        vaults: { watch: hand('vaults.watch') },
        buffers: { track: hand('buffers.track'), addFlushHandler: hand('buffers.addFlushHandler') },
      },
      ui: {
        commands: { getIds: () => [], watch: hand('commands.watch'), run: vi.fn() },
        window: { watch: hand('window.watch') },
        mountView: () => ({ focus: vi.fn(), ...hand('mountView')() }),
      },
    },
  } as unknown as PluginParts
}

describe('a start cancelled at the limit', () => {
  it('takes back what it registered through the host services, before and after the limit', async () => {
    vi.useFakeTimers()
    const handed = new Map<string, Disposable>()
    const pushed = { dispose: vi.fn() }
    const host = {
      createContext: () => createRecordingParts(handed),
      disposePlugin: vi.fn<(pluginId: string) => void>(),
      disposeAll: vi.fn(),
      getSetting: <T>() => undefined as T,
    } satisfies LoaderHost
    const loader = createTestPluginLoader({ host })
    loader.registerPluginModule(HARBOR, async () => ({
      activate: async (plugin: PluginContext, services: HostServices) => {
        services.ui.commands?.watch(() => undefined)
        services.vault.buffers?.track({} as never)
        await new Promise((resolve) => setTimeout(resolve, 30_000 + 1_000))
        services.vault.vaults?.watch(() => undefined)
        services.ui.window?.watch(() => undefined)
        services.vault.buffers?.addFlushHandler(async () => undefined)
        services.ui.mountView('note-editor', {} as HTMLElement, { path: 'a.md' })
        plugin.identity.subscriptions.push(pushed)
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
