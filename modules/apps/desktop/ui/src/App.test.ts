/**
 * What the window settles for itself, asked without a browser.
 *
 * The tabs, the plex and the agent are drawn; the word a tab carries beside its
 * title is a decision, and it is the one asked about here.
 */
import { describe, expect, it, vi } from 'vitest'
import type { State } from './tab'

// Neither port is reached: the window is asked about its own decisions, and
// both of these are built against a page that is not here.
vi.mock('./vault', () => ({ vault: {}, core: {} }))
vi.mock('./agent', () => ({ core: {} }))

const { markOf } = await import('./App.vue')

const marked = (state: State) => markOf(state)

describe('the word a note tab carries', () => {
  it('is nothing while the note is being read', () => {
    expect(marked('loading')).toBeUndefined()
  })

  it('is nothing when the text on screen is the text of the file', () => {
    expect(marked('clean')).toBeUndefined()
  })

  it('is unsaved while there is something to write', () => {
    expect(marked('unsaved')).toBe('unsaved')
  })

  it('is unsaved while the write is on its way', () => {
    expect(marked('saving')).toBe('unsaved')
  })

  it('is overtaken while the file has moved past what the tab read', () => {
    expect(marked('overtaken')).toBe('overtaken')
  })

  it('is stuck when the note can be neither read nor written', () => {
    expect(marked('stuck')).toBe('stuck')
  })

  it('is a different word for each of the three things a tab carries', () => {
    const words = [marked('stuck'), marked('overtaken'), marked('unsaved')]
    expect(new Set(words).size).toBe(3)
  })
})
