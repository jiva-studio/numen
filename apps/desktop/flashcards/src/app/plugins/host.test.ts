import { describe, expect, it, vi } from 'vitest'
import { defineComponent, effectScope, h, onScopeDispose } from 'vue'
import { CONVERSATION, VAULT_SUMMARY, createMockAgentHost } from '@numen/plugin-loader'
import { createMockManifest } from '@numen/core-client/testing'
import type { AgentHost, VaultsHost, WindowHost } from '@numen/plugin-loader'
import { TARGET } from '@/shared/lib/window'
import { createNoticeQueue } from '@numen/workbench'
import { createWindowTabs } from '@/entities/tab'
import { createShellHost as createHost } from './host'
import { createWindowCommands } from './windowCommands'

const createShellHost = (
  agent?: AgentHost,
  services: Parameters<typeof createHost>[0]['services'] = {},
  getPlatform?: () => 'windows' | 'darwin',
) => {
  const tabs = createWindowTabs()
  const notices = createNoticeQueue()
  const host = createHost({
    tabs,
    notices,
    services,
    ...(agent ? { agent } : {}),
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

const createMockVaultsHost = (): VaultsHost => ({
  getState: () => ({ vaults: [], current: null }),
  watch: () => NOTHING,
  switchTo: async () => undefined,
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
    permissions: ['vault:read', 'vault:write', 'network:outbound'],
    viewCapabilities: [CONVERSATION, VAULT_SUMMARY],
    tabKinds: [
      'unplaced',
      'second',
      'default-zero',
      'negative',
      'review',
      'fallback',
      'stateless',
    ],
    readers: ['document'],
    commands: ['open'],
    settingsViews: ['floor'],
    capabilities: ['ai.agent.provider'],
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
    const { host } = createShellHost(undefined, {}, () => platform)
    const context = host.createContext('md.numen.example', withVault('md.numen.example'))
    vi.unstubAllGlobals()

    expect(context.plugin.app.platform).toBe('windows')
    platform = 'darwin'
    expect(context.plugin.app.platform).toBe('darwin')
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
      host.createContext('md.numen.example', withVault('md.numen.example')).plugin.target,
    ).toBe(TARGET)
  })

  it('draws the main-pane view of the lowest order, and none that has no placement', () => {
    const { host, front } = createShellHost()
    const context = host.createContext('md.numen.example', withVault('md.numen.example'))

    context.register.tabKind(view('unplaced'))
    expect(front.value).toBeNull()

    context.register.tabKind(view('second', 2))
    const defaultOrder = context.register.tabKind({
      kind: 'default-zero',
      component: {},
      placement: { pane: 'main' } as never,
    })
    expect(front.value?.registration.kind).toBe('default-zero')

    const sub = context.register.tabKind(view('negative', -1))
    expect(front.value?.registration.kind).toBe('negative')
    expect(front.value?.pluginId).toBe('md.numen.example')

    sub.dispose()
    expect(front.value?.registration.kind).toBe('default-zero')

    defaultOrder.dispose()
    expect(front.value?.registration.kind).toBe('second')
  })

  it('releases what a plugin registered once it is disposed, whatever it does to its subscriptions', () => {
    const { host, front } = createShellHost()
    const ctx = host.createContext('md.numen.example', withVault('md.numen.example'))
    ctx.register.tabKind(view('review', 0))
    host
      .createContext('md.numen.other', withVault('md.numen.other'))
      .register.tabKind(view('fallback', 1))
    expect(() => {
      ;(ctx.plugin.subscriptions as { length: number }).length = 0
    }).toThrowError()
    expect((ctx.plugin.subscriptions as unknown as { splice?: unknown }).splice).toBeUndefined()

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

    const bar = host.createContext('org.example.ledger', withVault('org.example.ledger'))
    const cards = host.createContext('md.numen.cards', withVault('md.numen.cards'))

    expect(bar.vault.vaults?.getState()).toStrictEqual(services.vaults.getState())
    expect(cards.vault.vaults?.getState()).toStrictEqual(services.vaults.getState())
    expect(bar.ui.window?.getState()).toStrictEqual(services.window.getState())
    expect(cards.ui.commands?.getIds()).toStrictEqual(services.commands.getIds())
  })

  it('refuses the vaults to a plugin without vault:read and switching to one without vault:manage', async () => {
    const vaults = createMockVaultsHost()
    const switchTo = vi.spyOn(vaults, 'switchTo')
    const { host } = createShellHost(undefined, { vaults })
    const noGrants = host.createContext(
      'org.example.bare',
      createMockManifest('org.example.bare', { name: 'Bare' }),
    )
    const reader = host.createContext('org.example.reader', withVault('org.example.reader'))

    expect(() => noGrants.vault.vaults?.getState()).toThrowError('vault:read')
    await expect(reader.vault.vaults?.switchTo('roots')).rejects.toThrowError('vault:manage')
    expect(switchTo).not.toHaveBeenCalled()
  })

  it('gives a plugin no vaults and no window where the window offers none', () => {
    const context = createShellHost().host.createContext(
      'org.example.ledger',
      withVault('org.example.ledger'),
    )

    expect(context.vault.vaults).toBeUndefined()
    expect(context.ui.window).toBeUndefined()
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
    context.register.slotView({ id: 'bar', component: {} })
    expect(() => {
      ;(context.plugin.subscriptions as { length: number }).length = 0
    }).toThrowError()
    expect(titleBarView.value?.pluginId).toBe('org.example.ledger')

    host.disposePlugin('org.example.ledger')

    expect(titleBarView.value).toBeNull()
  })

  it('disposes all plugins and does not re-run disposed subscriptions on disposeAll', () => {
    const { host, front } = createShellHost()
    const calls: string[] = []
    const ctx1 = host.createContext('md.numen.one', withVault('md.numen.one'))
    ctx1.plugin.subscriptions.push({ dispose: () => calls.push('one') })
    ctx1.register.tabKind(view('review', 0))

    host.disposePlugin('md.numen.one')
    expect(calls).toStrictEqual(['one'])

    const ctx2 = host.createContext('md.numen.two', withVault('md.numen.two'))
    ctx2.plugin.subscriptions.push({ dispose: () => calls.push('two') })
    ctx2.register.tabKind(view('second', 1))

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

    expect(context.plugin.hasPermission?.('vault:read')).toBe(true)
    expect(context.plugin.hasPermission?.('vault:write')).toBe(false)
    expect(withoutManifest.plugin.hasPermission?.('vault:read')).toBe(false)
    expect(withoutGrants.plugin.hasPermission?.('vault:read')).toBe(false)
  })

  it('provides no-op capabilities and opens external web URLs', async () => {
    const { host } = createShellHost()
    const context = host.createContext('md.numen.example', withVault('md.numen.example'))
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)

    context.register.reader?.({ id: 'document' } as never).dispose()
    context.register.command?.({ id: 'open' } as never).dispose()
    context.register.agentProvider?.({ id: 'helper' } as never).dispose()
    context.register.settingsView?.({ id: 'floor' } as never).dispose()
    expect(context.settings.get?.('key')).toBeUndefined()
    context.settings.onChange?.('key', () => undefined).dispose()
    context.events.onVaultChange?.(() => undefined).dispose()
    context.events.onFileSelect?.(() => undefined).dispose()
    expect(context.events.emitVaultChange?.({} as never)).toBeUndefined()
    expect(context.events.emitFileSelect?.({} as never)).toBeUndefined()

    const externalResult = await context.plugin.openExternal?.('https://example.test')
    expect(externalResult?.ok).toBe(true)
    expect(openSpy).toHaveBeenCalledWith('https://example.test', '_blank')
    openSpy.mockRestore()
  })

  it('draws a view one plugin declares into the element of another, until the declaring one is disposed', () => {
    const { host } = createShellHost()
    const conversation = defineComponent({
      props: { context: { type: Object, required: true } },
      setup: (props) => () => h('p', (props.context as { focus: string }).focus),
    })
    host
      .createContext('md.numen.chat', withVault('md.numen.chat'))
      .register.view({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')

    host
      .createContext('md.numen.cards', withVault('md.numen.cards'))
      .ui.mountView(CONVERSATION, panel, {
        agent: createMockAgentHost(),
        focus: 'Decks/Physics.md',
      })
    expect(panel.textContent).toBe('Decks/Physics.md')

    host.disposePlugin('md.numen.chat')
    expect(panel.textContent).toBe('')
  })

  it('takes the views a plugin mounted away with the plugin', () => {
    const { host } = createShellHost()
    const conversation = defineComponent({ setup: () => () => h('p', 'conversation') })
    host
      .createContext('md.numen.chat', withVault('md.numen.chat'))
      .register.view({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')
    host
      .createContext('md.numen.cards', withVault('md.numen.cards'))
      .ui.mountView(CONVERSATION, panel, { agent: createMockAgentHost(), focus: '' })

    host.disposePlugin('md.numen.cards')

    expect(panel.textContent).toBe('')
  })

  it("draws what a plugin shows and the work it does in the window's one list", () => {
    const { host, notices } = createShellHost()
    const context = host.createContext('md.numen.example', withVault('md.numen.example'))

    context.ui.notices.show('Could not save the deck.', 'error')
    context.ui.notices.show('Saved.')
    context.ui.notices.setTasks([{ id: 'index', label: 'Reading the vault', done: 2, total: 5 }])

    expect(notices.notices.value.map((one) => [one.text, one.severity])).toEqual([
      ['Reading the vault', undefined],
      ['Could not save the deck.', 'error'],
      ['Saved.', 'info'],
    ])
    expect(notices.notices.value[0]).toMatchObject({ done: 2, total: 5, isWorking: true })
  })

  it('takes away what a plugin showed once it is disposed', () => {
    const { host, notices } = createShellHost()
    host.createContext('md.numen.example', withVault('md.numen.example')).ui.notices.show('Saved.')
    host.createContext('md.numen.other', withVault('md.numen.other')).ui.notices.show('Drawn.')

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

  it('opens the front kind on the vault chosen, in a detached scope of its own, even without an open callback', () => {
    const { host, tab, openTab, closeTab } = createShellHost()
    const { registration, released } = createKind()
    host.createContext('md.numen.cards', withVault('md.numen.cards')).register.tabKind(registration)

    const parent = effectScope()
    parent.run(() => openTab('roots'))
    parent.stop()

    expect(released).toStrictEqual([])
    expect(tab.value?.vault).toBe('roots')
    expect(tab.value?.view.pluginId).toBe('md.numen.cards')
    expect(tab.value?.state).toStrictEqual({ vault: 'roots', id: 'review#1' })

    closeTab()
    host.createContext('md.numen.stateless', withVault('md.numen.stateless')).register.tabKind({
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
    const context = host.createContext('md.numen.cards', withVault('md.numen.cards'))
    context.register.tabKind(registration)
    openTab('roots')
    const id = tab.value!.id

    context.tabs.close?.('another')
    expect(tab.value?.id).toBe(id)

    context.tabs.close?.(id)
    expect(tab.value).toBeNull()
    expect(released).toStrictEqual([id])
    expect(destroyed).toStrictEqual([id])
  })

  it('releases the open tab before it opens another', () => {
    const { host, tab, openTab } = createShellHost()
    const { registration, released } = createKind()
    host.createContext('md.numen.cards', withVault('md.numen.cards')).register.tabKind(registration)

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
    cardsContext.register.tabKind(createKind().registration)
    host.createContext('md.numen.other', withVault('md.numen.other'))
    openTab('roots')

    host.disposePlugin('md.numen.other')
    expect(tab.value).not.toBeNull()

    host.disposePlugin('md.numen.cards')
    expect(tab.value).toBeNull()

    const afterDispose = vi.fn()
    cardsContext.plugin.subscriptions.push({ dispose: afterDispose })
    host.disposeAll()
    expect(afterDispose).not.toHaveBeenCalled()
  })

  it('opens no tab when the kind fails to open, and releases what it began', () => {
    const { host, tab, openTab } = createShellHost()
    const released: string[] = []
    host.createContext('md.numen.cards', withVault('md.numen.cards')).register.tabKind({
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
    host.createContext('md.numen.cards', withVault('md.numen.cards')).register.view({
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

    expect(
      createShellHost(agent).host.createContext('md.numen.example', withVault('md.numen.example'))
        .work.agent,
    ).toBeDefined()
    expect(
      createShellHost().host.createContext('md.numen.example', withVault('md.numen.example')).work
        .agent,
    ).toBeUndefined()
  })
})

describe('the agent calls of a plugin in the window', () => {
  const ask = (agent: AgentHost) =>
    agent.ask({
      question: 'Quote the note',
      focus: 'Journal/Garden.md',
      conversation: 'c1',
      signal: new AbortController().signal,
    })

  const createContext = (permissions: string[]) => {
    const agent = createMockAgentHost()
    const { host } = createShellHost(agent)
    const context = host.createContext(
      'md.numen.example',
      createMockManifest('md.numen.example', { name: 'Example', permissions }),
    )
    return { agent, agentOfPlugin: context.work.agent! }
  }

  it('refuses the agent to a plugin without vault:write to ask', () => {
    const { agent, agentOfPlugin } = createContext([])
    const spy = vi.spyOn(agent, 'ask')

    expect(() => ask(agentOfPlugin)).toThrow(/vault:write/)
    expect(spy).not.toHaveBeenCalled()
  })

  it('refuses the agent to a plugin without vault:write to finish', async () => {
    const { agent, agentOfPlugin } = createContext([])
    const spy = vi.spyOn(agent, 'finish')

    await expect(agentOfPlugin.finish('c1')).rejects.toThrow(/vault:write/)
    expect(spy).not.toHaveBeenCalled()
  })

  it('refuses the agent to a plugin without vault:read to resolve links', async () => {
    const { agent, agentOfPlugin } = createContext([])
    const spy = vi.spyOn(agent, 'resolveLinks')

    await expect(agentOfPlugin.resolveLinks('[[Garden]]')).rejects.toThrow(/vault:read/)
    expect(spy).not.toHaveBeenCalled()
  })

  it('refuses the agent, which can write the vault, to a plugin with vault:read only to ask', () => {
    const { agent, agentOfPlugin } = createContext(['vault:read'])
    const spy = vi.spyOn(agent, 'ask')

    expect(() => ask(agentOfPlugin)).toThrow(/vault:write/)
    expect(spy).not.toHaveBeenCalled()
  })

  it('refuses the agent to a plugin that has no manifest', () => {
    const { host } = createShellHost(createMockAgentHost())
    const context = host.createContext('md.numen.example')

    expect(() => ask(context.work.agent!)).toThrow(/vault:write/)
  })

  it('lets a plugin with vault:write ask the agent', () => {
    const { agent, agentOfPlugin } = createContext(['vault:write'])
    const spy = vi.spyOn(agent, 'ask')

    ask(agentOfPlugin)

    expect(spy).toHaveBeenCalledOnce()
  })

  it('lets a plugin with vault:read and vault:write finish a conversation and resolve links', async () => {
    const { agent, agentOfPlugin } = createContext(['vault:read', 'vault:write'])
    const finish = vi.spyOn(agent, 'finish')
    const resolveLinks = vi.spyOn(agent, 'resolveLinks')

    await agentOfPlugin.finish('c1')
    await agentOfPlugin.resolveLinks('[[Garden]]')

    expect(finish).toHaveBeenCalledWith('c1')
    expect(resolveLinks).toHaveBeenCalledOnce()
  })
})
