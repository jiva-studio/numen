// @vitest-environment jsdom
/**
 * The front door while it is still working, and once it has an answer.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { Coming, Waiting } from '@numen/ui'

import Vaults from './Vaults.vue'
import type { Owing } from './core'

const vault = (said: Partial<Owing> = {}): Owing => ({
  vaultId: '01A',
  name: 'Studies',
  path: '/vaults/01A',
  counted: true,
  faces: 40,
  due: 8,
  new: 2,
  decks: [],
  presets: [],
  unread: '',
  reading: false,
  ...said,
})

/** A vault on the list whose count has not arrived. */
const uncounted = (said: Partial<Owing> = {}): Owing =>
  vault({ counted: false, faces: 0, due: 0, new: 0, ...said })

const shown = (counting: boolean, vaults: readonly Owing[] = []) =>
  mount(Vaults, { props: { vaults, counting, version: '0.1.0' } })

describe('the front door before it knows which vaults there are', () => {
  it('says what it is doing, with the mark that says it is working', () => {
    const one = shown(true)

    expect(one.find('.vaults__counting').text()).toBe('Reading the vaults')
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
})

describe('the front door while the vaults are being counted', () => {
  // The list is what the window opens on, and counting a vault runs behind it.
  it('draws every vault before any of them has a count', () => {
    const one = shown(true, [uncounted(), uncounted({ vaultId: '01B', name: 'Sanskrit' })])

    expect(one.findAll('.welcome__row--vault')).toHaveLength(2)
    expect(one.find('.vaults__counting').exists()).toBe(false)
    expect(one.text()).toContain('Studies')
    expect(one.text()).toContain('Sanskrit')
  })

  // A figure that has not been worked out is drawn as the shape it will be, in
  // the box it will stand in, so nothing moves when it lands.
  it('holds the room the number will take, and prints no number', () => {
    const one = shown(true, [uncounted()])

    expect(one.findComponent(Coming).exists()).toBe(true)
    expect(one.find('.welcome__row--vault').text()).toBe('Studies/vaults/01A')
  })

  // Nought is a vault with nothing to review, and it is not what a vault
  // nobody has counted says.
  it('is not the same as a vault counted at nothing', () => {
    const counted = shown(false, [vault({ due: 0, new: 0 })])

    expect(counted.findComponent(Coming).exists()).toBe(false)
    expect(counted.find('.welcome__row--vault').text()).toContain('0')
  })

  it('does not open a vault whose count has not arrived', async () => {
    const one = shown(true, [uncounted()])

    const row = one.find('.welcome__row--vault')
    expect(row.attributes('disabled')).toBeDefined()
    await row.trigger('click')
    expect(one.emitted('choose')).toBeUndefined()
  })

  // The letter is what the row is opened by, and a row that opens nothing
  // carries none.
  it('draws no letter on a vault it will not open', () => {
    expect(shown(true, [uncounted()]).find('.cap').exists()).toBe(false)
    expect(shown(false, [vault()]).find('.cap').exists()).toBe(true)
  })

  it('opens a vault as soon as that vault has been counted', async () => {
    const one = shown(true, [vault(), uncounted({ vaultId: '01B', name: 'Sanskrit' })])

    await one.findAll('.welcome__row--vault')[0]?.trigger('click')

    expect(one.emitted('choose')).toStrictEqual([['01A']])
  })
})

describe('the front door once the counts are in', () => {
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

  // A vault that could not be counted says why, and says nothing about cards.
  it('says why a vault could not be counted, and prints no shape for it', () => {
    const one = shown(false, [vault({ unread: 'this folder cannot be read as a vault' })])

    expect(one.find('.welcome__row--vault').text()).toContain('cannot be read')
    expect(one.findComponent(Coming).exists()).toBe(false)
  })

  // A vault the index does not carry is being read into it, which is what its
  // row says while that runs.
  it('says a vault is being read, and prints no shape for it', () => {
    const one = shown(false, [vault({ counted: false, reading: true })])

    expect(one.find('.welcome__row--vault').text()).toContain('Reading the vault')
    expect(one.findComponent(Coming).exists()).toBe(false)
  })
})
