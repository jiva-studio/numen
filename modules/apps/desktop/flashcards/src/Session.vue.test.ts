// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Session from './Session.vue'
import type { Where } from './Beside.vue'
import { grades } from './core'
import type { CardFace } from './core'

/** A card as the application hands it over. */
const card: CardFace = {
  deck: 'decks/Words.md',
  section: '',
  card: '3f4g5h6j7k',
  face: 'Say it',
  heading: 'Leaf mould',
  front: '<p>Leaf mould</p>',
  back: '<p>Compost made of fallen leaves alone. <a href="notes/Leaf mould.md">more</a></p>',
  seen: true,
  ahead: null,
}

/** The sitting on the screen, with something of its own standing in the panel. */
const sitting = (more: { at?: Where; shown?: boolean } = {}) =>
  mount(Session, {
    props: {
      card,
      shown: more.shown ?? true,
      left: 3,
      takenBack: false,
      at: more.at ?? 'here',
    },
    slots: { panel: '<p>the panel</p>', reading: '<p>the reading</p>' },
  })

/** The way into the panel, wherever it stands. */
const wayIn = (one: VueWrapper) => one.findAll('button').filter((it) => it.text().includes('Ask'))

/** The way into the reading, wherever it stands. */
const wayBack = (one: VueWrapper) => one.findAll('button').filter((it) => it.text().includes('Read'))

describe('the way into the panel', () => {
  // A person asks about the card they are on, turned or not.
  it('is to be pressed on either side of the card', () => {
    expect(wayIn(sitting({ shown: false }))[0]?.attributes('disabled')).toBeUndefined()
    expect(wayIn(sitting({ shown: true }))[0]?.attributes('disabled')).toBeUndefined()
  })

  // The two panels stand in the order they stand in the strip.
  it('stands beside the way into the reading, which is offered the same way', async () => {
    const one = sitting({ shown: false })
    expect(wayBack(one)[0]?.attributes('disabled')).toBeUndefined()

    await wayBack(one)[0]?.trigger('click')
    expect(one.emitted('read')).toEqual([['']])
  })

  // The reading is on one side of the card and the conversation on the other,
  // and the strip is scrolled from one to the next in that order.
  it('stands the reading before the card and the conversation after it', () => {
    const one = sitting()
    expect(one.find('.beside__before').text()).toContain('the reading')
    expect(one.find('.beside__other').text()).toContain('the panel')
  })

  // A link inside a card is the other way in, and what it names goes with it.
  it('carries a link pressed in the card out to the window', async () => {
    const one = sitting({ shown: true })
    await one.find('.card a').trigger('click')
    expect(one.emitted('read')).toEqual([['notes/Leaf mould.md']])
  })

  it('asks about the card when it is pressed', async () => {
    const one = sitting({ shown: true })
    await wayIn(one)[0]?.trigger('click')
    expect(one.emitted('ask')).toHaveLength(1)
  })

  // The control is the way in and the way out both: what it asks for is the
  // same whichever of the three the window is on, and the window decides.
  it('asks for the panel in the same words while the panel is up', async () => {
    const one = sitting({ at: 'after' })
    await wayIn(one)[0]?.trigger('click')
    expect(one.emitted('ask')).toHaveLength(1)
  })
})

describe('a sitting with the panel up', () => {
  // A person grades the card while the panel stands.
  it('still draws the four answers, and still says which was pressed', async () => {
    const one = sitting({ at: 'after' })
    const answers = one.findAll('.session__answer')
    expect(answers).toHaveLength(grades.length)

    await answers[0]?.trigger('click')
    expect(one.emitted('answer')).toEqual([['again']])
  })

  it('draws what was put in the panel', () => {
    expect(sitting({ at: 'after' }).text()).toContain('the panel')
  })

  // Which of the two is in the window is the strip's own; the sitting only says
  // which it wants.
  it('tells what stands the two beside each other which of them is wanted', () => {
    const up = sitting({ at: 'after' })
    expect(up.find('.beside__other').attributes('inert')).toBeUndefined()
    expect(up.find('.beside__one').attributes('inert')).toBeDefined()

    const down = sitting({ at: 'here' })
    expect(down.find('.beside__other').attributes('inert')).toBeDefined()
    expect(down.find('.beside__one').attributes('inert')).toBeUndefined()
  })
})
