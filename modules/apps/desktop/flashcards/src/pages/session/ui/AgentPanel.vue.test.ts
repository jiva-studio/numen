// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import type { AgentPort } from '@numen/ui'

import AgentPanel from './AgentPanel.vue'
import { useAgentPanel } from '../model/panel'
import type { AgentPanelState } from '../model/panel'
import { WORDS as words } from '../lib/agentWords'
import type { CardFace } from '@/entities/card'

/** A card as the session hands one over. */
const card: CardFace = {
  deck: 'decks/Words.md',
  section: '',
  mark: '3f4g5h6j7k',
  face: 'Say it',
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  isSeen: true,
  ahead: null,
}

/** An agent that says one thing to whatever it is asked. */
const agent: AgentPort = {
  async *ask() {
    yield { kind: 'said', text: 'Because of the leaves.' }
  },
  async finish() {},
}

/** The panel over one card, with the session around it standing in for it. */
const createPanel = (unreachable = ''): AgentPanelState => {
  // What the window is showing is the window's, and the test holds it for it.
  const open = ref(false)
  return useAgentPanel({
    agent: () => agent,
    card: () => card,
    unreachable: () => unreachable,
    open: () => open.value,
    showPanel: (it) => {
      open.value = it
    },
    showNotice: () => {},
    // The words are put up as they arrive, so a test reads them without waiting
    // for a frame.
    paint: (draw) => draw(),
  })
}

/** A panel holding the card, with a reason nothing can be asked where there is one. */
const createOpenPanel = (unreachable = ''): AgentPanelState => {
  const panel = createPanel(unreachable)
  panel.openPanel()
  return panel
}

const mountPanel = (panel: AgentPanelState) => mount(AgentPanel, { props: { held: panel } })

describe('the panel a card is asked about in', () => {
  // The card it is about is the card the session is on, and the session says
  // which above both of them.
  it('names no card of its own', () => {
    const one = mountPanel(createOpenPanel())
    expect(one.text()).not.toContain(card.heading)
    expect(one.findAll('button').map((it) => it.text())).not.toContain(words.shut)
  })

  it('stands the reason nothing can be asked where the answers would be', () => {
    const one = mountPanel(createOpenPanel(words.unreachable))
    expect(one.text()).toContain(words.unreachable)
    expect(one.text()).not.toContain(words.nothingSaid)
  })

  // A panel opened is a panel opened to write in.
  it('takes the keyboard into the field when it comes up', async () => {
    const panel = createPanel()
    const one = mount(AgentPanel, { props: { held: panel }, attachTo: document.body })

    panel.openPanel()
    await nextTick()
    await nextTick()

    expect(document.activeElement).toBe(one.find('textarea').element)
  })
})
