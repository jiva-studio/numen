// @vitest-environment jsdom
/**
 * The list and the vault in front cannot disagree about which vault is open.
 *
 * They were two refs kept in step by hand, and the command that renames a vault
 * wrote one of them. From that moment the welcome screen and the command target
 * named different vaults.
 */
import { ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { useVaults } from './useVaults'
import { WORDS } from '@/shared/words'

const TWO = {
  vaults: [
    { id: 'roots', name: 'Roots', path: '/vaults/Roots', missing: false },
    { id: 'leaves', name: 'Leaves', path: '/vaults/Leaves', missing: false },
  ],
  showing: 'roots',
}

const open = () =>
  useVaults({
    core: { vaults: async () => TWO } as never,
    words: WORDS,
    log: { getWriter: () => () => {} } as never,
    chunks: ref(0),
    embedded: ref(0),
    isEmbedding: ref(false),
  })

describe('the vault in front', () => {
  it('is the one the list says it is showing', async () => {
    const held = open()
    await held.loadVaults()

    expect(held.shown.value).toEqual({ id: 'roots', name: 'Roots' })
    expect(held.listed.value.showing).toBe('roots')
  })

  it('carries a rename without the list falling out of step', async () => {
    const held = open()
    await held.loadVaults()

    held.setVaultName({ id: 'roots', name: 'Rooted' })

    expect(held.shown.value).toEqual({ id: 'roots', name: 'Rooted' })
    // The list carries the new name, and still shows the same vault.
    expect(held.listed.value.vaults.map((one) => one.name)).toEqual(['Rooted', 'Leaves'])
    expect(held.listed.value.showing).toBe('roots')
  })

  it('names nothing while the list shows a vault it does not hold', async () => {
    const held = open()
    await held.loadVaults()

    held.listed.value = { ...held.listed.value, showing: 'gone' }

    expect(held.shown.value).toEqual({ id: '', name: '' })
  })
})
