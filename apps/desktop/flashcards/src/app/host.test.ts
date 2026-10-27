import { describe, expect, it } from 'vitest'
import { defineComponent, h, onScopeDispose } from 'vue'
import { CONVERSATION, createMockAgentHost, VAULT_SUMMARY } from '@numen/plugin-sdk'
import { createShellHost, TARGET } from './host'

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
    context.registerTabKind(view('first', 1))
    expect(front.value?.registration.kind).toBe('first')
    expect(front.value?.pluginId).toBe('md.numen.example')
  })

  it('lets go of what a plugin registered once it is disposed', () => {
    const { host, front } = createShellHost()
    host.createContext('md.numen.example').registerTabKind(view('review', 0))

    host.disposePlugin('md.numen.example')

    expect(front.value).toBeNull()
  })

  it('answers a plugin with the permissions its manifest declares', () => {
    const { host } = createShellHost()
    const context = host.createContext('md.numen.example', {
      id: 'md.numen.example',
      name: 'Example',
      version: '1.0.0',
      permissions: ['vault:read'],
    })

    expect(context.hasPermission?.('vault:read')).toBe(true)
    expect(context.hasPermission?.('vault:write')).toBe(false)
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

    context.notices?.show('Could not save the deck.', 'alarm')
    context.notices?.show('Saved.')
    context.notices?.setTasks([{ id: 'index', label: 'Reading the vault', done: 2, total: 5 }])

    expect(notices.notices.value.map((one) => [one.text, one.tone])).toEqual([
      ['Reading the vault', undefined],
      ['Could not save the deck.', 'alarm'],
      ['Saved.', 'plain'],
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

  it('opens the front kind on the vault chosen, in a scope of its own', () => {
    const { host, tab, openTab } = createShellHost()
    const { registration } = createKind()
    host.createContext('md.numen.cards').registerTabKind(registration)

    openTab('roots')

    expect(tab.value?.vault).toBe('roots')
    expect(tab.value?.view.pluginId).toBe('md.numen.cards')
    expect(tab.value?.state).toStrictEqual({ vault: 'roots', id: tab.value?.id })
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

    expect(released).toStrictEqual([first])
    expect(tab.value?.vault).toBe('words')
    expect(tab.value?.id).not.toBe(first)
  })

  it('closes the tab of a plugin that is disposed, and of no other', () => {
    const { host, tab, openTab } = createShellHost()
    host.createContext('md.numen.cards').registerTabKind(createKind().registration)
    host.createContext('md.numen.other')
    openTab('roots')

    host.disposePlugin('md.numen.other')
    expect(tab.value).not.toBeNull()

    host.disposePlugin('md.numen.cards')
    expect(tab.value).toBeNull()
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
