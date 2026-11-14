import { describe, expect, it, vi } from 'vitest'
import { defineComponent, effectScope, h, onScopeDispose } from 'vue'
import { CONVERSATION, VAULT_SUMMARY } from '@numen/plugin-loader'
import { createMockManifest } from '@numen/core-client/testing'
import type { ConnectionHost, VaultsHost, WindowHost } from '@numen/plugin-loader'
import { TARGET } from '@/shared/lib/window'
import { createNoticeQueue } from '@numen/workbench'
import { createWindowCommands } from '@/entities/plugin'
import { createWindowTabs } from '@/entities/tab'
import { createShellHost as createHost } from './host'

/** Connections that end every call at once, for a conversation view that needs them. */
const CONNECTIONS: ConnectionHost = {
  call: async function* () {
    yield* []
    return { ok: true, value: new Uint8Array() }
  },
}

const createShellHost = (
  services: Parameters<typeof createHost>[0]['services'] = {},
  getPlatform?: () => 'windows' | 'darwin',
) => {
  const tabs = createWindowTabs()
  const notices = createNoticeQueue()
  const host = createHost({
    tabs,
    notices,
    services,
    ...(getPlatform ? { getPlatform } : {}),
  })
  return {
    ...host,
    notices,
    front: tabs.front,
    tab: tabs.tab,
    openTab: tabs.openTab,
    closeTab: tabs.closeTab,
  }
}

const NOTHING = { dispose: () => undefined }

const createMockVaultsHost = (switched: string[] = []): VaultsHost => ({
  getState: () => ({ vaults: [], current: null }),
  watch: () => NOTHING,
  switchTo: async (id) => {
    switched.push(id)
  },
})

const createMockWindowHost = (): WindowHost => ({
  getState: () => ({ isMaximized: false, isFullscreen: false }),
  watch: () => NOTHING,
  minimize: () => undefined,
  toggleMaximize: () => undefined,
  close: () => undefined,
})

const withVault = (id: string) =>
  createMockManifest(id, {
    name: id,
    permissions: ['vault:read', 'vault:write', 'link:open'],
    viewCapabilities: [CONVERSATION, VAULT_SUMMARY],
    tabKinds: ['unplaced', 'second', 'default-zero', 'negative', 'review', 'fallback'],
    readers: ['document'],
    commands: ['open'],
    settingsViews: ['floor'],
    capabilities: ['tools'],
  })

const view = (kind: string, order?: number) => ({
  kind,
  component: {},
  ...(order === undefined ? {} : { placement: { pane: 'main' as const, order } }),
})

describe('createShellHost', () => {
  it('gives a plugin the platform of the window, not the one the user agent names', () => {
    vi.stubGlobal('navigator', {
      ...navigator,
      userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
    })
    let platform: 'windows' | 'darwin' = 'windows'
    const { host } = createShellHost({}, () => platform)
    const context = host.createContext('md.numen.example', withVault('md.numen.example'))
    vi.unstubAllGlobals()

    expect(context.plugin.identity.app.platform).toBe('windows')
    platform = 'darwin'
    expect(context.plugin.identity.app.platform).toBe('darwin')
  })

  it('gives a plugin Windows where the window gives no platform, as the frame does without the runtime', () => {
    expect(
      createShellHost().host.createContext('md.numen.example', withVault('md.numen.example')).plugin
        .app.platform,
    ).toBe('windows')
  })

  it('activates a plugin in the window of the flashcards target', () => {
    const { host } = createShellHost()

    expect(
      host.createContext('md.numen.example', withVault('md.numen.example')).plugin.identity.target,
    ).toBe(TARGET)
  })

  it('draws the main-pane view of the lowest order, and none that has no placement', () => {
    const { host, front } = createShellHost()
    const context = host.createContext('md.numen.example', withVault('md.numen.example'))

    context.plugin.register.tabKind(view('unplaced'))
    expect(front.value).toBeNull()

    context.plugin.register.tabKind(view('second', 2))
    const defaultOrder = context.plugin.register.tabKind({
      kind: 'default-zero',
      component: {},
      placement: { pane: 'main' } as never,
    })
    expect(front.value?.kind).toBe('default-zero')

    const sub = context.plugin.register.tabKind(view('negative', -1))
    expect(front.value?.kind).toBe('negative')
    expect(front.value?.pluginId).toBe('md.numen.example')

    sub.dispose()
    expect(front.value?.kind).toBe('default-zero')

    defaultOrder.dispose()
    expect(front.value?.kind).toBe('second')
  })

  it('releases what a plugin registered once it is disposed, whatever it does to its subscriptions', () => {
    const { host, front } = createShellHost()
    const ctx = host.createContext('md.numen.example', withVault('md.numen.example'))
    ctx.plugin.register.tabKind(view('review', 0))
    host
      .createContext('md.numen.other', withVault('md.numen.other'))
      .plugin.register.tabKind(view('fallback', 1))
    expect(() => {
      ;(ctx.plugin.identity.subscriptions as { length: number }).length = 0
    }).toThrowError()
    expect(
      (ctx.plugin.identity.subscriptions as unknown as { splice?: unknown }).splice,
    ).toBeUndefined()

    host.disposePlugin('md.numen.example')
    host.disposePlugin('md.numen.unknown')

    expect(front.value?.kind).toBe('fallback')
  })

  it('gives every plugin the commands, the vaults and the window the window offers', () => {
    const services = {
      commands: createWindowCommands(),
      vaults: createMockVaultsHost(),
      window: createMockWindowHost(),
    }
    const { host } = createShellHost(services)

    const bar = host.createContext('org.example.ledger', withVault('org.example.ledger'))
    const cards = host.createContext('md.numen.cards', withVault('md.numen.cards'))

    expect(bar.host.vault.vaults?.getState()).toStrictEqual(services.vaults.getState())
    expect(cards.host.vault.vaults?.getState()).toStrictEqual(services.vaults.getState())
    expect(bar.host.ui.window?.getState()).toStrictEqual(services.window.getState())
    expect(cards.host.ui.commands?.getIds()).toStrictEqual(services.commands.getIds())
  })

  it('refuses the vaults to a plugin without vault:read and switching to one without vault:manage', async () => {
    const switched: string[] = []
    const vaults = createMockVaultsHost(switched)
    const { host } = createShellHost({ vaults })
    const noGrants = host.createContext(
      'org.example.bare',
      createMockManifest('org.example.bare', { name: 'Bare' }),
    )
    const reader = host.createContext('org.example.reader', withVault('org.example.reader'))

    expect(() => noGrants.host.vault.vaults?.getState()).toThrowError('vault:read')
    await expect(reader.host.vault.vaults?.switchTo('roots')).rejects.toThrowError('vault:manage')
    expect(switched).toStrictEqual([])
  })

  it('gives a plugin no vaults and no window where the window offers none', () => {
    const context = createShellHost().host.createContext(
      'org.example.ledger',
      withVault('org.example.ledger'),
    )

    expect(context.host.vault.vaults).toBeUndefined()
    expect(context.host.ui.window).toBeUndefined()
  })

  it('takes the bar of a plugin away once it is disposed, whatever it does to its subscriptions', () => {
    const { host, titleBarView } = createShellHost()
    const manifest = {
      id: 'org.example.ledger',
      name: 'Ledger bar',
      version: '1.0.0',
      contributes: { slots: [{ slot: 'titleBar', placed: { case: 'view', value: 'bar' } }] },
    } as unknown as Parameters<typeof host.createContext>[1]
    const context = host.createContext('org.example.ledger', manifest)
    context.plugin.register.slotView({ id: 'bar', component: {} })
    expect(() => {
      ;(context.plugin.identity.subscriptions as { length: number }).length = 0
    }).toThrowError()
    expect(titleBarView.value?.pluginId).toBe('org.example.ledger')

    host.disposePlugin('org.example.ledger')

    expect(titleBarView.value).toBeNull()
  })

  it('disposes all plugins and does not re-run disposed subscriptions on disposeAll', () => {
    const { host, front } = createShellHost()
    const calls: string[] = []
    const ctx1 = host.createContext('md.numen.one', withVault('md.numen.one'))
    ctx1.plugin.identity.subscriptions.push({ dispose: () => calls.push('one') })
    ctx1.plugin.register.tabKind(view('review', 0))

    host.disposePlugin('md.numen.one')
    expect(calls).toStrictEqual(['one'])

    const ctx2 = host.createContext('md.numen.two', withVault('md.numen.two'))
    ctx2.plugin.identity.subscriptions.push({ dispose: () => calls.push('two') })
    ctx2.plugin.register.tabKind(view('second', 1))

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
    const withoutGrants = host.createContext(
      'md.numen.empty',
      createMockManifest('md.numen.empty', { name: 'Empty', permissions: [] }),
    )

    expect(context.plugin.identity.hasPermission?.('vault:read')).toBe(true)
    expect(context.plugin.identity.hasPermission?.('vault:write')).toBe(false)
    expect(withoutManifest.plugin.identity.hasPermission?.('vault:read')).toBe(false)
    expect(withoutGrants.plugin.identity.hasPermission?.('vault:read')).toBe(false)
  })

  it('provides no-op capabilities and opens external web URLs', async () => {
    const { host } = createShellHost()
    const context = host.createContext('md.numen.example', withVault('md.numen.example'))
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)

    context.plugin.register.reader?.({ id: 'document', title: 'Document', component: {} }).dispose()
    context.plugin.register.command?.({ id: 'open' } as never).dispose()
    context.plugin.register.settingsView?.({ id: 'floor' } as never).dispose()
    expect(context.plugin.settings.get?.('key')).toBeUndefined()
    context.plugin.settings.onChange?.('key', () => undefined).dispose()
    context.host.events.onVaultChange?.(() => undefined).dispose()
    context.host.events.onFileSelect?.(() => undefined).dispose()
    expect(context.host.events.emitVaultChange?.({} as never)).toBeUndefined()
    expect(context.host.events.emitFileSelect?.({} as never)).toBeUndefined()

    const externalResult = await context.plugin.identity.openExternal?.('https://example.test')
    expect(externalResult?.ok).toBe(true)
    expect(openSpy).toHaveBeenCalledWith('https://example.test', '_blank')
    openSpy.mockRestore()
  })

  it('draws a view one plugin declares into the element of another, until the declaring one is disposed', () => {
    const { host } = createShellHost()
    const conversation = defineComponent({
      props: { context: { type: Object, required: true } },
      setup: (props) => () => h('p', (props.context as { vault: string }).vault),
    })
    host
      .createContext('md.numen.chat', withVault('md.numen.chat'))
      .plugin.register.view({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')

    host
      .createContext('md.numen.cards', withVault('md.numen.cards'))
      .host.ui.mountView(CONVERSATION, panel, { connections: CONNECTIONS, vault: 'physics' })
    expect(panel.textContent).toBe('physics')

    host.disposePlugin('md.numen.chat')
    expect(panel.textContent).toBe('')
  })

  it('takes the views a plugin mounted away with the plugin', () => {
    const { host } = createShellHost()
    const conversation = defineComponent({ setup: () => () => h('p', 'conversation') })
    host
      .createContext('md.numen.chat', withVault('md.numen.chat'))
      .plugin.register.view({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')
    host
      .createContext('md.numen.cards', withVault('md.numen.cards'))
      .host.ui.mountView(CONVERSATION, panel, { connections: CONNECTIONS })

    host.disposePlugin('md.numen.cards')

    expect(panel.textContent).toBe('')
  })

  it("draws what a plugin shows and the work it does in the window's one list", () => {
    const { host, notices } = createShellHost()
    const context = host.createContext('md.numen.example', withVault('md.numen.example'))

    context.host.ui.notices.show('Could not save the deck.', 'error')
    context.host.ui.notices.show('Saved.')
    context.host.ui.notices.setTasks([
      { id: 'index', label: 'Reading the vault', done: 2, total: 5 },
    ])

    expect(notices.notices.value.map((one) => [one.text, one.severity])).toEqual([
      ['Reading the vault', undefined],
      ['Could not save the deck.', 'error'],
      ['Saved.', 'info'],
    ])
    expect(notices.notices.value[0]).toMatchObject({ done: 2, total: 5, isWorking: true })
  })

  it('takes away what a plugin showed once it is disposed', () => {
    const { host, notices } = createShellHost()
    host
      .createContext('md.numen.example', withVault('md.numen.example'))
      .host.ui.notices.show('Saved.')
    host.createContext('md.numen.other', withVault('md.numen.other')).host.ui.notices.show('Drawn.')

    host.disposePlugin('md.numen.example')

    expect(notices.notices.value.map((one) => one.text)).toEqual(['Drawn.'])
  })
})

describe('the tab of the window', () => {
  /** A kind that opens on a vault, keeping what it was opened on and when it was released. */
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

  it('opens the front kind on the vault chosen, in a detached scope of its own', () => {
    const { host, tab, openTab, closeTab } = createShellHost()
    const { registration, released } = createKind()
    host
      .createContext('md.numen.cards', withVault('md.numen.cards'))
      .plugin.register.tabKind(registration)

    const parent = effectScope()
    parent.run(() => openTab('roots'))
    parent.stop()

    expect(released).toStrictEqual([])
    expect(tab.value?.vault).toBe('roots')
    expect(tab.value?.kind.pluginId).toBe('md.numen.cards')
    expect(tab.value?.state).toStrictEqual({ vault: 'roots', id: 'review#1' })

    closeTab()
    expect(tab.value).toBeNull()
  })

  it('closes the tab a plugin names, which takes the window back to its vaults', () => {
    const { host, tab, openTab } = createShellHost()
    const { registration, released, destroyed } = createKind()
    const context = host.createContext('md.numen.cards', withVault('md.numen.cards'))
    context.plugin.register.tabKind(registration)
    openTab('roots')
    const id = tab.value!.id

    context.host.tabs.close?.('another')
    expect(tab.value?.id).toBe(id)

    context.host.tabs.close?.(id)
    expect(tab.value).toBeNull()
    expect(released).toStrictEqual([id])
    expect(destroyed).toStrictEqual([id])
  })

  it('releases the open tab before it opens another', () => {
    const { host, tab, openTab } = createShellHost()
    const { registration, released } = createKind()
    host
      .createContext('md.numen.cards', withVault('md.numen.cards'))
      .plugin.register.tabKind(registration)

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
    const cardsContext = host.createContext('md.numen.cards', withVault('md.numen.cards'))
    cardsContext.plugin.register.tabKind(createKind().registration)
    host.createContext('md.numen.other', withVault('md.numen.other'))
    openTab('roots')

    host.disposePlugin('md.numen.other')
    expect(tab.value).not.toBeNull()

    host.disposePlugin('md.numen.cards')
    expect(tab.value).toBeNull()

    const afterDispose = vi.fn()
    cardsContext.plugin.identity.subscriptions.push({ dispose: afterDispose })
    host.disposeAll()
    expect(afterDispose).not.toHaveBeenCalled()
  })

  it('opens no tab when the kind fails to open, and releases what it began', () => {
    const { host, tab, openTab } = createShellHost()
    const released: string[] = []
    host.createContext('md.numen.cards', withVault('md.numen.cards')).plugin.register.tabKind({
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
    host.createContext('md.numen.cards', withVault('md.numen.cards')).plugin.register.view({
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

describe('the relationships of the window', () => {
  it('refuses a plugin without vault:read the wikilinks of a text', async () => {
    const context = createShellHost().host.createContext(
      'md.numen.example',
      createMockManifest('md.numen.example', { name: 'Example', permissions: [] }),
    )

    await expect(context.host.vault.relations?.resolveLinks('[[Garden]]')).rejects.toThrow(
      /vault:read/,
    )
  })
})
