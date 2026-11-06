import { describe, expect, it, vi } from 'vitest'
import { defineComponent, effectScope, h, onScopeDispose } from 'vue'
import { CONVERSATION, VAULT_SUMMARY, createMockAgentHost } from '@numen/plugin-host'
import { createMockManifest } from '@numen/core-client/testing'
import type { VaultsHost, WindowHost } from '@numen/plugin-host'
import { createShellHost, TARGET } from './host'
import { createWindowCommands } from './windowCommands'

const NOTHING = { dispose: () => undefined }

/** Vaults that list none, standing in for the vaults of the window. */
const createMockVaultsHost = (): VaultsHost => ({
  getState: () => ({ vaults: [], current: null }),
  watch: () => NOTHING,
  switchTo: async () => undefined,
})

/** A window whose buttons do nothing. */
const createMockWindowHost = (): WindowHost => ({
  getState: () => ({ isMaximized: false, isFullscreen: false }),
  watch: () => NOTHING,
  minimize: () => undefined,
  toggleMaximize: () => undefined,
  close: () => undefined,
})

const view = (kind: string, order?: number) => ({
  kind,
  component: {},
  ...(order === undefined ? {} : { placement: { pane: 'main' as const, order } }),
})

describe('createShellHost', () => {
  it('activates a plugin in the window of the flashcards target', () => {
    const { host } = createShellHost()

    expect(host.createContext('md.numen.example').target).toBe(TARGET)
  })

  it('draws the main-pane view of the lowest order, and none that has no placement', () => {
    const { host, front } = createShellHost()
    const context = host.createContext('md.numen.example')

    context.registerTabKind(view('unplaced'))
    expect(front.value).toBeNull()

    context.registerTabKind(view('second', 2))
    const defaultOrder = context.registerTabKind({
      kind: 'default-zero',
      component: {},
      placement: { pane: 'main' } as never,
    })
    expect(front.value?.registration.kind).toBe('default-zero')

    const sub = context.registerTabKind(view('negative', -1))
    expect(front.value?.registration.kind).toBe('negative')
    expect(front.value?.pluginId).toBe('md.numen.example')

    sub.dispose()
    expect(front.value?.registration.kind).toBe('default-zero')

    defaultOrder.dispose()
    expect(front.value?.registration.kind).toBe('second')
  })

  it('lets go of what a plugin registered once it is disposed, even if subscriptions were cleared', () => {
    const { host, front } = createShellHost()
    const ctx = host.createContext('md.numen.example')
    ctx.registerTabKind(view('review', 0))
    host.createContext('md.numen.other').registerTabKind(view('fallback', 1))
    ctx.subscriptions.length = 0

    host.disposePlugin('md.numen.example')
    host.disposePlugin('md.numen.unknown')

    expect(front.value?.registration.kind).toBe('fallback')
  })

  it('gives every plugin the commands, the vaults and the window the window offers', () => {
    const services = {
      commands: createWindowCommands(),
      vaults: createMockVaultsHost(),
      window: createMockWindowHost(),
    }
    const { host } = createShellHost(undefined, services)

    const bar = host.createContext('org.example.ledger')
    const cards = host.createContext('md.numen.cards')

    expect(bar.vaults).toBe(services.vaults)
    expect(cards.vaults).toBe(services.vaults)
    expect(bar.window).toBe(services.window)
    expect(cards.commands).toBe(services.commands)
  })

  it('gives a plugin the vaults and the window the window offers after the host was made', () => {
    const services: Parameters<typeof createShellHost>[1] = {}
    const { host } = createShellHost(undefined, services)
    const vaults = createMockVaultsHost()
    const window = createMockWindowHost()

    Object.assign(services, { vaults, window })

    expect(host.createContext('org.example.ledger').vaults).toBe(vaults)
    expect(host.createContext('org.example.tide').window).toBe(window)
  })

  it('gives a plugin no vaults and no window where the window offers none', () => {
    const context = createShellHost().host.createContext('org.example.ledger')

    expect(context.vaults).toBeUndefined()
    expect(context.window).toBeUndefined()
  })

  it('takes the bar of a plugin away once it is disposed, even if its subscriptions were cleared', () => {
    const { host, titleBarView } = createShellHost()
    const manifest = {
      id: 'org.example.ledger',
      name: 'Ledger bar',
      version: '1.0.0',
      contributes: { slots: [{ slot: 'titleBar', placed: { case: 'view', value: 'bar' } }] },
    } as unknown as Parameters<typeof host.createContext>[1]
    const context = host.createContext('org.example.ledger', manifest)
    context.registerSlotView({ id: 'bar', component: {} })
    context.subscriptions.length = 0
    expect(titleBarView.value?.pluginId).toBe('org.example.ledger')

    host.disposePlugin('org.example.ledger')

    expect(titleBarView.value).toBeNull()
  })

  it('disposes all plugins and does not re-run disposed subscriptions on disposeAll', () => {
    const { host, front } = createShellHost()
    const calls: string[] = []
    const ctx1 = host.createContext('md.numen.one')
    ctx1.subscriptions.push({ dispose: () => calls.push('one') })
    ctx1.registerTabKind(view('review', 0))

    host.disposePlugin('md.numen.one')
    expect(calls).toStrictEqual(['one'])

    const ctx2 = host.createContext('md.numen.two')
    ctx2.subscriptions.push({ dispose: () => calls.push('two') })
    ctx2.registerTabKind(view('second', 1))

    host.disposeAll()
    expect(calls).toStrictEqual(['one', 'two'])
    expect(front.value).toBeNull()
  })

  it('answers a plugin with the permissions its manifest declares', () => {
    const { host } = createShellHost()
    const context = host.createContext(
      'md.numen.example',
      createMockManifest('md.numen.example', { name: 'Example', permissions: ['vault:read'] }),
    )
    const withoutManifest = host.createContext('md.numen.bare')
    const withoutPermissions = host.createContext('md.numen.empty', {
      id: 'md.numen.empty',
      name: 'Empty',
      version: '1.0.0',
    } as never)

    expect(context.hasPermission?.('vault:read')).toBe(true)
    expect(context.hasPermission?.('vault:write')).toBe(false)
    expect(withoutManifest.hasPermission?.('vault:read')).toBe(false)
    expect(withoutPermissions.hasPermission?.('vault:read')).toBe(false)
  })

  it('provides no-op capabilities and opens external web URLs', async () => {
    const { host } = createShellHost()
    const context = host.createContext('md.numen.example')
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)

    context.registerReader?.({} as never).dispose()
    context.registerCommand?.({} as never).dispose()
    context.registerContextMenu?.({} as never).dispose()
    context.registerAgentProvider?.({} as never).dispose()
    context.registerSettingsView?.({} as never).dispose()
    expect(context.getSetting?.('key')).toBeUndefined()
    context.onSettingChange?.('key', () => undefined).dispose()
    context.onVaultChange?.(() => undefined).dispose()
    context.onFileSelect?.(() => undefined).dispose()
    expect(context.emitVaultChange?.({} as never)).toBeUndefined()
    expect(context.emitFileSelect?.({} as never)).toBeUndefined()

    const externalResult = await context.openExternal?.('https://example.test')
    expect(externalResult?.ok).toBe(true)
    expect(openSpy).toHaveBeenCalledWith('https://example.test', '_blank')
    openSpy.mockRestore()
  })

  it('draws a view one plugin declares into the element of another, until the declaring one goes', () => {
    const { host } = createShellHost()
    const conversation = defineComponent({
      props: { context: { type: Object, required: true } },
      setup: (props) => () => h('p', (props.context as { focus: string }).focus),
    })
    host
      .createContext('md.numen.chat')
      .registerView({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')

    host
      .createContext('md.numen.cards')
      .mountView(CONVERSATION, panel, { focus: 'Decks/Physics.md' })
    expect(panel.textContent).toBe('Decks/Physics.md')

    host.disposePlugin('md.numen.chat')
    expect(panel.textContent).toBe('')
  })

  it('takes the views a plugin mounted away with the plugin', () => {
    const { host } = createShellHost()
    const conversation = defineComponent({ setup: () => () => h('p', 'conversation') })
    host
      .createContext('md.numen.chat')
      .registerView({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')
    host.createContext('md.numen.cards').mountView(CONVERSATION, panel, {})

    host.disposePlugin('md.numen.cards')

    expect(panel.textContent).toBe('')
  })

  it("draws what a plugin shows and the work it does in the window's one list", () => {
    const { host, notices } = createShellHost()
    const context = host.createContext('md.numen.example')

    context.notices?.show('Could not save the deck.', 'error')
    context.notices?.show('Saved.')
    context.notices?.setTasks([{ id: 'index', label: 'Reading the vault', done: 2, total: 5 }])

    expect(notices.notices.value.map((one) => [one.text, one.severity])).toEqual([
      ['Reading the vault', undefined],
      ['Could not save the deck.', 'error'],
      ['Saved.', 'info'],
    ])
    expect(notices.notices.value[0]).toMatchObject({ done: 2, total: 5, isWorking: true })
  })

  it('takes away what a plugin showed once it is disposed', () => {
    const { host, notices } = createShellHost()
    host.createContext('md.numen.example').notices?.show('Saved.')
    host.createContext('md.numen.other').notices?.show('Drawn.')

    host.disposePlugin('md.numen.example')

    expect(notices.notices.value.map((one) => one.text)).toEqual(['Drawn.'])
  })
})

describe('the tab of the window', () => {
  /** A kind that opens on a vault, keeping what it was opened on and when it was let go. */
  const createKind = (order = 0) => {
    const released: string[] = []
    const destroyed: string[] = []
    const registration = {
      kind: 'review',
      component: {},
      placement: { pane: 'main' as const, order },
      open: (vault: string, id: string) => {
        onScopeDispose(() => released.push(id))
        return { vault, id }
      },
      onDestroy: (_: unknown, id: string) => destroyed.push(id),
      onKeyPress: (_: unknown, event: KeyboardEvent) => event.key === 'Escape',
    }
    return { registration, released, destroyed }
  }

  it('opens nothing while no plugin offers a view', () => {
    const { tab, openTab } = createShellHost()

    openTab('roots')

    expect(tab.value).toBeNull()
  })

  it('opens the front kind on the vault chosen, in a detached scope of its own, even without an open callback', () => {
    const { host, tab, openTab, closeTab } = createShellHost()
    const { registration, released } = createKind()
    host.createContext('md.numen.cards').registerTabKind(registration)

    const parent = effectScope()
    parent.run(() => openTab('roots'))
    parent.stop()

    expect(released).toStrictEqual([])
    expect(tab.value?.vault).toBe('roots')
    expect(tab.value?.view.pluginId).toBe('md.numen.cards')
    expect(tab.value?.state).toStrictEqual({ vault: 'roots', id: 'review#1' })

    closeTab()
    host.createContext('md.numen.stateless').registerTabKind({
      kind: 'stateless',
      component: {},
      placement: { pane: 'main', order: -1 },
    })
    openTab('words')
    expect(tab.value?.id).toBe('stateless#2')
    expect(tab.value?.state).toBeUndefined()
  })

  it('closes the tab a plugin names, which takes the window back to its vaults', () => {
    const { host, tab, openTab } = createShellHost()
    const { registration, released, destroyed } = createKind()
    const context = host.createContext('md.numen.cards')
    context.registerTabKind(registration)
    openTab('roots')
    const id = tab.value!.id

    context.closeTab?.('another')
    expect(tab.value?.id).toBe(id)

    context.closeTab?.(id)
    expect(tab.value).toBeNull()
    expect(released).toStrictEqual([id])
    expect(destroyed).toStrictEqual([id])
  })

  it('lets go of the open tab before it opens another', () => {
    const { host, tab, openTab } = createShellHost()
    const { registration, released } = createKind()
    host.createContext('md.numen.cards').registerTabKind(registration)

    openTab('roots')
    const first = tab.value!.id
    openTab('words')

    expect(first).toBe('review#1')
    expect(released).toStrictEqual([first])
    expect(tab.value?.vault).toBe('words')
    expect(tab.value?.id).toBe('review#2')
  })

  it('closes the tab of a plugin that is disposed, and of no other', () => {
    const { host, tab, openTab } = createShellHost()
    const cardsContext = host.createContext('md.numen.cards')
    cardsContext.registerTabKind(createKind().registration)
    host.createContext('md.numen.other')
    openTab('roots')

    host.disposePlugin('md.numen.other')
    expect(tab.value).not.toBeNull()

    host.disposePlugin('md.numen.cards')
    expect(tab.value).toBeNull()

    const afterDispose = vi.fn()
    cardsContext.subscriptions.push({ dispose: afterDispose })
    host.disposeAll()
    expect(afterDispose).not.toHaveBeenCalled()
  })

  it('opens no tab when the kind fails to open, and lets go of what it began', () => {
    const { host, tab, openTab } = createShellHost()
    const released: string[] = []
    host.createContext('md.numen.cards').registerTabKind({
      kind: 'review',
      component: {},
      placement: { pane: 'main', order: 0 },
      open: () => {
        onScopeDispose(() => released.push('scope'))
        throw new Error('the vault is gone')
      },
    })

    expect(() => openTab('roots')).toThrow('the vault is gone')
    expect(tab.value).toBeNull()
    expect(released).toStrictEqual(['scope'])
  })

  it("draws a view a plugin declares into the window's own element", () => {
    const { host, mountView } = createShellHost()
    host.createContext('md.numen.cards').registerView({
      capability: VAULT_SUMMARY,
      component: defineComponent({
        props: { context: { type: Object, required: true } },
        setup: (props) => () => h('b', (props.context as { vault: string }).vault),
      }),
    })
    const row = document.createElement('span')

    const view = mountView(VAULT_SUMMARY, row, { vault: 'roots' })
    expect(row.textContent).toBe('roots')

    view.dispose()
    expect(row.textContent).toBe('')
  })
})

describe('the agent of the window', () => {
  it('is given to every plugin where the window has one', () => {
    const agent = createMockAgentHost()

    expect(createShellHost(agent).host.createContext('md.numen.example').agent).toBe(agent)
    expect(createShellHost().host.createContext('md.numen.example').agent).toBeUndefined()
  })
})
