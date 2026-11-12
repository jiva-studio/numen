// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Session from './Session.vue'
import type { PanelPlace } from '../../model/session/carousel'
import { grades } from '../../lib/card/grades'
import type { ReviewCard } from '../../lib/card/cards'

/** A card as the application provides it. */
const card: ReviewCard = {
  address: { deck: 'decks/Words.md', section: '', cardId: '3f4g5h6j7k', template: 'Say it' },
  heading: 'Leaf mould',
  front: '<p>Leaf mould</p>',
  back: '<p>Compost made of fallen leaves alone. <a href="notes/Leaf mould.md">more</a></p>',
  isNew: false,
  ahead: null,
}

const session = (more: { at?: PanelPlace; isShown?: boolean } = {}) =>
  mount(Session, {
    props: {
      card,
      isShown: more.isShown ?? true,
      left: 3,
      canTakeBack: false,
      at: more.at ?? 'here',
    },
    slots: { panel: '<p>the panel</p>', reading: '<p>the reading</p>' },
  })

const getAskButtons = (one: VueWrapper) =>
  one.findAll('button').filter((it) => it.text().includes('Ask'))

const getReadingButtons = (one: VueWrapper) =>
  one.findAll('button').filter((it) => it.text().includes('Read'))

describe('the way into the panel', () => {
  // The user opens the panel for the current card, turned or not.
  it('is to be pressed on either side of the card', () => {
    expect(getAskButtons(session({ isShown: false }))[0]?.attributes('disabled')).toBeUndefined()
    expect(getAskButtons(session({ isShown: true }))[0]?.attributes('disabled')).toBeUndefined()
  })

  // The two panels stand in the order they stand in the strip.
  it('remains alongside the way into the reading, which is offered the same way', async () => {
    const one = session({ isShown: false })
    expect(getReadingButtons(one)[0]?.attributes('disabled')).toBeUndefined()

    await getReadingButtons(one)[0]?.trigger('click')
    expect(one.emitted('read')).toEqual([['']])
  })

  // The reading is on one side of the card and the conversation on the other,
  // and the strip is scrolled from one to the next in that order.
  it('remains the reading before the card and the conversation after it', () => {
    const one = session()
    expect(one.find('.carousel__before').text()).toContain('the reading')
    expect(one.find('.carousel__after').text()).toContain('the panel')
  })

  // A link inside a card is the other way in, and what it names goes with it.
  it('carries a link pressed in the card out to the window', async () => {
    const one = session({ isShown: true })
    await one.find('.card a').trigger('click')
    expect(one.emitted('read')).toEqual([['notes/Leaf mould.md']])
  })

  it('asks about the card when it is pressed', async () => {
    const one = session({ isShown: true })
    await getAskButtons(one)[0]?.trigger('click')
    expect(one.emitted('ask')).toHaveLength(1)
  })

  // The control both opens and closes the panel: it emits the same request
  // whichever of the three views is displayed, and the window decides.
  it('asks for the panel in the same words while the panel is up', async () => {
    const one = session({ at: 'after' })
    await getAskButtons(one)[0]?.trigger('click')
    expect(one.emitted('ask')).toHaveLength(1)
  })
})

describe('a session with the panel up', () => {
  // The user grades the card while the panel is open.
  it('still draws the four answers, and still says which was pressed', async () => {
    const one = session({ at: 'after' })
    const answers = one.findAll('.session__answer')
    expect(answers).toHaveLength(grades.length)

    await answers[0]?.trigger('click')
    expect(one.emitted('answer')).toEqual([['again']])
  })

  it('draws what was put in the panel', () => {
    expect(session({ at: 'after' }).text()).toContain('the panel')
  })

  // Which of the two panels is displayed is decided by the strip; the session
  // only requests one.
  it('notifies what remains the two alongside each other which of them is wanted', () => {
    const up = session({ at: 'after' })
    expect(up.find('.carousel__after').attributes('inert')).toBeUndefined()
    expect(up.find('.carousel__here').attributes('inert')).toBeDefined()

    const down = session({ at: 'here' })
    expect(down.find('.carousel__after').attributes('inert')).toBeDefined()
    expect(down.find('.carousel__here').attributes('inert')).toBeUndefined()
  })
})

describe('the line that says where the card is from', () => {
  // The review of one deck shows which deck it is and how many cards are left in it.
  it('names the deck of the card and says how many cards are left', () => {
    const one = session()
    expect(one.find('.session__deck').text()).toBe('Words')
    expect(one.find('.session__left').text()).toBe('3 left')
  })
})
