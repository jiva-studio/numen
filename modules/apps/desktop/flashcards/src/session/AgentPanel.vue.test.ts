// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import type { AgentPort } from '@numen/ui'

import AgentPanel from './AgentPanel.vue'
import { agentPanel } from './panel'
import type { AgentPanelState } from './panel'
import { WORDS as words } from './agent/words'
import type { CardFace } from '../core'

/** A card as the session hands one over. */
const card: CardFace = {
  deck: 'decks/Words.md',
  section: '',
  mark: '3f4g5h6j7k',
  face: 'Say it',
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  seen: true,
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
const holding = (unreachable = ''): AgentPanelState => {
  // What the window is showing is the window's, and the test holds it for it.
  const open = ref(false)
  return agentPanel({
    agent: () => agent,
    card: () => card,
    unreachable: () => unreachable,
    open: () => open.value,
    shows: (it) => {
      open.value = it
    },
    says: () => {},
    // The words are put up as they arrive, so a test reads them without waiting
    // for a frame.
    paint: (draw) => draw(),
  })
}

/** A panel holding the card, with a reason nothing can be asked where there is one. */
const held = (unreachable = ''): AgentPanelState => {
  const panel = holding(unreachable)
  panel.opens()
  return panel
}

const shown = (panel: AgentPanelState) => mount(AgentPanel, { props: { held: panel } })

describe('the panel a card is asked about in', () => {
  // The card it is about is the card the session is on, and the session says
  // which above both of them.
  it('names no card of its own', () => {
    const one = shown(held())
    expect(one.text()).not.toContain(card.heading)
    expect(one.findAll('button').map((it) => it.text())).not.toContain(words.shut)
  })

  it('stands the reason nothing can be asked where the answers would be', () => {
    const one = shown(held(words.unreachable))
    expect(one.text()).toContain(words.unreachable)
    expect(one.text()).not.toContain(words.nothingSaid)
  })

  // A panel opened is a panel opened to write in.
  it('takes the keyboard into the field when it comes up', async () => {
    const panel = holding()
    const one = mount(AgentPanel, { props: { held: panel }, attachTo: document.body })

    panel.opens()
    await nextTick()
    await nextTick()

    expect(document.activeElement).toBe(one.find('textarea').element)
  })
})
