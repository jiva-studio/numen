import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, shallowRef } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { formatErrorMessage, THEME_MODE, THEME_ORIGIN } from '@numen/core-client'
import { installCoreHandlers } from '@numen/core-client/testing'
import { VAULT_SUMMARY, type PluginContext } from '@numen/plugin-loader'
import type { VaultRow } from '@numen/workbench'
import App from './App.vue'
import { createWindow } from './window/createWindow'

// jsdom draws no layout, and the palette scrolls the highlighted row into view.
Element.prototype.scrollIntoView = () => undefined

async function* waits(): AsyncGenerator<never> {
  await new Promise<never>(() => undefined)
}

/** What the core is asked, and what it answers, during a test. */
const wire = {
  waits,
  agentSignals: [] as AbortSignal[],
  resolveAddresses: vi.fn(),
  getPlugins: vi.fn(),
  listVaults: vi.fn(),
  tasks: (): AsyncIterable<unknown> => waits(),
  requestedTasks: [] as unknown[],
  taskSignals: [] as AbortSignal[],
  listThemes: vi.fn(),
  readTheme: vi.fn(),
  writeAppearance: vi.fn(),
  themeStream: (): AsyncIterable<{ names: readonly string[] }> => waits(),
  themeSignals: [] as AbortSignal[],
}

/** What the window shows for a call the core rejects with a plain fault. */
const CORE_FAULT = 'internal error'

let restoreCore: () => void

const answerCore = (): (() => void) =>
  installCoreHandlers({
    getAgentState: async (_request: unknown, context: { signal: AbortSignal }) => {
      wire.agentSignals.push(context.signal)
      return { unreachable: '' }
    },
    askAgent: wire.waits,
    finishConversation: async () => ({}),
    resolveAddresses: (request: unknown) => wire.resolveAddresses(request),
    getPlugins: (request: unknown) => wire.getPlugins(request),
    listVaults: (request: unknown) => wire.listVaults(request),
    listThemes: (request: unknown) => wire.listThemes(request),
    readTheme: (request: unknown) => wire.readTheme(request),
    writeAppearance: (request: unknown) => wire.writeAppearance(request),
    watchThemes: (_request: unknown, context: { signal: AbortSignal }) => {
      wire.themeSignals.push(context.signal)
      return wire.themeStream()
    },
    watchTasks: (request: unknown, context: { signal: AbortSignal }) => {
      wire.requestedTasks.push(request)
      wire.taskSignals.push(context.signal)
      return wire.tasks()
    },
  })

const VAULTS: readonly VaultRow[] = [
  { id: 'physics', name: 'Physics', path: '/vaults/Physics' },
  { id: 'words', name: 'Words', path: '/vaults/Words' },
]

const activateCards = (context: {
  register: { tabKind: (one: unknown) => void; view: (one: unknown) => void }
  tabs: { close?: (id: string) => void }
}) => {
  context.register.tabKind({
    kind: 'review',
    component: defineComponent({
      props: { state: { type: Object, required: true } },
      setup: (props) => () => {
        const state = props.state as { vault: string; id: string }
        return h('button', { class: 'review', onClick: () => context.tabs.close?.(state.id) }, [
          `Review of ${state.vault}`,
        ])
      },
    }),
    placement: { pane: 'main', order: 0 },
    open: (vault: string, id: string) => ({ vault, id }),
    onKeyPress: (_: unknown, event: KeyboardEvent) => event.key === 'Escape',
  })
  context.register.view({
    capability: VAULT_SUMMARY,
    component: defineComponent({
      props: { context: { type: Object, required: true } },
      setup: (props) => () =>
        h('i', { class: 'summary' }, `due ${(props.context as { vault: string }).vault}`),
    }),
  })
}

const MANIFEST = {
  id: 'md.numen.cards',
  name: 'Cards',
  version: '1.0.0',
  frontend: {},
  permissions: ['vault:read', 'vault:write'],
  contributes: {
    viewCapabilities: [VAULT_SUMMARY],
    tabKinds: [{ kind: 'review', title: 'Review' }],
  },
}

const serveCore = (
  pluginIds: readonly string[],
  listVaults: () => Promise<readonly VaultRow[]> = async () => VAULTS,
) => {
  wire.getPlugins.mockResolvedValue({ plugins: pluginIds.map((id) => ({ id, name: id })) })
  wire.listVaults.mockImplementation(async () => ({ vaults: await listVaults() }))
}

const createDeps = (listVaults?: () => Promise<readonly VaultRow[]>) => {
  serveCore(['md.numen.cards'], listVaults)
  return {
    fetch: (async () => new Response(JSON.stringify(MANIFEST))) as unknown as typeof fetch,
    dynamicImport: async () => ({ activate: activateCards }),
  }
}

const mounted: { unmount(): void }[] = []

beforeEach(() => {
  wire.getPlugins.mockReset()
  wire.listVaults.mockReset()
  wire.listThemes.mockReset()
  wire.readTheme.mockReset()
  wire.writeAppearance.mockReset()
  wire.resolveAddresses.mockReset()
  wire.resolveAddresses.mockResolvedValue({ found: [], resolved: [] })
  wire.getPlugins.mockResolvedValue({ plugins: [] })
  wire.listVaults.mockResolvedValue({ vaults: [] })
  restoreCore = answerCore()
  wire.listThemes.mockResolvedValue({
    themes: [],
    applied: 'preset:numen',
    mode: THEME_MODE.SYSTEM,
    interfaceScale: 1,
    textScale: 1,
  })
  wire.readTheme.mockResolvedValue({ css: '' })
  wire.writeAppearance.mockResolvedValue({ error: '' })
})

afterEach(() => {
  for (const one of mounted.splice(0)) one.unmount()
  restoreCore()
  wire.tasks = () => wire.waits()
  wire.themeStream = () => wire.waits()
  wire.requestedTasks.length = 0
  wire.taskSignals.length = 0
  wire.themeSignals.length = 0
  wire.agentSignals.length = 0
  vi.useRealTimers()
})

describe('App', () => {
  it('says why it draws nothing when the plugins cannot be listed', async () => {
    wire.getPlugins.mockRejectedValueOnce(new Error('the window is gone'))
    wire.listVaults.mockResolvedValueOnce({ vaults: [] })

    const window = mount(App)
    mounted.push(window)
    await flushPromises()

    expect(window.text()).toContain('No plugin views.')
    expect(window.text()).toContain(CORE_FAULT)
  })

  it('lists the vaults of the installation, as the window is served them', async () => {
    wire.getPlugins.mockResolvedValueOnce({
      plugins: [{ id: 'md.numen.missing', name: 'Missing' }],
    })
    wire.listVaults.mockResolvedValueOnce({
      vaults: [{ id: 'roots', name: 'Roots', path: '/vaults/Roots', isMissing: false }],
    })
    let shell: ReturnType<typeof createWindow> | undefined
    let initialLoading = false
    const wrapper = mount(
      defineComponent({
        setup: () => {
          shell = createWindow({
            fetch: (async () =>
              new Response('not found', { status: 404 })) as unknown as typeof fetch,
          })
          initialLoading = shell.plugins.isLoading.value
          return () => null
        },
      }),
    )
    mounted.push(wrapper)
    expect(initialLoading).toBe(true)
    await flushPromises()

    expect(shell?.plugins.isLoading.value).toBe(false)
    expect(shell?.plugins.failures.value[0]).toContain('md.numen.missing:')
    expect(shell?.vaults.list.value).toStrictEqual([
      { id: 'roots', name: 'Roots', path: '/vaults/Roots' },
    ])

    const keyEvent = new KeyboardEvent('keydown', { key: 'a', cancelable: true })
    globalThis.dispatchEvent(keyEvent)
    expect(keyEvent.defaultPrevented).toBe(false)
    expect(shell?.tab.value).toBeNull()
  })

  it('records failed plugins, resolves links through the note service, and cleans up on unmount', async () => {
    let agentHost: { resolveLinks: (text: string, vault: string) => Promise<unknown> } | undefined
    serveCore(['md.numen.broken', 'md.numen.closing'])
    let shellDefault: ReturnType<typeof createWindow> | undefined
    const defaultWrapper = mount(
      defineComponent({
        setup: () => {
          shellDefault = createWindow({
            fetch: (async (url: string) => {
              if (url.includes('broken')) return new Response('not json', { status: 200 })
              return new Response(JSON.stringify({ ...MANIFEST, id: 'md.numen.closing' }))
            }) as unknown as typeof fetch,
            dynamicImport: async () => ({
              activate: (ctx: {
                work: {
                  agent?: { resolveLinks: (text: string, vault: string) => Promise<unknown> }
                }
                plugin: { subscriptions: { dispose: () => void }[] }
                register: { tabKind: (one: unknown) => void }
              }) => {
                agentHost = ctx.work.agent
                ctx.plugin.subscriptions.push({
                  dispose: () => {
                    throw new Error('close failure')
                  },
                })
                ctx.register.tabKind({
                  kind: 'review',
                  component: {},
                  placement: { pane: 'main', order: 0 },
                })
              },
            }),
          })
          return () => null
        },
      }),
    )
    mounted.push(defaultWrapper)
    await flushPromises()

    expect(shellDefault?.plugins.failures.value[0]).toContain('md.numen.broken:')
    expect(
      document.head.querySelector('link[data-plugin-id="md.numen.closing"]')?.getAttribute('href'),
    ).toBe('/plugin/md.numen.closing/dist/styles.css')
    await agentHost?.resolveLinks('[[Note]]', 'physics')
    expect(wire.resolveAddresses).toHaveBeenCalled()

    defaultWrapper.unmount()
    await flushPromises()
    expect(wire.agentSignals.at(-1)?.aborted).toBe(true)
    expect(shellDefault?.plugins.failures.value).toContain(
      formatErrorMessage(new Error('close failure')),
    )
    const keyAfterUnmount = new KeyboardEvent('keydown', { key: 'a', cancelable: true })
    globalThis.dispatchEvent(keyAfterUnmount)
    expect(keyAfterUnmount.defaultPrevented).toBe(false)
    expect(shellDefault?.tab.value).toBeNull()
  })
})

describe('the window over a plugin that draws the review', () => {
  /** The window drawn over the plugin, as the page draws it. */
  const mountWindow = async (listVaults?: () => Promise<readonly VaultRow[]>) => {
    const window = mount(App, { props: { deps: createDeps(listVaults) }, attachTo: document.body })
    mounted.push(window)
    await flushPromises()
    return { window }
  }

  it('lists the vaults with the summary the plugin draws for each', async () => {
    const { window } = await mountWindow()

    expect(window.findAll('.welcome-page__row--vault')).toHaveLength(2)
    expect(window.findAll('.summary').map((one) => one.text())).toStrictEqual([
      'due physics',
      'due words',
    ])
    expect(window.find('link').attributes('href')).toBe('/plugin/md.numen.cards/dist/styles.css')
  })

  it('opens the review on a vault pressed, and goes back to the vaults when the review closes', async () => {
    const { window } = await mountWindow()

    await window.findAll('.welcome-page__row--vault')[1]!.trigger('click')
    expect(window.find('.review').text()).toBe('Review of words')
    expect(window.find('.welcome-page').exists()).toBe(false)

    await window.find('.review').trigger('click')
    expect(window.find('.review').exists()).toBe(false)
    expect(window.findAll('.welcome-page__row--vault')).toHaveLength(2)
  })

  it('opens the vault of a letter key and passes later keys to the review', async () => {
    const { window } = await mountWindow()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'a', cancelable: true }))
    await flushPromises()
    expect(window.find('.review').text()).toBe('Review of physics')

    const escape = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true })
    globalThis.dispatchEvent(escape)
    expect(escape.defaultPrevented).toBe(true)
  })

  it('says the list is being read until it is', async () => {
    let answer: (vaults: readonly VaultRow[]) => void = vi.fn()
    const { window } = await mountWindow(() => new Promise((then) => (answer = then)))

    expect(window.find('[role="status"]').text()).toBe('Loading vaults…')

    answer(VAULTS)
    await flushPromises()
    expect(window.find('.window__listing').exists()).toBe(false)
    expect(window.findAll('.welcome-page__row--vault')).toHaveLength(2)
  })

  it('lists no vault, records a notice, and finishes listing when the vaults could not be listed', async () => {
    let shell: ReturnType<typeof createWindow> | undefined
    mounted.push(
      mount(
        defineComponent({
          setup: () => {
            shell = createWindow(
              createDeps(async () => {
                throw new Error('the list is locked')
              }),
            )
            return () => null
          },
        }),
      ),
    )
    await flushPromises()

    expect(shell?.vaults.isListing.value).toBe(false)
    expect(shell?.vaults.list.value).toStrictEqual([])
    expect(shell?.notices.notices.value).toMatchObject([
      {
        text: CORE_FAULT,
        severity: 'error',
      },
    ])
  })
})

describe('the work behind the window', () => {
  it("is followed for the window's own target and held as notices, and aborts on unmount", async () => {
    let attempts = 0
    let finishSecond: (() => void) | undefined
    wire.tasks = async function* () {
      attempts += 1
      if (attempts === 1) {
        throw new Error('task stream lost')
      }
      yield {
        tasks: [
          {
            id: 'indexing\t01A',
            title: 'Indexing the vault',
            about: 'Garden',
            error: '',
            isUserStarted: false,
          },
        ],
      }
      await new Promise<void>((resolve) => {
        finishSecond = resolve
      })
    }
    vi.useFakeTimers()
    let shell: ReturnType<typeof createWindow> | undefined
    const wrapper = mount(
      defineComponent({
        setup: () => {
          shell = createWindow(createDeps())
          return () => null
        },
      }),
    )
    mounted.push(wrapper)
    await flushPromises()

    expect(wire.requestedTasks).toMatchObject([{ window: 'desktop.flashcards' }])
    expect(shell?.notices.notices.value).toMatchObject([
      {
        text: 'lost touch with numen — the window keeps trying',
        severity: 'error',
      },
    ])

    await vi.advanceTimersByTimeAsync(1_000)
    await flushPromises()
    expect(shell?.notices.notices.value.map((one) => one.text)).toContain('Indexing the vault')

    const signal = wire.taskSignals.at(-1)
    expect(signal?.aborted).toBe(false)
    wrapper.unmount()
    expect(signal?.aborted).toBe(true)
    finishSecond?.()
    await vi.advanceTimersByTimeAsync(1_000)
    await flushPromises()
    expect(attempts).toBe(2)
  })
})

describe('external links in the window', () => {
  it('intercepts external anchor clicks, reports errors to notices, and stops on unmount', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => {
      throw new Error('blocked popup')
    })
    let shell: ReturnType<typeof createWindow> | undefined
    const wrapper = mount(
      defineComponent({
        setup: () => {
          shell = createWindow(createDeps())
          return () => h('a', { href: 'https://example.test/docs' }, 'Docs')
        },
      }),
      { attachTo: document.body },
    )
    mounted.push(wrapper)
    await flushPromises()

    wrapper.find('a').element.click()
    await vi.waitFor(() => {
      expect(openSpy).toHaveBeenCalledWith('https://example.test/docs', '_blank')
      expect(shell?.notices.notices.value).toMatchObject([
        {
          text: formatErrorMessage(new Error('blocked popup')),
          severity: 'error',
        },
      ])
    })

    openSpy.mockClear()
    const anchor = document.createElement('a')
    anchor.href = 'https://example.test/after'
    document.body.appendChild(anchor)
    wrapper.unmount()
    anchor.click()
    await flushPromises()
    anchor.remove()
    expect(openSpy).not.toHaveBeenCalled()
    openSpy.mockRestore()
  })
})

describe('the appearance palette', () => {
  const openPalette = async () => {
    wire.listThemes.mockResolvedValue({
      themes: [
        {
          name: 'preset:numen',
          title: 'numen',
          origin: THEME_ORIGIN.PRESET,
          pluginId: '',
          isPinned: false,
        },
        {
          name: 'plugin:org.example.themes/night',
          title: 'Night',
          origin: THEME_ORIGIN.PLUGIN,
          pluginId: 'org.example.themes',
          isPinned: true,
        },
      ],
      applied: 'preset:numen',
      mode: THEME_MODE.SYSTEM,
      interfaceScale: 1,
      textScale: 1,
    })
    wire.getPlugins.mockResolvedValueOnce({ plugins: [] })
    wire.getPlugins.mockResolvedValue({
      plugins: [{ id: 'org.example.themes', name: 'Sample Themes' }],
    })
    const window = mount(App, { attachTo: document.body })
    mounted.push(window)
    await flushPromises()
    globalThis.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, cancelable: true }),
    )
    await flushPromises()
    return window
  }

  it('opens on Ctrl+P and offers the themes of a plugin under the name of the plugin', async () => {
    await openPalette()

    expect(document.body.textContent).toContain('Sample Themes')
    expect(document.body.textContent).toContain('Night')
    expect(document.body.textContent).toContain('numen')
  })

  it('writes the theme chosen and applies it to the page', async () => {
    wire.readTheme.mockResolvedValue({ css: ':root { color-scheme: dark; }' })
    await openPalette()
    const rows = [...document.body.querySelectorAll<HTMLElement>('[role="option"]')]
    const night = rows.find((one) => one.textContent.includes('Night'))

    night?.click()
    await flushPromises()

    expect(wire.writeAppearance).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'plugin:org.example.themes/night',
        mode: THEME_MODE.SYSTEM,
      }),
    )
    expect(wire.readTheme).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'plugin:org.example.themes/night' }),
    )
    expect(document.head.querySelector('style[data-appearance="theme"]')?.textContent).toBe(
      ':root { color-scheme: dark; }',
    )
  })

  it('toggles only on unmodified Ctrl+P or Cmd+P, watches theme updates, and handles errors', async () => {
    let streamStep = 0
    let finishSecondThemeStream: (() => void) | undefined
    wire.themeStream = async function* () {
      streamStep += 1
      if (streamStep === 1) {
        yield { names: [] }
        yield { names: ['preset:numen'] }
        yield { names: ['preset:numen', 'plugin:org.example.themes/night'] }
        throw new Error('theme stream lost')
      }
      await new Promise<void>((resolve) => {
        finishSecondThemeStream = resolve
      })
    }
    vi.useFakeTimers()
    let shell: ReturnType<typeof createWindow> | undefined
    const wrapper = mount(
      defineComponent({
        setup: () => {
          shell = createWindow()
          return () => null
        },
      }),
    )
    mounted.push(wrapper)
    await flushPromises()

    expect(shell?.appearance.isOpen.value).toBe(false)
    expect(shell?.appearance.query.value).toBe('')
    expect(wire.listThemes).toHaveBeenCalledTimes(3)
    expect(streamStep).toBe(1)

    await vi.advanceTimersByTimeAsync(1_000)
    await flushPromises()
    expect(streamStep).toBe(2)
    expect(
      shell?.notices.notices.value.some(
        (one) =>
          one.severity === 'error' &&
          one.text === 'lost touch with numen — the window keeps trying',
      ),
    ).toBe(true)

    for (const ignored of [
      new KeyboardEvent('keydown', { key: 'p', cancelable: true }),
      new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, altKey: true, cancelable: true }),
      new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, shiftKey: true, cancelable: true }),
      new KeyboardEvent('keydown', { key: 'k', ctrlKey: true, cancelable: true }),
    ]) {
      globalThis.dispatchEvent(ignored)
      expect(shell?.appearance.isOpen.value).toBe(false)
    }

    const prePrevented = new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, cancelable: true })
    prePrevented.preventDefault()
    globalThis.dispatchEvent(prePrevented)
    expect(shell?.appearance.isOpen.value).toBe(false)

    shell!.appearance.query.value = 'stale'
    wire.getPlugins.mockRejectedValueOnce(new Error('plugins offline'))
    const cmdP = new KeyboardEvent('keydown', { key: 'P', metaKey: true, cancelable: true })
    globalThis.dispatchEvent(cmdP)
    expect(cmdP.defaultPrevented).toBe(true)
    expect(shell?.appearance.isOpen.value).toBe(true)
    expect(shell?.appearance.query.value).toBe('')
    await flushPromises()
    expect(
      shell?.notices.notices.value.some(
        (one) => one.severity === 'error' && one.text === CORE_FAULT,
      ),
    ).toBe(true)

    globalThis.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, cancelable: true }),
    )
    expect(shell?.appearance.isOpen.value).toBe(false)

    globalThis.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, cancelable: true }),
    )
    expect(shell?.appearance.isOpen.value).toBe(true)
    await shell?.appearance.onChoose('other:noop')
    expect(shell?.appearance.isOpen.value).toBe(false)

    const themeSignal = wire.themeSignals.at(-1)
    expect(themeSignal?.aborted).toBe(false)
    wrapper.unmount()
    expect(themeSignal?.aborted).toBe(true)
    finishSecondThemeStream?.()
    await vi.advanceTimersByTimeAsync(1_000)
    await flushPromises()
    expect(streamStep).toBe(2)
    globalThis.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, cancelable: true }),
    )
    expect(shell?.appearance.isOpen.value).toBe(false)
  })
})

const createBarPlugin = (pluginId: string) => ({
  manifest: {
    id: pluginId,
    name: 'Bar',
    version: '1.0.0',
    frontend: {},
    permissions: ['vault:read', 'vault:write', 'vault:manage'],
    contributes: { slots: [{ slot: 'titleBar', view: 'bar' }] },
  },
  activate: (context: PluginContext) => {
    context.register.slotView({
      id: 'bar',
      component: defineComponent({
        setup: () => {
          const vaults = shallowRef(context.vault.vaults!.getState())
          context.vault.vaults!.watch((state) => (vaults.value = state))
          return () =>
            h(
              'header',
              {
                'data-title-bar': pluginId,
                'data-maximized': String(context.ui.window!.getState().isMaximized),
              },
              [
                h('span', { 'data-title': '' }, context.plugin.app.name),
                h('span', { 'data-vault': '' }, vaults.value.current?.name ?? ''),
                h('span', { 'data-vaults': '' }, vaults.value.vaults.map((one) => one.id).join()),
                h('button', {
                  'data-switch': 'words',
                  onClick: () => context.vault.vaults!.switchTo('words'),
                }),
              ],
            )
        },
      }),
    })
  },
})

const createBarDeps = (plugins: readonly ReturnType<typeof createBarPlugin>[]) => {
  const findPlugin = (url: string) => plugins.find((one) => url.includes(`/${one.manifest.id}/`))
  serveCore(['md.numen.cards', ...plugins.map((one) => one.manifest.id)])
  return {
    fetch: (async (url: string) =>
      new Response(
        JSON.stringify(findPlugin(String(url))?.manifest ?? MANIFEST),
      )) as unknown as typeof fetch,
    dynamicImport: async (url: string) => {
      const plugin = findPlugin(url)
      return plugin ? { activate: plugin.activate } : { activate: activateCards }
    },
  }
}

describe('the title bar of the flashcards window', () => {
  it('is not drawn while no plugin draws it, and the window draws no buttons of its own', async () => {
    const window = mount(App, { props: { deps: createDeps() }, attachTo: document.body })
    mounted.push(window)
    await flushPromises()

    expect(window.find('[data-title-bar]').exists()).toBe(false)
    expect(window.find('[data-control]').exists()).toBe(false)
  })

  it('is drawn by the plugin that registers the view of the slot titleBar', async () => {
    const window = mount(App, {
      props: { deps: createBarDeps([createBarPlugin('org.example.ledger')]) },
      attachTo: document.body,
    })
    mounted.push(window)
    await flushPromises()

    expect(window.find('[data-title-bar="org.example.ledger"]').exists()).toBe(true)
    expect(window.find('[data-title]').text()).toBe('numen')
  })

  it('gives the bar the vaults of the window, and the vault of the open review as the current one', async () => {
    const window = mount(App, {
      props: { deps: createBarDeps([createBarPlugin('org.example.ledger')]) },
      attachTo: document.body,
    })
    mounted.push(window)
    await flushPromises()
    expect(window.find('[data-vaults]').text()).toBe('physics,words')
    expect(window.find('[data-vault]').text()).toBe('')

    await window.findAll('.welcome-page__row--vault')[0]!.trigger('click')

    expect(window.find('[data-vault]').text()).toBe('Physics')
  })

  it('opens the review on the vault the bar switches to', async () => {
    const window = mount(App, {
      props: { deps: createBarDeps([createBarPlugin('org.example.ledger')]) },
      attachTo: document.body,
    })
    mounted.push(window)
    await flushPromises()

    await window.find('[data-switch="words"]').trigger('click')
    await flushPromises()

    expect(window.find('.review').text()).toBe('Review of words')
    expect(window.find('[data-vault]').text()).toBe('Words')
  })

  it('gives the bar the state of the window', async () => {
    const window = mount(App, {
      props: { deps: createBarDeps([createBarPlugin('org.example.ledger')]) },
      attachTo: document.body,
    })
    mounted.push(window)
    await flushPromises()

    expect(window.find('[data-title-bar]').attributes('data-maximized')).toBe('false')
  })

  it('is drawn by the plugin first by id, and the other plugin is reported', async () => {
    let shell: ReturnType<typeof createWindow> | undefined
    const deps = createBarDeps([
      createBarPlugin('org.example.ledger'),
      createBarPlugin('org.example.harbor'),
    ])
    mounted.push(
      mount(
        defineComponent({
          setup: () => {
            shell = createWindow(deps)
            return () => h('div')
          },
        }),
      ),
    )
    await flushPromises()

    expect(shell?.plugins.titleBarView.value?.pluginId).toBe('org.example.harbor')
    expect(
      shell?.notices.notices.value.some(
        (one) => one.severity === 'error' && one.text.includes('org.example.ledger'),
      ),
    ).toBe(true)
  })

  it('shows only the loading indicator while the plugins of the window slots start', async () => {
    const deps = createDeps()
    wire.getPlugins.mockReturnValue(new Promise(() => undefined))
    const window = mount(App, { props: { deps }, attachTo: document.body })
    mounted.push(window)
    await flushPromises()

    expect(window.find('[role="status"]').exists()).toBe(true)
    expect(window.find('main').exists()).toBe(false)
  })

  it('draws the window at once when the bar has started, while another plugin still loads', async () => {
    vi.useFakeTimers()
    const deps = createBarDeps([createBarPlugin('org.example.ledger')])
    const window = mount(App, {
      props: {
        deps: {
          ...deps,
          dynamicImport: (url: string) =>
            url.includes('/md.numen.cards/')
              ? new Promise<object>(() => undefined)
              : deps.dynamicImport(url),
        },
      },
      attachTo: document.body,
    })
    mounted.push(window)

    await vi.advanceTimersByTimeAsync(10)

    expect(window.find('main').exists()).toBe(true)
    expect(window.find('[data-title-bar="org.example.ledger"]').exists()).toBe(true)
  })

  it('draws the window at 3 s without a bar that has not started', async () => {
    vi.useFakeTimers()
    const deps = createBarDeps([createBarPlugin('org.example.ledger')])
    const window = mount(App, {
      props: {
        deps: {
          ...deps,
          dynamicImport: (url: string) =>
            url.includes('/org.example.ledger/')
              ? new Promise<object>(() => undefined)
              : deps.dynamicImport(url),
        },
      },
      attachTo: document.body,
    })
    mounted.push(window)

    await vi.advanceTimersByTimeAsync(3_000 - 1)
    expect(window.find('main').exists()).toBe(false)
    await vi.advanceTimersByTimeAsync(1)

    expect(window.find('main').exists()).toBe(true)
    expect(window.find('[data-title-bar]').exists()).toBe(false)
  })

  it('draws a bar that starts after 3 s once it has started', async () => {
    vi.useFakeTimers()
    const deps = createBarDeps([createBarPlugin('org.example.ledger')])
    const window = mount(App, {
      props: {
        deps: {
          ...deps,
          dynamicImport: async (url: string) => {
            if (url.includes('/org.example.ledger/'))
              await new Promise((resolve) => setTimeout(resolve, 3_000 + 2_000))
            return deps.dynamicImport(url)
          },
        },
      },
      attachTo: document.body,
    })
    mounted.push(window)

    await vi.advanceTimersByTimeAsync(3_000)
    expect(window.find('main').exists()).toBe(true)
    expect(window.find('[data-title-bar]').exists()).toBe(false)
    await vi.advanceTimersByTimeAsync(2_010)

    expect(window.find('[data-title-bar="org.example.ledger"]').exists()).toBe(true)
  })
})
