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
import type { Held, Offer } from './welcome'

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
