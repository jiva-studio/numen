/**
 * What the corner emits, and what it does not.
 *
 * A card that has been read tells whoever handed it in that they may forget it.
 * Nobody outside can see that happen, so it is asserted here.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Notices from './Notices.vue'
import { SETTLE, type Notice } from './model'

/** A clock a test winds by hand, so nothing waits on the real one. */
function wound(at = 0) {
  const clock = { at }
  return { clock: () => clock.at, wind: (by: number) => (clock.at += by) }
}

const draw = (notices: readonly Notice[], clock: () => number) =>
  mount(Notices, { props: { notices, wait: 0, clock, hidden: () => false } })

const report: Notice = { id: 'renamed', says: 'Renamed', stay: 'read', asked: true }
const refusal: Notice = { id: 'occupied', says: 'Filed there', stay: 'kept', asked: true }
const work: Notice = { id: 'embedding', says: 'Indexing', working: true }

describe('a card the person is finished with', () => {
  it('is named once it has been read long enough', async () => {
    const { clock, wind } = wound()
    const corner = draw([report], clock)

    expect(corner.emitted('gone')).toBeUndefined()

    wind(SETTLE * 10)
    await corner.setProps({ notices: [report] })

    expect(corner.emitted('gone')).toStrictEqual([['renamed']])
  })

  it('is named once when it is put away', async () => {
    const { clock } = wound()
    const corner = draw([report, refusal], clock)

    await corner.findAll('.notice__away')[1]!.trigger('click')

    expect(corner.emitted('gone')).toStrictEqual([['occupied']])
  })

  it('is not named twice when it is put away and then read out', async () => {
    const { clock, wind } = wound()
    const corner = draw([report], clock)

    await corner.find('.notice__away').trigger('click')
    wind(SETTLE * 10)
    await corner.setProps({ notices: [report] })

    expect(corner.emitted('gone')).toStrictEqual([['renamed']])
  })

  it('is never named for work, however long it runs', async () => {
    const { clock, wind } = wound()
    const corner = draw([work], clock)

    wind(SETTLE * 100)
    await corner.setProps({ notices: [work] })

    expect(corner.emitted('gone')).toBeUndefined()
  })

  it('is never named for trouble, which waits for the person', async () => {
    const { clock, wind } = wound()
    const corner = draw([refusal], clock)

    wind(SETTLE * 100)
    await corner.setProps({ notices: [refusal] })

    expect(corner.emitted('gone')).toBeUndefined()
    expect(corner.findAll('article.notice')).toHaveLength(1)
  })
})

describe('a corner nothing is holding any more', () => {
  it('goes on reading once what the pointer was on has gone', async () => {
    // A card is taken out from under the pointer, and a browser owes nothing
    // about the boundary event for one that has gone.
    const { clock, wind } = wound()
    const corner = draw([work, report], clock)

    await corner.find('article.notice').trigger('pointerover')
    await corner.find('.notice__away').trigger('click')
    await corner.setProps({ notices: [report] })

    wind(SETTLE * 10)
    await corner.setProps({ notices: [report] })

    expect(corner.emitted('gone')).toStrictEqual([['embedding'], ['renamed']])
  })
})

describe('the corner as a place', () => {
  it('stands empty with the two regions that read it out', () => {
    const { clock } = wound()
    const corner = draw([], clock)

    expect(corner.findAll('article.notice')).toHaveLength(0)
    expect(corner.find('aside').exists()).toBe(false)
    expect(corner.findAll('[aria-live]')).toHaveLength(2)
  })

  it('brings out what it folded away when the count is pressed', async () => {
    const { clock } = wound()
    const many = Array.from({ length: 5 }, (_, at) => ({
      id: `said-${at}`,
      says: `Renamed ${at}`,
      stay: 'read' as const,
      asked: true,
    }))
    const corner = mount(Notices, {
      props: { notices: many, wait: 0, room: 2, clock, hidden: () => false },
    })

    expect(corner.findAll('article.notice')).toHaveLength(2)

    await corner.find('.notice__folded').trigger('click')

    expect(corner.findAll('article.notice')).toHaveLength(5)
    expect(corner.find('.notice__folded').exists()).toBe(false)
  })
})
