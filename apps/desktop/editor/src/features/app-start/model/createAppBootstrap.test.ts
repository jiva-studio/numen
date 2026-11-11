/** What a window starts when it mounts and closes when it unmounts. */
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createAppBootstrap, type BootstrapDeps } from './createAppBootstrap'

const STEP_LIMIT_MS = 10_000

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
  let started: ReturnType<typeof createAppBootstrap> | undefined
  const view = mount(
    defineComponent({
      setup() {
        started = createAppBootstrap(deps)
        return () => h('div')
      },
    }),
  )
  return { calls, view, bootstrap: () => started! }
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

      await vi.advanceTimersByTimeAsync(3_000 - 1)
      expect(calls).not.toContain('startLayout')
      await vi.advanceTimersByTimeAsync(1)
      expect(calls).toContain('startLayout')
      expect(calls).toContain('startWindow')

      finishPlugins()
    } finally {
      vi.useRealTimers()
    }
  })

  it('is drawn only once the plugins started or the wait for them passed', async () => {
    vi.useFakeTimers()
    try {
      const { bootstrap } = setup({ startPlugins: () => new Promise<void>(() => undefined) })

      await vi.advanceTimersByTimeAsync(3_000 - 1)
      expect(bootstrap().isStarted.value).toBe(false)
      await vi.advanceTimersByTimeAsync(1)
      expect(bootstrap().isStarted.value).toBe(true)
    } finally {
      vi.useRealTimers()
    }
  })

  it('is drawn as soon as the plugins of the window slots started, while the others go on', async () => {
    const { bootstrap, calls } = setup({
      startPlugins: () => new Promise<void>(() => undefined),
      waitForInterfacePlugins: () => Promise.resolve(),
    })

    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(bootstrap().isStarted.value).toBe(true)
    expect(calls).toContain('startLayout')
  })

  it('is drawn at 3 s while the settings have not answered', async () => {
    vi.useFakeTimers()
    try {
      const { bootstrap, calls } = setup({
        startSettings: () => new Promise<void>(() => undefined),
      })

      await vi.advanceTimersByTimeAsync(3_000 - 1)
      expect(bootstrap().isStarted.value).toBe(false)
      await vi.advanceTimersByTimeAsync(1)

      expect(bootstrap().isStarted.value).toBe(true)
      expect(calls).not.toContain('startPlugins')
    } finally {
      vi.useRealTimers()
    }
  })

  it('waits for the plugins of the window slots, and not only for every plugin to have started', async () => {
    vi.useFakeTimers()
    try {
      const { bootstrap } = setup({
        waitForInterfacePlugins: () => new Promise<void>(() => undefined),
      })

      await vi.advanceTimersByTimeAsync(3_000 - 1)
      expect(bootstrap().isStarted.value).toBe(false)
      await vi.advanceTimersByTimeAsync(1)
      expect(bootstrap().isStarted.value).toBe(true)
    } finally {
      vi.useRealTimers()
    }
  })

  it('is drawn as soon as the plugins started', async () => {
    const { bootstrap } = setup()

    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(bootstrap().isStarted.value).toBe(true)
  })

  it('is drawn when a step before the plugins fails, to show what did not load', async () => {
    const { bootstrap } = setup({ loadVaults: () => Promise.reject(new Error('vaults gone')) })

    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(bootstrap().isStarted.value).toBe(true)
    expect(bootstrap().failure.value).toBe('vaults gone')
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

describe('a window mounting while the settings never answer', () => {
  it('shows what did not load, opens nothing, and loads once the person retries', async () => {
    vi.useFakeTimers()
    try {
      let answers = 0
      const { calls, bootstrap } = setup({
        startSettings: () => {
          answers += 1
          return answers === 1 ? new Promise<void>(() => undefined) : Promise.resolve()
        },
      })

      await vi.advanceTimersByTimeAsync(STEP_LIMIT_MS - 1)
      expect(bootstrap().failure.value).toBe('')
      await vi.advanceTimersByTimeAsync(1)
      expect(bootstrap().failure.value).toBe('The settings request did not answer in 10 s')
      expect(calls).not.toContain('startLayout')
      expect(calls).not.toContain('startWindow')

      await bootstrap().retry()

      expect(bootstrap().failure.value).toBe('')
      expect(calls).toContain('startLayout')
      expect(calls).toContain('startWindow')
    } finally {
      vi.useRealTimers()
    }
  })

  it('shows a vault list that fails to load', async () => {
    const { bootstrap } = setup({ loadVaults: () => Promise.reject(new Error('no vaults')) })

    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(bootstrap().failure.value).toBe('no vaults')
  })

  it('starts with an empty failure before mounting and skips startPlugins when omitted', async () => {
    let initialFailure = 'unset'
    let bootstrap!: ReturnType<typeof createAppBootstrap>
    const startLayout = vi.fn(async () => undefined)
    const view = mount(
      defineComponent({
        setup() {
          bootstrap = createAppBootstrap({
            loadVaults: async () => undefined,
            startSettings: async () => undefined,
            startWindow: async () => undefined,
            startEditing: () => undefined,
            startLayout,
            closeWindow: () => undefined,
            closeEditing: () => undefined,
            closeTabs: () => undefined,
            closeSettings: () => undefined,
            reportError: vi.fn(),
          })
          initialFailure = bootstrap.failure.value
          return () => h('div')
        },
      }),
    )
    expect(initialFailure).toBe('')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(bootstrap.failure.value).toBe('')
    expect(startLayout).toHaveBeenCalledTimes(1)
    view.unmount()
  })

  it('names vault list and tab restore timeouts and stringifies non-Error rejections', async () => {
    vi.useFakeTimers()
    try {
      const { bootstrap: bVault } = setup({
        loadVaults: () => new Promise<void>(() => undefined),
      })
      await vi.advanceTimersByTimeAsync(STEP_LIMIT_MS)
      expect(bVault().failure.value).toBe('The vault list did not answer in 10 s')

      const { bootstrap: bLayout } = setup({
        startLayout: () => new Promise<void>(() => undefined),
      })
      await vi.advanceTimersByTimeAsync(STEP_LIMIT_MS)
      expect(bLayout().failure.value).toBe('The tab restore did not answer in 10 s')
    } finally {
      vi.useRealTimers()
    }

    const { bootstrap: bStr } = setup({
      loadVaults: () => Promise.reject('raw string error'),
    })
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(bStr().failure.value).toBe('raw string error')
  })

  it('ignores late failures or completions after unmount', async () => {
    let rejectVaults!: (err: Error) => void
    const { bootstrap: b1, view: v1 } = setup({
      loadVaults: () =>
        new Promise<void>((_, reject) => {
          rejectVaults = reject
        }),
    })
    v1.unmount()
    rejectVaults(new Error('late error'))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(b1().failure.value).toBe('')

    let resolveLayout!: () => void
    const { calls, view: v2 } = setup({
      startLayout: () =>
        new Promise<void>((resolve) => {
          resolveLayout = resolve
        }),
    })
    await new Promise((resolve) => setTimeout(resolve, 0))
    v2.unmount()
    calls.length = 0
    resolveLayout()
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(calls).not.toContain('startWindow')
  })
})
