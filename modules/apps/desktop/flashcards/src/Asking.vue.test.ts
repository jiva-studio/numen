// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
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

/** A panel holding the card, with a reason nothing can be asked where there is one. */
const held = (unreachable = ''): Held => {
  const panel = asking({
    agent,
    unreachable: () => unreachable,
    everyCard: () => false,
    // The words are put up as they arrive, so a test reads them without waiting
    // for a frame.
    paint: (draw) => draw(),
  })
  panel.opens(card, true)
  return panel
}

const shown = (panel: Held, heading = 'Leaf mould') =>
  mount(Asking, { props: { held: panel, heading } })

/** The control a word stands on. */
const named = (one: ReturnType<typeof shown>, word: string) =>
  one.findAll('button').filter((it) => it.text().includes(word))

describe('the panel a card is asked about in', () => {
  it('names the card it was handed', () => {
    expect(shown(held(), 'Leaf mould').find('.asking__card').text()).toBe('Leaf mould')
  })

  // The panel is the sitting's to hold, and closing it is the panel's own.
  it('closes the panel it holds and tells the sitting nothing', async () => {
    const panel = held()
    const one = shown(panel)

    await named(one, words.shut)[0]?.trigger('click')

    expect(panel.up.value).toBe(false)
    expect(one.emitted('shut')).toBeUndefined()
    expect(one.emitted('ask')).toBeUndefined()
  })

  it('stands the reason nothing can be asked where the answers would be', () => {
    const one = shown(held(words.unreachable))
    expect(one.text()).toContain(words.unreachable)
    expect(one.text()).not.toContain(words.nothingSaid)
  })
})
