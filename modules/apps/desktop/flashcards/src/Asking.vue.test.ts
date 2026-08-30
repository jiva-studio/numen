// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import type { AgentPort } from '@numen/ui'

import Asking from './Asking.vue'
import { asking } from './asking'
import type { Held } from './asking'
import { WORDS as words } from './agent/words'
import type { Asked } from './core'

/** A card as the sitting hands one over. */
const card: Asked = {
  deck: 'decks/Words.md',
  section: '',
  card: '3f4g5h6j7k',
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

/** The panel over one card, with the sitting around it standing in for it. */
const holding = (unreachable = ''): Held =>
  asking({
    agent,
    card: () => card,
    unreachable: () => unreachable,
    says: () => {},
    // The words are put up as they arrive, so a test reads them without waiting
    // for a frame.
    paint: (draw) => draw(),
  })

/** A panel holding the card, with a reason nothing can be asked where there is one. */
const held = (unreachable = ''): Held => {
  const panel = holding(unreachable)
  panel.opens()
  return panel
}

const shown = (panel: Held) => mount(Asking, { props: { held: panel } })

describe('the panel a card is asked about in', () => {
  // The card it is about is the card the sitting is on, and the sitting says
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
    const one = mount(Asking, { props: { held: panel }, attachTo: document.body })

    panel.opens()
    await nextTick()
    await nextTick()

    expect(document.activeElement).toBe(one.find('textarea').element)
  })
})
