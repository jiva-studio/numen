// @vitest-environment jsdom
/**
 * The list and the vault in front cannot disagree about which vault is open.
 *
 * They were two refs kept in step by hand, and the command that renames a vault
 * wrote one of them. From that moment the welcome screen and the command target
 * named different vaults.
 */
import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
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
    log: { getWriter: () => vi.fn() } as never,
    chunks: ref(0),
    embedded: ref(0),
    isEmbedding: ref(false),
  })

describe('the vault in front', () => {
  it('is the one the list says it is showing', async () => {
    const target = open()
    await target.loadVaults()

    expect(target.activeVault.value).toEqual({ id: 'roots', name: 'Roots' })
    expect(target.vaultList.value.showing).toBe('roots')
  })

  it('carries a rename without the list falling out of step', async () => {
    const target = open()
    await target.loadVaults()

    target.setVaultName({ id: 'roots', name: 'Rooted' })

    expect(target.activeVault.value).toEqual({ id: 'roots', name: 'Rooted' })
    // The list carries the new name, and still shows the same vault.
    expect(target.vaultList.value.vaults.map((one) => one.name)).toEqual(['Rooted', 'Leaves'])
    expect(target.vaultList.value.showing).toBe('roots')
  })

  it('is empty while the list shows a vault it does not hold', async () => {
    const target = open()
    await target.loadVaults()

    target.vaultList.value = { ...target.vaultList.value, showing: 'gone' }

    expect(target.activeVault.value).toEqual({ id: '', name: '' })
  })
})

describe('the vault list before and after it is requested for', () => {
  it('is empty and shows nothing before the first answer', () => {
    const target = open()

    expect(target.vaultList.value).toEqual({ vaults: [], showing: '' })
    expect(target.activeVault.value).toEqual({ id: '', name: '' })
  })

  it('reloads the page through the location', () => {
    const reload = vi.fn()
    vi.stubGlobal('location', { reload })
    try {
      open().reload()
    } finally {
      vi.unstubAllGlobals()
    }

    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('writes an error to the listed log when the vaults cannot be listed', async () => {
    const getWriter = vi.fn()
    const write = vi.fn()
    getWriter.mockReturnValue(write)
    const target = useVaults({
      core: {
        vaults: async () => {
          throw new Error('down')
        },
      } as never,
      words: WORDS,
      log: { getWriter } as never,
      chunks: ref(0),
      embedded: ref(0),
      isEmbedding: ref(false),
    })

    await target.loadVaults()

    expect(getWriter).toHaveBeenCalledWith('listed')
    expect(write).toHaveBeenCalledWith(WORDS.unlistedVaults, 'error')
    expect(target.vaultList.value).toEqual({ vaults: [], showing: '' })
  })
})
