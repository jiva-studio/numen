import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { INTERFACE_WAIT_MS } from '@numen/plugin-loader'
import { installCoreHandlers } from '@numen/core-client/testing'
import { useAppStart, type PluginStarter } from './useAppStart'

let restoreCore: () => void
const getPlugins = vi.fn()

const createLoader = () => {
  const calls: string[] = []
  let settleInterface: () => void = () => undefined
  const failed: { id: string; error: string }[] = []
  const loader: PluginStarter = {
    trackPlugins: async (ids) => {
      calls.push(`track ${ids.join(',')}`)
    },
    start: async () => {
      calls.push('start')
    },
    close: async () => {
      calls.push('close')
    },
    status: {
      waitForInterface: () => new Promise<void>((resolve) => (settleInterface = resolve)),
      getFailedPlugins: () => failed,
    },
  }
  return { loader, calls, failed, settleInterface: () => settleInterface() }
}

const mountStart = (loader: PluginStarter) => {
  let result!: ReturnType<typeof useAppStart>
  const wrapper = mount(
    defineComponent({
      setup() {
        result = useAppStart(loader)
        return () => h('div')
      },
    }),
  )
  return { wrapper, start: result }
}

beforeEach(() => {
  getPlugins.mockResolvedValue({ plugins: [{ id: 'md.numen.cards' }] })
  restoreCore = installCoreHandlers({ getPlugins: (request: unknown) => getPlugins(request) })
})
afterEach(() => {
  restoreCore()
  vi.useRealTimers()
})

describe('useAppStart', () => {
  it('tracks the listed plugins, starts them and closes them when the window unmounts', async () => {
    const { loader, calls, settleInterface } = createLoader()
    const { wrapper, start } = mountStart(loader)

    expect(start.isLoading.value).toBe(true)
    expect(start.isStarted.value).toBe(false)
    await vi.waitFor(() => expect(start.isLoading.value).toBe(false))
    settleInterface()
    await flushPromises()
    expect(start.isStarted.value).toBe(true)
    expect(calls).toEqual(['track md.numen.cards', 'start'])

    wrapper.unmount()
    expect(calls).toContain('close')
  })

  it('draws the window once the wait for the interface plugins has passed', async () => {
    vi.useFakeTimers()
    const { loader } = createLoader()
    getPlugins.mockReturnValue(new Promise(() => undefined))
    const { start } = mountStart(loader)

    await vi.advanceTimersByTimeAsync(INTERFACE_WAIT_MS - 1)
    expect(start.isStarted.value).toBe(false)
    await vi.advanceTimersByTimeAsync(1)
    expect(start.isStarted.value).toBe(true)
    expect(start.isLoading.value).toBe(true)
  })

  it('says why there is no view: the list failed, or a plugin did not start', async () => {
    const { loader, failed } = createLoader()
    failed.push({ id: 'md.numen.cards', error: 'no entry' })
    getPlugins.mockRejectedValue(new Error('the core is down'))
    const { start } = mountStart(loader)

    await vi.waitFor(() => expect(start.isStarted.value).toBe(true))
    expect(start.failures.value).toHaveLength(2)
    expect(start.failures.value[1]).toBe('md.numen.cards: no entry')
  })
})
