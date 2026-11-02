/** What a window starts when it mounts and closes when it unmounts. */
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { PLUGINS_WAIT_MS, useAppBootstrap, type BootstrapDeps } from './useAppBootstrap'

const setup = (overrides: Partial<BootstrapDeps> = {}) => {
  const calls: string[] = []
  const note = (name: string) => () => {
    calls.push(name)
  }
  const noteLater = (name: string) => async () => {
    calls.push(name)
  }
  const deps: BootstrapDeps = {
    loadVaults: noteLater('loadVaults'),
    startSettings: noteLater('startSettings'),
    startWindow: noteLater('startWindow'),
    startEditing: note('startEditing'),
    startLayout: noteLater('startLayout'),
    startPlugins: noteLater('startPlugins'),
    closeWindow: note('closeWindow'),
    closeEditing: note('closeEditing'),
    closeTabs: note('closeTabs'),
    closeSettings: note('closeSettings'),
    reportError: vi.fn(),
    ...overrides,
  }
  const view = mount(
    defineComponent({
      setup() {
        useAppBootstrap(deps)
        return () => h('div')
      },
    }),
  )
  return { calls, view }
}

describe('a window mounting', () => {
  it('loads the vaults, settings, plugins and layout, then starts the window and the editing', async () => {
    const { calls } = setup()

    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(calls).toStrictEqual([
      'loadVaults',
      'startSettings',
      'startPlugins',
      'startLayout',
      'startWindow',
      'startEditing',
    ])
  })
})

describe('a window unmounting', () => {
  it('closes the window, the editing, the tabs and the settings, in that order', async () => {
    const { calls, view } = setup()
    await new Promise((resolve) => setTimeout(resolve, 0))
    calls.length = 0

    view.unmount()

    expect(calls).toStrictEqual(['closeWindow', 'closeEditing', 'closeTabs', 'closeSettings'])
  })
})

describe('a window mounting while a plugin does not start', () => {
  it('opens the layout after the wait and lets the plugin go on starting', async () => {
    vi.useFakeTimers()
    try {
      let finishPlugins: () => void = () => undefined
      const { calls } = setup({
        startPlugins: () =>
          new Promise<void>((resolve) => {
            finishPlugins = resolve
          }),
      })

      await vi.advanceTimersByTimeAsync(PLUGINS_WAIT_MS - 1)
      expect(calls).not.toContain('startLayout')
      await vi.advanceTimersByTimeAsync(1)
      expect(calls).toContain('startLayout')
      expect(calls).toContain('startWindow')

      finishPlugins()
    } finally {
      vi.useRealTimers()
    }
  })

  it('reports a plugin start that fails and opens the layout', async () => {
    const failure = new Error('plugins stopped')
    const reportError = vi.fn()
    const { calls } = setup({ startPlugins: () => Promise.reject(failure), reportError })

    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(reportError).toHaveBeenCalledWith(failure)
    expect(calls).toContain('startLayout')
  })
})
