/** How a window starts: the layout it opens with, and the plugins it loads before that. */
import { describe, expect, it, vi } from 'vitest'
import { computed, defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import type { PluginLoader } from '@numen/plugin-loader'
import { createWindowLayout, useOpenTabs, type AnyTabKind, type MovedTab } from '@/entities/tab'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createTestTabs } from '@/testing/tabs'
import { createLayoutStart } from './createLayoutStart'
import { createPluginStart } from './createPluginStart'
import { createWindowBootstrap, type WindowBootstrapDeps } from './createWindowBootstrap'

const withPlacement = (kind: string): AnyTabKind => ({
  kind,
  open: () => ({ title: kind }),
  pane: {},
  getTitle: (state: { title: string }) => state.title,
  display: { placement: { pane: 'main', order: 0 } },
})

const SAVED = { id: 'note:1', kind: 'note', path: 'Tides.md', title: 'Tides', state: '' }

interface Parts {
  readonly saved?: boolean
  readonly isReadFailing?: boolean
  readonly vaultId?: string
  readonly detached?: MovedTab | null
}

const writeOpenTabsToCore = vi.fn(() => ({}))

const setup = ({
  saved = false,
  isReadFailing = false,
  vaultId = 'sea',
  detached = null,
}: Parts = {}) => {
  const api = createAnsweredApi({
    workspace: {
      readOpenTabs: () => {
        if (isReadFailing) throw new Error('core is busy')
        return { tabs: saved ? [SAVED] : [], activeId: saved ? SAVED.id : '' }
      },
      writeOpenTabs: writeOpenTabsToCore,
    },
  })
  const windowTabs = createTestTabs()
  windowTabs.groups.registerKinds([
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
    settings: { start: () => Promise.resolve(), close: () => undefined },
    window: { start: () => Promise.resolve(), close: () => undefined },
    editing: { fileFlush: { start: () => Promise.resolve() }, close: () => undefined },
    tabLinkSession: { start: () => Promise.resolve(), close: () => undefined },
    writeMessage,
    plugins: createPluginStart({
      pluginLoader,
      getPlugins: () => [
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
      isSecondary: false,
      windowId: 'window-a',
      readDetachedTab: vi.fn(() => Promise.resolve(detached)),
      readOpenTabs: api.savedTabs.readOpenTabs,
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
          writeOpenTabs: api.savedTabs.writeOpenTabs,
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

  it('reports a failed read of the saved tabs and never writes the open tabs to the core', async () => {
    writeOpenTabsToCore.mockClear()
    const { windowTabs, layout, writeMessage } = setup({ isReadFailing: true })

    await vi.waitFor(() => expect(layout.isReady.value).toBe(true))
    const [first] = windowTabs.tabs.tabs.value
    if (first) windowTabs.closeTab(first.id)
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
})
