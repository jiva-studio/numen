/**
 * The names tabs and conversations are filed under, and what a tab is called.
 *
 * Two tabs under one name are one tab to the workspace, and two conversations
 * under one name are one conversation to the agent. The second is the worse of
 * the two: a question lands in a talk nobody can see.
 */
import { describe, expect, it, vi } from 'vitest'
import { AGENT, CONVERSATION, PLEX, named, plexCalled, shortened } from './workspace'

describe('the name a tab is filed under', () => {
  it('is a new one every time, so a second of a kind is a second tab', () => {
    const names = [named(PLEX), named(PLEX), named(PLEX)]

    expect(new Set(names).size).toBe(3)
  })

  it('is never the name of a tab of another kind', () => {
    expect(named(PLEX)).not.toBe(named(AGENT))
  })

  it('says what kind of thing the tab holds', () => {
    expect(named(AGENT).startsWith(`${AGENT}:`)).toBe(true)
  })
})

describe('the name a conversation is answered under', () => {
  it('is a new one for every conversation opened', () => {
    const names = [named(CONVERSATION), named(CONVERSATION), named(CONVERSATION)]

    expect(new Set(names).size).toBe(3)
  })

  /**
   * A reloaded page has forgotten every name it gave out, and the agent holds
   * the conversation each of them stands for. The next name is new to both.
   */
  it('is not one the page gave out before it was reloaded', async () => {
    const before = [named(CONVERSATION), named(CONVERSATION)]

    // The page again, with everything it held forgotten.
    vi.resetModules()
    const reloaded = await import('./workspace')
    const after = [reloaded.named(CONVERSATION), reloaded.named(CONVERSATION)]

    expect(new Set([...before, ...after]).size).toBe(4)
  })
})

describe('what a plex tab is called', () => {
  it('is the note it stands on, under the word for a plex', () => {
    expect(plexCalled('Plex', 'Entropy')).toBe('Plex · Entropy')
  })

  it('is the word alone while it stands nowhere', () => {
    expect(plexCalled('Plex', '')).toBe('Plex')
  })

  it('is never what the tab holding that note is called', () => {
    expect(plexCalled('Plex', 'Entropy')).not.toBe('Entropy')
  })
})

describe('what a tab is called by a question', () => {
  it('is the question itself when it is short enough to read', () => {
    expect(shortened('what is here?')).toBe('what is here?')
  })

  it('is the first line of one written over several', () => {
    expect(shortened('what is here?\nand below it?')).toBe('what is here?')
  })

  it('is the first line with anything on it', () => {
    expect(shortened('\nwhat is here?')).toBe('what is here?')
  })

  it('is cut at a word rather than through one', () => {
    const asked = 'what is the note about entropy joined to'

    expect(shortened(asked, 24)).toBe('what is the note about…')
  })

  it('is cut where it has to be when there is nothing to cut at', () => {
    expect(shortened('x'.repeat(40), 10)).toBe(`${'x'.repeat(10)}…`)
  })

  it('is nothing when nothing has been asked', () => {
    expect(shortened('')).toBe('')
  })
})
