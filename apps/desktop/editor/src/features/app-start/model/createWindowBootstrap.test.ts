/** How a window starts: the layout it opens with, and the plugins it loads before that. */
import { describe, expect, it, vi } from 'vitest'
import { computed, defineComponent, h } from 'vue'
import { pane } from '@numen/workbench'
import { flushPromises, mount } from '@vue/test-utils'
import type { PluginLoader } from '@numen/plugin-loader'
import { createWindowLayout, useOpenTabs, type AnyTabKind, type MovedTab } from '@/entities/tab'
import { ReportedError } from '@numen/result'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createTestTabs } from '@/testing/tabs'
import { createLayoutStart } from './createLayoutStart'
import { createPluginStart } from './createPluginStart'
import { createWindowBootstrap, type WindowBootstrapDeps } from './createWindowBootstrap'

const closeWindow = vi.fn<() => Promise<void>>(() => Promise.resolve())

const withPlacement = (kind: string): AnyTabKind => ({
  kind,
  open: () => ({ title: kind }),
  pane: {},
  getTitle: (state: { title: string }) => state.title,
  display: { placement: { pane: 'main', order: 0 } },
})

const SAVED = { id: 'note:1', kind: 'note', path: 'Tides.md', title: 'Tides', state: '' }

interface Parts {
  readonly savedLayout?: string
  readonly saved?: boolean
  readonly isReadFailing?: boolean
  readonly vaultId?: string
  readonly detached?: MovedTab | null
}

const writeOpenTabsToCore = vi.fn(() => ({}))

const setup = ({
  savedLayout,
  saved = false,
  isReadFailing = false,
  vaultId = 'sea',
  detached = null,
}: Parts = {}) => {
  const api = createAnsweredApi({
    workspace: {
      readOpenTabs: () => {
        if (isReadFailing) throw new Error('core is busy')
        return {
          tabs: saved ? [SAVED] : [],
          activeId: saved ? SAVED.id : '',
          layout: savedLayout ?? '',
        }
      },
      writeOpenTabs: writeOpenTabsToCore,
    },
  })
  const windowTabs = createTestTabs()
  windowTabs.kinds.registerKinds([
    withPlacement('welcome'),
    { ...withPlacement('note'), display: {} },
  ])
  const layout = createWindowLayout(false)
  const calls: string[] = []
  const pluginLoader = {
    status: {
      startTabPlugins: (kinds?: readonly string[]) => {
        calls.push(`tab plugins:${kinds?.join(',') ?? 'every kind'}`)
        return Promise.resolve()
      },
      waitForInterface: () => Promise.resolve(),
    },
    trackPlugins: (ids: readonly string[]) => {
      calls.push(`track:${ids.join(',')}`)
      return Promise.resolve()
    },
    start: () => {
      calls.push('start')
      return Promise.resolve()
    },
    close: () => Promise.resolve(),
  } as unknown as PluginLoader
  const writeMessage = vi.fn()
  const openDetachedTab = vi.fn()
  const vaults = {
    activeVault: computed(() => ({ id: vaultId, name: 'Sea' })),
    loadVaults: () => Promise.resolve(),
  }
  const deps: WindowBootstrapDeps = {
    vaults,
    settings: {
      start: () => Promise.resolve(),
      close: () => undefined,
      editorTokens: { start: () => () => undefined },
    },
    window: { start: () => Promise.resolve(), close: () => undefined },
    editing: { fileFlush: { start: () => Promise.resolve() }, close: () => undefined },
    tabLinkSession: { start: () => Promise.resolve(), close: () => undefined },
    writeMessage,
    plugins: createPluginStart({
      pluginLoader,
      listPlugins: () => [
        { id: 'md.example.notes', hasFrontend: true },
        { id: 'md.example.server', hasFrontend: false },
      ],
      waitForDeclaredCommands: () => Promise.resolve(),
      writeMessage,
    }),
    layout: createLayoutStart({
      vaults,
      windowTabs,
      pluginLoader,
      writeMessage,
      layout,
      closeWhenEmpty: undefined,
      saved: {
        readDetachedTab: vi.fn(() => Promise.resolve(detached)),
        readOpenTabs: api.tabs.savedTabs.readOpenTabs,
      },
      openDetachedTab,
    }),
  }
  mount(
    defineComponent({
      setup() {
        createWindowBootstrap(deps)
        useOpenTabs({
          windowTabs,
          isWriting: layout.isWritingTabs,
          writeMessage,
          writeOpenTabs: api.tabs.savedTabs.writeOpenTabs,
        })
        return () => h('div')
      },
    }),
  )
  return { openDetachedTab, windowTabs, layout, calls, writeMessage }
}

describe('a window starting', () => {
  it('loads the plugins that declare a frontend, and those of every tab kind where no tab was saved', async () => {
    const { calls, layout } = setup()

    await vi.waitFor(() => expect(layout.isReady.value).toBe(true))

    expect(calls).toStrictEqual(['track:md.example.notes', 'start', 'tab plugins:every kind'])
  })

  it('starts only the plugins of the kinds of the saved tabs before it opens them', async () => {
    const { calls, layout } = setup({ saved: true })

    await vi.waitFor(() => expect(layout.isReady.value).toBe(true))

    expect(calls).toStrictEqual(['track:md.example.notes', 'start', 'tab plugins:note'])
  })

  it('opens the tabs its kinds place where none were saved, and starts writing the tabs', async () => {
    const { windowTabs, layout } = setup()

    await vi.waitFor(() => expect(layout.isReady.value).toBe(true))

    expect(windowTabs.tabs.tabs.value.map((one) => one.title)).toStrictEqual(['welcome'])
    expect(layout.isWritingTabs.value).toBe(true)
  })

  it('opens the tabs the core kept where there are some', async () => {
    const { windowTabs, layout } = setup({ saved: true })

    await vi.waitFor(() => expect(layout.isReady.value).toBe(true))

    expect(windowTabs.tabs.tabs.value.map((one) => one.id)).toStrictEqual(['note:1'])
  })

  it.each([true, false])('restores the saved pane when it contains a tab: %s', async (saved) => {
    const workspace = {
      axis: 'vertical',
      root: pane('restored', saved ? [SAVED.id] : []),
      focus: 'restored',
    }
    const { layout, windowTabs } = setup({ saved, savedLayout: JSON.stringify(workspace) })
    await vi.waitFor(() => expect(layout.isReady.value).toBe(true))
    expect(windowTabs.panes.layout.value).toEqual(workspace)
  })

  it('reports a failed read of the saved tabs and never writes the open tabs to the core', async () => {
    writeOpenTabsToCore.mockClear()
    const { windowTabs, layout, writeMessage } = setup({ isReadFailing: true })

    await vi.waitFor(() => expect(layout.isReady.value).toBe(true))
    const [first] = windowTabs.tabs.tabs.value
    if (first) windowTabs.panes.closeTab(first.id)
    await new Promise((resolve) => setTimeout(resolve, 20))

    expect(writeMessage).toHaveBeenCalledWith(expect.stringContaining('saved tabs'), 'error')
    expect(layout.isWritingTabs.value).toBe(false)
    expect(writeOpenTabsToCore).not.toHaveBeenCalled()
  })

  it('writes the open tabs to the core where the read of the saved tabs succeeds', async () => {
    writeOpenTabsToCore.mockClear()
    const { layout } = setup()

    await vi.waitFor(() => expect(layout.isReady.value).toBe(true))

    await vi.waitFor(() => expect(writeOpenTabsToCore).toHaveBeenCalled())
  })

  it('opens no tab and keeps the layout unready where no vault is open', async () => {
    const { windowTabs, layout, openDetachedTab } = setup({ vaultId: '' })

    await new Promise((resolve) => setTimeout(resolve, 20))

    expect(windowTabs.tabs.tabs.value).toStrictEqual([])
    expect(layout.isReady.value).toBe(false)
    expect(openDetachedTab).not.toHaveBeenCalled()
  })

  it('opens a detached tab when the URL names one and skips saved tabs', async () => {
    window.history.replaceState(null, '', '/?id=move-1')
    try {
      const detached: MovedTab = {
        moveId: 'move-1',
        tab: { id: 'note:9', kind: 'note', path: 'Detached.md', title: 'Detached', state: '' },
      }
      const { openDetachedTab, layout, windowTabs } = setup({ detached, saved: true })

      await vi.waitFor(() => expect(layout.isReady.value).toBe(true))

      expect(openDetachedTab).toHaveBeenCalledExactlyOnceWith(detached)
      expect(windowTabs.tabs.tabs.value).toStrictEqual([])
    } finally {
      window.history.replaceState(null, '', '/')
    }
  })

  it('opens no tab and marks the layout started when the detached tab is missing', async () => {
    window.history.replaceState(null, '', '/?id=move-missing')
    try {
      const { openDetachedTab, layout, windowTabs } = setup({ detached: null, saved: true })

      await vi.waitFor(() => expect(layout.isReady.value).toBe(true))

      expect(openDetachedTab).not.toHaveBeenCalled()
      expect(windowTabs.tabs.tabs.value).toStrictEqual([])
    } finally {
      window.history.replaceState(null, '', '/')
    }
  })

  it('closes a secondary window when its last tab closes and reports close failures', async () => {
    closeWindow.mockClear()
    closeWindow
      .mockRejectedValueOnce(new ReportedError('cannot close'))
      .mockResolvedValue(undefined)
    const windowTabs = createTestTabs()
    windowTabs.kinds.registerKinds([withPlacement('welcome')])
    const layout = createWindowLayout(false)
    const writeMessage = vi.fn()
    const pluginLoader = {
      status: { startTabPlugins: () => Promise.resolve() },
    } as unknown as PluginLoader

    const secondaryLayout = createLayoutStart({
      vaults: { activeVault: computed(() => ({ id: 'sea', name: 'Sea' })) },
      windowTabs,
      pluginLoader,
      writeMessage,
      layout,
      closeWhenEmpty: closeWindow,
      saved: {
        readDetachedTab: () => Promise.resolve(null),
        readOpenTabs: () => Promise.resolve({ tabs: [], activeId: '' }),
      },
      openDetachedTab: vi.fn(),
    })

    expect(closeWindow).not.toHaveBeenCalled()
    await secondaryLayout.start()
    expect(windowTabs.tabs.tabs.value).toHaveLength(1)
    expect(closeWindow).not.toHaveBeenCalled()

    windowTabs.panes.closeTab(windowTabs.tabs.tabs.value[0]?.id ?? '')
    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith('The window did not close: cannot close', 'error'),
    )
    expect(closeWindow).toHaveBeenCalledOnce()
  })

  it('does not close a primary window when its last tab closes', async () => {
    closeWindow.mockClear()
    const windowTabs = createTestTabs()
    windowTabs.kinds.registerKinds([withPlacement('welcome')])
    const layout = createWindowLayout(false)
    const primaryLayout = createLayoutStart({
      vaults: { activeVault: computed(() => ({ id: 'sea', name: 'Sea' })) },
      windowTabs,
      pluginLoader: {
        status: { startTabPlugins: () => Promise.resolve() },
      } as unknown as PluginLoader,
      writeMessage: vi.fn(),
      layout,
      closeWhenEmpty: undefined,
      saved: {
        readDetachedTab: () => Promise.resolve(null),
        readOpenTabs: () => Promise.resolve({ tabs: [], activeId: '' }),
      },
      openDetachedTab: vi.fn(),
    })

    await primaryLayout.start()
    primaryLayout.close()
    await flushPromises()

    expect(closeWindow).not.toHaveBeenCalled()
  })

  it('reports plugin, tab-link, and file-flush errors and closes all subsystems on unmount', async () => {
    const writeMessage = vi.fn()
    const closeOrder: string[] = []
    const deps: WindowBootstrapDeps = {
      vaults: { loadVaults: async () => undefined },
      settings: {
        start: async () => undefined,
        close: () => {
          closeOrder.push('settings')
        },
        editorTokens: { start: () => () => undefined },
      },
      window: {
        start: async () => undefined,
        close: () => {
          closeOrder.push('window')
        },
      },
      editing: {
        fileFlush: { start: () => Promise.reject(new ReportedError('flush broke')) },
        close: () => {
          closeOrder.push('editing')
        },
      },
      tabLinkSession: {
        start: () => Promise.reject(new ReportedError('link broke')),
        close: () => {
          closeOrder.push('tabLinkSession')
        },
      },
      plugins: {
        start: () => Promise.reject(new ReportedError('plugin broke')),
        waitForInterface: async () => undefined,
        close: async () => undefined,
      },
      layout: {
        start: async () => undefined,
        close: () => {
          closeOrder.push('layout')
        },
      },
      writeMessage,
    }
    const view = mount(
      defineComponent({
        setup() {
          createWindowBootstrap(deps)
          return () => h('div')
        },
      }),
    )

    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith('Saving on quit stopped: flush broke', 'error'),
    )
    expect(writeMessage).toHaveBeenCalledWith(
      'The window did not start cleanly: plugin broke',
      'error',
    )
    expect(writeMessage).toHaveBeenCalledWith('The tab links stopped: link broke', 'error')

    view.unmount()
    expect(closeOrder).toStrictEqual(['tabLinkSession', 'window', 'editing', 'layout', 'settings'])
  })
})

describe('the editor tokens of a window starting', () => {
  const createOrderedDeps = (order: string[]): WindowBootstrapDeps => ({
    vaults: {
      loadVaults: async () => {
        order.push('vaults')
      },
    },
    settings: {
      start: async () => {
        await Promise.resolve()
        order.push('settings')
      },
      close: () => undefined,
      editorTokens: {
        start: () => {
          order.push('editor tokens')
          return () => undefined
        },
      },
    },
    window: { start: async () => undefined, close: () => undefined },
    editing: { fileFlush: { start: async () => undefined }, close: () => undefined },
    tabLinkSession: { start: async () => undefined, close: () => undefined },
    plugins: {
      start: async () => {
        order.push('plugins')
      },
      waitForInterface: async () => undefined,
      close: async () => undefined,
    },
    layout: {
      start: async () => {
        order.push('layout')
      },
      close: () => undefined,
    },
    writeMessage: vi.fn(),
  })

  const mountBootstrap = (deps: WindowBootstrapDeps) =>
    mount(
      defineComponent({
        setup() {
          createWindowBootstrap(deps)
          return () => h('div')
        },
      }),
    )

  it('start once the settings are loaded and before the tabs are restored', async () => {
    const order: string[] = []

    mountBootstrap(createOrderedDeps(order))

    await vi.waitFor(() => expect(order).toContain('layout'))
    expect(order.indexOf('editor tokens')).toBeGreaterThan(order.indexOf('settings'))
    expect(order.indexOf('editor tokens')).toBeGreaterThan(-1)
    expect(order.indexOf('editor tokens')).toBeLessThan(order.indexOf('layout'))
  })

  it('start exactly once', async () => {
    const order: string[] = []

    mountBootstrap(createOrderedDeps(order))

    await vi.waitFor(() => expect(order).toContain('layout'))
    expect(order.filter((one) => one === 'editor tokens')).toHaveLength(1)
  })
})
