// @vitest-environment jsdom
/**
 * The front door while it is still working, and once it has an answer.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { Waiting } from '@numen/ui'

import Vaults from './Vaults.vue'
import type { Owing } from './core'

const vault = (said: Partial<Owing> = {}): Owing => ({
  vaultId: '01A',
  name: 'Studies',
  path: '/vaults/01A',
  faces: 40,
  due: 8,
  new: 2,
  decks: [],
  presets: [],
  unread: '',
  ...said,
})

const shown = (counting: boolean, vaults: readonly Owing[] = []) =>
  mount(Vaults, { props: { vaults, counting, version: '0.1.0' } })

describe('the front door while the vaults are being counted', () => {
  it('says what it is doing, with the mark that says it is working', () => {
    const one = shown(true)

    expect(one.find('.vaults__counting').text()).toBe('Counting the vaults')
    expect(one.findComponent(Waiting).exists()).toBe(true)
  })

  // A person is told what is happening, and the turning mark is for the eye.
  it('announces it, and draws no row while it has none', () => {
    const one = shown(true)

    expect(one.find('.vaults__counting').attributes('role')).toBe('status')
    expect(one.findAll('.welcome__row--vault')).toHaveLength(0)
  })

  // The list is what the room is for, so the heading names it throughout.
  it('calls the list by its name while it fills', () => {
    expect(shown(true).find('.welcome__heading').text()).toBe('Vaults')
  })

  it('says nothing of the sort once the count is in', () => {
    const one = shown(false, [vault()])

    expect(one.find('.vaults__counting').exists()).toBe(false)
    expect(one.findComponent(Waiting).exists()).toBe(false)
    expect(one.findAll('.welcome__row--vault')).toHaveLength(1)
  })

  // What a vault owes is what the window puts at the end of its row.
  it('puts what each vault owes on its row', () => {
    const one = shown(false, [vault()])

    expect(one.find('.welcome__row--vault').text()).toContain('10')
  })
})
