import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { VAULT_SUMMARY } from '@numen/plugin-sdk'
import { formatErrorMessage } from '@numen/wire'
import type { VaultRow } from '@numen/shell'

const wire = vi.hoisted(() => {
  async function* waits(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  }
  return {
    waits,
    getPlugins: vi.fn(),
    listVaults: vi.fn(),
    tasks: (): AsyncIterable<unknown> => waits(),
    tasksAsked: [] as unknown[],
  }
})
vi.mock('@/shared/clients', () => ({
  agentService: {
    getAgentState: async () => ({ unreachable: '' }),
    askAgent: wire.waits,
    finishConversation: async () => ({}),
  },
  pluginService: { getPlugins: wire.getPlugins },
  vaultsService: { listVaults: wire.listVaults },
  windowService: {
    watchTasks: (request: unknown) => {
      wire.tasksAsked.push(request)
      return wire.tasks()
    },
  },
}))

const { default: App } = await import('./App.vue')
const { useShell } = await import('./useShell')

const VAULTS: readonly VaultRow[] = [
  { id: 'physics', name: 'Physics', path: '/vaults/Physics' },
  { id: 'words', name: 'Words', path: '/vaults/Words' },
]

/** A tab kind that draws the vault it was mounted on, and a summary that draws the vault it is beside. */
const activateCards = (context: {
  registerTabKind: (one: unknown) => void
  registerView: (one: unknown) => void
  closeTab?: (id: string) => void
}) => {
  context.registerTabKind({
    kind: 'review',
    component: defineComponent({
      props: { state: { type: Object, required: true } },
      setup: (props) => () => {
        const state = props.state as { vault: string; id: string }
        return h('button', { class: 'review', onClick: () => context.closeTab?.(state.id) }, [
          `Review of ${state.vault}`,
        ])
      },
    }),
    placement: { pane: 'main', order: 0 },
    open: (vault: string, id: string) => ({ vault, id }),
    onKeyPress: (_: unknown, event: KeyboardEvent) => event.key === 'Escape',
  })
  context.registerView({
    capability: VAULT_SUMMARY,
    component: defineComponent({
      props: { context: { type: Object, required: true } },
      setup: (props) => () =>
        h('i', { class: 'summary' }, `owes ${(props.context as { vault: string }).vault}`),
    }),
  })
}

const MANIFEST = {
  id: 'md.numen.cards',
  name: 'Cards',
  version: '1.0.0',
  frontend: { entrypoint: 'dist/index.js', styles: 'dist/styles.css' },
}

/** What the shell is given: one plugin that draws the review, and how the vaults are listed. */
const createDeps = (listVaults: () => Promise<readonly VaultRow[]> = async () => VAULTS) => ({
  listPlugins: async () => ['md.numen.cards'],
  listVaults,
  fetch: (async () => new Response(JSON.stringify(MANIFEST))) as unknown as typeof fetch,
  dynamicImport: async () => ({ activate: activateCards }),
})

const mounted: { unmount(): void }[] = []

afterEach(() => {
  for (const one of mounted.splice(0)) one.unmount()
  wire.tasks = () => wire.waits()
  wire.tasksAsked.length = 0
})

describe('App', () => {
  it('says why it draws nothing when the plugins cannot be listed', async () => {
    wire.getPlugins.mockRejectedValueOnce(new Error('the window is gone'))
    wire.listVaults.mockResolvedValueOnce({ vaults: [] })

    const window = mount(App)
    mounted.push(window)
    await flushPromises()

    expect(window.text()).toContain('No plugin views.')
    expect(window.text()).toContain(formatErrorMessage(new Error('the window is gone')))
  })

  it('lists the vaults of the installation, as the window is served them', async () => {
    wire.getPlugins.mockResolvedValueOnce({ plugins: [] })
    wire.listVaults.mockResolvedValueOnce({
      vaults: [{ id: 'roots', name: 'Roots', path: '/vaults/Roots', isMissing: false }],
    })
    let shell: ReturnType<typeof useShell> | undefined
    mounted.push(
      mount(
        defineComponent({
          setup: () => {
            shell = useShell({ listPlugins: async () => [] })
            return () => null
          },
        }),
      ),
    )
    await flushPromises()

    expect(shell?.vaults.list.value).toStrictEqual([
      { id: 'roots', name: 'Roots', path: '/vaults/Roots' },
    ])
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
      'owes physics',
      'owes words',
    ])
    expect(window.find('link').attributes('href')).toBe(
      '/api/plugins/md.numen.cards/dist/styles.css',
    )
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
    let answer: (vaults: readonly VaultRow[]) => void = () => {}
    const { window } = await mountWindow(() => new Promise((then) => (answer = then)))

    expect(window.find('[role="status"]').text()).toBe('Loading vaults…')

    answer(VAULTS)
    await flushPromises()
    expect(window.find('.window__listing').exists()).toBe(false)
    expect(window.findAll('.welcome-page__row--vault')).toHaveLength(2)
  })

  it('says why the vaults could not be listed', async () => {
    const { window } = await mountWindow(async () => {
      throw new Error('the list is locked')
    })

    expect(window.text()).toContain(formatErrorMessage(new Error('the list is locked')))
  })
})

describe('the work behind the window', () => {
  it("is followed for the window's own target and drawn in its notices", async () => {
    wire.tasks = async function* () {
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
      await new Promise<never>(() => {})
    }
    let shell: ReturnType<typeof useShell> | undefined
    mounted.push(
      mount(
        defineComponent({
          setup: () => {
            shell = useShell(createDeps())
            return () => null
          },
        }),
      ),
    )
    await flushPromises()

    expect(wire.tasksAsked).toStrictEqual([{ window: 'desktop.flashcards' }])
    expect(shell?.notices.notices.value.map((one) => one.text)).toStrictEqual([
      'Indexing the vault',
    ])
  })
})
