import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { CONVERSATION } from '@numen/plugin-sdk'
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
