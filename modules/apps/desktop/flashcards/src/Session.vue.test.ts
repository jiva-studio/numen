// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Session from './Session.vue'
import { said } from './core'
import type { Asked } from './core'

/** A card as the application hands it over. */
const card: Asked = {
  deck: 'decks/Words.md',
  section: '',
  card: '3f4g5h6j7k',
  face: 'Say it',
  heading: 'Leaf mould',
  front: '<p>Leaf mould</p>',
  back: '<p>Compost made of fallen leaves alone</p>',
  seen: true,
  ahead: null,
}

/** The sitting on the screen, with something of its own standing in the panel. */
const sitting = (more: { asking?: boolean; offered?: boolean } = {}) =>
  mount(Session, {
    props: {
      card,
      shown: true,
      left: 3,
      takenBack: false,
      asking: more.asking ?? false,
      offered: more.offered ?? false,
    },
    slots: { panel: '<p>the panel</p>' },
  })

/** The way into the panel, wherever it stands. */
const wayIn = (one: VueWrapper) => one.findAll('button').filter((it) => it.text().includes('Ask'))

describe('the way into the panel', () => {
  it('is not drawn on a card that is not offered one', () => {
    expect(wayIn(sitting({ offered: false }))).toHaveLength(0)
  })

  it('is drawn on a card that is offered one', () => {
    expect(wayIn(sitting({ offered: true }))).toHaveLength(1)
  })

  it('asks about the card when it is pressed', async () => {
    const one = sitting({ offered: true })
    await wayIn(one)[0]?.trigger('click')
    expect(one.emitted('ask')).toHaveLength(1)
  })

  // The panel is sent away by the panel and by the keys, and by nothing the
  // sitting draws.
  it('is the only thing the sitting sends the panel away by', async () => {
    const one = sitting({ asking: true, offered: true })
    for (const control of one.findAll('button')) await control.trigger('click')
    expect(one.emitted('shut')).toBeUndefined()
  })
})

describe('a sitting with the panel up', () => {
  // A person grades the card while the panel stands.
  it('still draws the four answers, and still says which was pressed', async () => {
    const one = sitting({ asking: true })
    const answers = one.findAll('.session__answer')
    expect(answers).toHaveLength(said.length)

    await answers[0]?.trigger('click')
    expect(one.emitted('answer')).toEqual([['again']])
  })

  it('draws what was put in the panel', () => {
    expect(sitting({ asking: true }).text()).toContain('the panel')
  })

  it('draws nothing of it while the panel is down', () => {
    const one = sitting({ asking: false })
    expect(one.find('.session__panel').exists()).toBe(false)
    expect(one.text()).not.toContain('the panel')
  })
})
