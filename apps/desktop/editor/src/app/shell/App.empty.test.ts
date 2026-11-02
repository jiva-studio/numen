/**
 * The window over a vault it could not read, and over one that holds no note.
 */
import { describe, expect, it } from 'vitest'
import { cards, mountWindow, vaultState } from '@/testing/window'

describe('the window with no note to show', () => {
  it('lists the vault that could not be read as a failed task', async () => {
    vaultState.opening = null
    vaultState.error = 'the vault folder is not there'

    const window = await mountWindow()

    const corner = cards(window)
    expect(corner[0]).toContain('the vault folder is not there')
    expect(corner).toHaveLength(1)
  })

  it('says nothing when the vault was read and holds none', async () => {
    vaultState.opening = null
    vaultState.error = ''

    const window = await mountWindow()

    expect(cards(window)).toStrictEqual([])
  })
})
