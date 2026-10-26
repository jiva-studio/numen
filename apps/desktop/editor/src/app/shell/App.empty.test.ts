/**
 * The window over a vault it could not read, and over one that holds no note.
 */
import { describe, expect, it } from 'vitest'
import { cards, mountWindow, said } from '@/testing/window'

describe('the window with no note to show', () => {
  it('says the vault could not be read, and that nothing was read from it', async () => {
    said.opening = null
    said.error = 'the vault folder is not there'

    const window = await mountWindow()

    const corner = cards(window)
    expect(corner[0]).toContain('the vault folder is not there')
    expect(corner[1]).toBe('No notes loaded')
  })

  it('says nothing when the vault was read and holds none', async () => {
    said.opening = null
    said.error = ''

    const window = await mountWindow()

    expect(cards(window)).toStrictEqual([])
  })
})
