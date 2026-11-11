import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { PluginLoader } from '@numen/plugin-loader'
import { createPluginStart } from './createPluginStart'

const setup = () => {
  const plugins = ref([
    { id: 'md.example.notes', hasFrontend: true },
    { id: 'md.example.server', hasFrontend: false },
  ])
  const pluginLoader = {
    trackPlugins: vi.fn(() => Promise.resolve()),
    start: vi.fn(() => Promise.resolve()),
    close: vi.fn(() => Promise.resolve()),
    status: { waitForInterface: vi.fn(() => Promise.resolve()) },
  }
  const writeMessage = vi.fn()
  const start = createPluginStart({
    pluginLoader: pluginLoader as unknown as PluginLoader,
    getPlugins: () => plugins.value,
    waitForDeclaredCommands: () => Promise.resolve(),
    writeMessage,
  })
  return { start, plugins, pluginLoader, writeMessage }
}

describe('the plugins a window loads', () => {
  it('are tracked and started: the ones with a frontend', async () => {
    const { start, pluginLoader } = setup()

    await start.start()

    expect(pluginLoader.trackPlugins).toHaveBeenCalledExactlyOnceWith(['md.example.notes'])
    expect(pluginLoader.start).toHaveBeenCalledOnce()
  })

  it('are tracked again when the settings change which of them are on', async () => {
    const { plugins, pluginLoader } = setup()

    plugins.value = [{ id: 'md.example.sketch', hasFrontend: true }]
    await nextTick()

    expect(pluginLoader.trackPlugins).toHaveBeenCalledExactlyOnceWith(['md.example.sketch'])
  })

  it('report a track that failed', async () => {
    const { plugins, pluginLoader, writeMessage } = setup()
    pluginLoader.trackPlugins.mockRejectedValueOnce(new Error('no import'))

    plugins.value = []
    await nextTick()

    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledExactlyOnceWith(
        expect.stringContaining('Plugins were not tracked'),
        'error',
      ),
    )
  })

  it('are waited for through the interface and the first list of declared commands', async () => {
    const { start, pluginLoader } = setup()

    await start.waitForInterface()

    expect(pluginLoader.status.waitForInterface).toHaveBeenCalledOnce()
  })
})
