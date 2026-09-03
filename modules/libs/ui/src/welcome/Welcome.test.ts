/**
 * What the screen puts in front of a vault on the list.
 *
 * The letter is the whole of how a vault is opened from the keyboard, so what
 * is drawn on the row is what the window presses, and the list is asked here
 * both inside the alphabet and past it.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import KeyCap from '../palette/KeyCap.vue'
import Welcome from './Welcome.vue'
import { VAULT_LETTERS } from './picking'
import type { Held, Offer, Way } from './welcome'

const vault = (id: string): Held => ({ id, name: id, path: `/vaults/${id}` })

/** What a window offers below the list, with the keystroke that reaches it. */
const offer: Offer = {
  text: 'New vault',
  detail: 'Choose a folder',
  keys: { marks: ['control', 'shift'], letter: 'N' },
}

const draw = (vaults: readonly Held[]) => mount(Welcome, { props: { vaults, heading: 'Vaults' } })

/** The letters drawn on the list, in the order the rows stand. */
const caps = (screen: ReturnType<typeof draw>): readonly string[] =>
  screen.findAllComponents(KeyCap).map((cap) => cap.props('keys').letter)

describe('a vault on the list', () => {
  it('carries the letter it is opened by, from the top down', () => {
    expect(caps(draw([vault('physics'), vault('heat')]))).toStrictEqual(['A', 'B'])
  })

  it('carries no letter at all past the alphabet', () => {
    const many = [...VAULT_LETTERS].map((letter) => vault(letter))

    const screen = draw([...many, vault('last')])

    expect(caps(screen)).toHaveLength(VAULT_LETTERS.length)
    expect(screen.findAll('.welcome__row--vault')).toHaveLength(VAULT_LETTERS.length + 1)
  })

  it('carries what the window says at the end of its row, and its letter after it', () => {
    const screen = mount(Welcome, {
      props: { vaults: [vault('physics')], heading: 'Vaults' },
      slots: { vault: '<span class="owed">2</span>' },
    })

    expect(screen.find('.welcome__row--vault').text()).toContain('2')
    expect(caps(screen)).toStrictEqual(['A'])
  })

  it('is still opened by the hand, by the identity it was given', async () => {
    const screen = draw([vault('physics'), vault('heat')])

    await screen.findAll('.welcome__row--vault')[1]!.trigger('click')

    expect(screen.emitted('opens')).toStrictEqual([['heat']])
  })
})

describe('a vault the window has not answered for yet', () => {
  const waiting = (id: string): Held => ({ ...vault(id), waiting: true })

  it('stands on the list under its own name', () => {
    const screen = draw([waiting('physics')])

    expect(screen.find('.welcome__row--vault').text()).toContain('physics')
  })

  it('is not opened by a hand', async () => {
    const screen = draw([waiting('physics'), vault('heat')])

    await screen.findAll('.welcome__row--vault')[0]!.trigger('click')

    expect(screen.emitted('opens')).toBeUndefined()
  })

  // The letter is the whole of how a row is opened from the keyboard, so a row
  // that opens nothing carries none. The rows below it keep theirs.
  it('carries no letter, and moves no letter off the rows beside it', () => {
    expect(caps(draw([waiting('physics'), vault('heat')]))).toStrictEqual(['B'])
  })

  it('is opened once the window has answered for it', async () => {
    const screen = draw([vault('physics')])

    await screen.findAll('.welcome__row--vault')[0]!.trigger('click')

    expect(screen.emitted('opens')).toStrictEqual([['physics']])
  })
})

describe('the room the list will fill', () => {
  const drawEmpty = (vaults: readonly Held[]) =>
    mount(Welcome, {
      props: { vaults, heading: 'Vaults' },
      slots: { waiting: '<p class="counting">Counting</p>' },
    })

  it('holds what the window says while it has no rows to give', () => {
    const screen = drawEmpty([])

    expect(screen.find('.welcome__waiting').text()).toBe('Counting')
    expect(screen.find('.welcome__list').exists()).toBe(false)
  })

  it('holds the rows once the window has them', () => {
    const screen = drawEmpty([vault('physics')])

    expect(screen.find('.welcome__waiting').exists()).toBe(false)
    expect(screen.findAll('.welcome__row--vault')).toHaveLength(1)
  })

  it('stands empty where the window says nothing about it', () => {
    expect(draw([]).find('.welcome__waiting').exists()).toBe(false)
  })
})

describe('what the screen offers below the list', () => {
  const drawWith = (one: Offer) =>
    mount(Welcome, { props: { vaults: [vault('physics')], heading: 'Vaults', offer: one } })

  it('carries the keystroke that reaches it, after the letters on the list', () => {
    expect(caps(drawWith(offer))).toStrictEqual(['A', 'N'])
  })

  it('carries none where the window binds none', () => {
    expect(caps(drawWith({ text: offer.text }))).toStrictEqual(['A'])
  })
})

/**
 * The width the screen is drawn at is the column's own, so what a narrow one
 * gives up is given up in the stylesheet. What is asked here is that everything
 * it gives up is marked to be given up, and that what it says is still said
 * somewhere a person can reach.
 */
describe('the screen drawn narrow', () => {
  const drawWith = (one: Offer) =>
    mount(Welcome, {
      props: {
        vaults: [vault('physics')],
        heading: 'Vaults',
        offer: one,
        ways: [{ id: 'find', text: 'Search', keys: { marks: ['control'], letter: 'K' } }],
      },
    })

  it('marks every keystroke it gives up, on a way in, on a vault and on the offer', () => {
    const screen = drawWith(offer)
    expect(screen.findAllComponents(KeyCap)).toHaveLength(3)
    expect(screen.findAll('.welcome__keys')).toHaveLength(3)
  })

  it('leaves the row itself pressed by hand where its keystroke is given up', async () => {
    const screen = drawWith(offer)

    await screen.get('.welcome__row').trigger('click')

    expect(screen.emitted('runs')).toStrictEqual([['find']])
  })

  it('marks what is said under a name, and keeps the whole path on the row', () => {
    const screen = drawWith(offer)
    const path = screen.get('.welcome__row--vault .welcome__aside')

    expect(path.text()).toBe('/vaults/physics')
    expect(path.attributes('title')).toBe('/vaults/physics')
  })
})

/**
 * Short of the height the one column takes, the ways in and the vaults stand
 * side by side and the emblem over them gives way a step at a time, both of
 * which the stylesheet does. What is asked here is that they are two regions to
 * begin with, so that each of them can be a column; that what a short screen
 * gives up is marked to be given up; and that every row of both is drawn and
 * pressed at every height.
 */
describe('the screen drawn short', () => {
  const WAYS: readonly Way[] = [
    { id: 'find', text: 'Search the vault', keys: { marks: ['control'], letter: 'K' } },
    { id: 'commands', text: 'Show the commands' },
    { id: 'note', text: 'New note' },
    { id: 'plex', text: 'Open plex' },
    { id: 'agent', text: 'New agent' },
    { id: 'settings', text: 'Settings' },
  ]

  const drawBoth = () =>
    mount(Welcome, {
      props: {
        vaults: [vault('physics'), vault('heat'), vault('optics')],
        heading: 'Vaults',
        ways: WAYS,
        offer,
      },
    })

  it('holds the ways in and the vaults in regions of their own', () => {
    const screen = drawBoth()

    expect(screen.get('.welcome__lead').findAll('.welcome__row')).toHaveLength(WAYS.length)
    expect(screen.get('.welcome__vaults').findAll('.welcome__row--vault')).toHaveLength(3)
  })

  it('keeps the mark and the name with the ways in', () => {
    const lead = drawBoth().get('.welcome__lead')

    expect(lead.find('.welcome__mark').exists()).toBe(true)
    expect(lead.find('.welcome__name').exists()).toBe(true)
  })

  it('drops no row from either region', () => {
    const screen = drawBoth()
    const rows = screen.findAll('.welcome__row').map((row) => row.text())

    expect(rows).toHaveLength(WAYS.length + 4)
    expect(rows[0]).toContain('Search the vault')
    expect(rows.at(-1)).toContain('New vault')
  })

  it('presses a row of each region', async () => {
    const screen = drawBoth()

    await screen.get('.welcome__lead').findAll('.welcome__row')[5]!.trigger('click')
    await screen.get('.welcome__vaults').findAll('.welcome__row--vault')[2]!.trigger('click')

    expect(screen.emitted('runs')).toStrictEqual([['settings']])
    expect(screen.emitted('opens')).toStrictEqual([['optics']])
  })
})
