// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { Skeleton } from '@numen/ui'

import VaultSummary from './VaultSummary.vue'
import type { VaultCardsDue } from '../../entities/vault'

const vault = (fields: Partial<VaultCardsDue> = {}): VaultCardsDue => ({
  vault: '01A',
  name: 'Studies',
  path: '/vaults/01A',
  isCounted: true,
  faces: 40,
  due: 8,
  new: 2,
  decks: [],
  presets: [],
  unread: '',
  isIndexing: false,
  ...fields,
})

const mountSummary = (one: VaultCardsDue | null) => mount(VaultSummary, { props: { vault: one } })

describe('the summary of a vault', () => {
  it('prints what the vault owes', () => {
    expect(mountSummary(vault()).text()).toContain('10')
  })

  // A figure that has not been worked out is drawn as the shape it will be, in
  // the box it will stand in, so nothing moves when it lands.
  it('holds the room the number will take, and prints no number, before the count', () => {
    for (const one of [null, vault({ isCounted: false, due: 0, new: 0 })]) {
      const summary = mountSummary(one)

      expect(summary.findComponent(Skeleton).exists()).toBe(true)
      expect(summary.text()).not.toMatch(/\d/)
    }
  })

  // Nought is a vault with nothing to review, and it is not what a vault
  // nobody has counted says.
  it('is not the same as a vault counted at nothing', () => {
    const counted = mountSummary(vault({ due: 0, new: 0 }))

    expect(counted.findComponent(Skeleton).exists()).toBe(false)
    expect(counted.text()).toContain('0')
  })

  // A vault that could not be counted says why, and says nothing about cards.
  it('says why a vault could not be counted, and prints no shape for it', () => {
    const one = mountSummary(vault({ unread: 'this folder cannot be read as a vault' }))

    expect(one.text()).toBe('this folder cannot be read as a vault')
    expect(one.findComponent(Skeleton).exists()).toBe(false)
  })

  it('says a vault is being read, and prints no shape for it', () => {
    const one = mountSummary(vault({ isCounted: false, isIndexing: true }))

    expect(one.text()).toBe('Indexing the vault')
    expect(one.findComponent(Skeleton).exists()).toBe(false)
  })
})
