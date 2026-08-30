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
import type { Held } from './welcome'

const vault = (id: string): Held => ({ id, name: id, path: `/vaults/${id}` })

const draw = (vaults: readonly Held[]) => mount(Welcome, { props: { vaults, heading: 'Vaults' } })

/** The letters drawn on the list, in the order the rows stand. */
const caps = (screen: ReturnType<typeof draw>): readonly string[] =>
  screen.findAllComponents(KeyCap).map((cap) => cap.props('keys').letter)

describe('a vault on the list', () => {
  it('carries the letter it is opened by, from the top down', () => {
    expect(caps(draw([vault('physics'), vault('heat')]))).toStrictEqual(['a', 'b'])
  })

  it('carries a mark in place of a letter past the alphabet', () => {
    const many = [...VAULT_LETTERS].map((letter) => vault(letter))

    const screen = draw([...many, vault('last')])

    expect(caps(screen)).toHaveLength(VAULT_LETTERS.length)
    expect(screen.findAll('.welcome__icon')).toHaveLength(1)
  })

  it('is still opened by the hand, by the identity it was given', async () => {
    const screen = draw([vault('physics'), vault('heat')])

    await screen.findAll('.welcome__row--vault')[1]!.trigger('click')

    expect(screen.emitted('opens')).toStrictEqual([['heat']])
  })
})
