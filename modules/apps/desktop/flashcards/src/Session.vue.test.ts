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
const sitting = (more: { asking?: boolean; shown?: boolean } = {}) =>
  mount(Session, {
    props: {
      card,
      shown: more.shown ?? true,
      left: 3,
      takenBack: false,
      asking: more.asking ?? false,
    },
    slots: { panel: '<p>the panel</p>' },
  })

/** The way into the panel, wherever it stands. */
const wayIn = (one: VueWrapper) => one.findAll('button').filter((it) => it.text().includes('Ask'))

describe('the way into the panel', () => {
  // A person asks about the card they are on, turned or not.
  it('is to be pressed on either side of the card', () => {
    expect(wayIn(sitting({ shown: false }))[0]?.attributes('disabled')).toBeUndefined()
    expect(wayIn(sitting({ shown: true }))[0]?.attributes('disabled')).toBeUndefined()
  })

  it('asks about the card when it is pressed', async () => {
    const one = sitting({ shown: true })
    await wayIn(one)[0]?.trigger('click')
    expect(one.emitted('ask')).toHaveLength(1)
  })

  // The panel is sent away by the panel and by the keys, and by nothing the
  // sitting draws.
  it('is the only thing the sitting sends the panel away by', async () => {
    const one = sitting({ asking: true })
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

  // Which of the two is in the window is the strip's own; the sitting only says
  // which it wants.
  it('tells what stands the two beside each other which of them is wanted', () => {
    const up = sitting({ asking: true })
    expect(up.find('.beside__other').attributes('inert')).toBeUndefined()
    expect(up.find('.beside__one').attributes('inert')).toBeDefined()

    const down = sitting({ asking: false })
    expect(down.find('.beside__other').attributes('inert')).toBeDefined()
    expect(down.find('.beside__one').attributes('inert')).toBeUndefined()
  })
})
