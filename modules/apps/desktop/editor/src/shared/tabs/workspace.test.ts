/**
 * The names tabs and conversations are filed under, and what a tab is called.
 *
 * Two tabs under one name are one tab to the workspace, and two conversations
 * under one name are one conversation to the agent. The second is the worse of
 * the two: a question lands in a talk nobody can see.
 */
import { describe, expect, it, vi } from 'vitest'
import { panesOf } from '@numen/ui'
import { AGENT, CONVERSATION, PLEX, begun, minted } from './workspace'

describe('the identity a tab is filed under', () => {
  it('is a new one every time, so a second of a kind is a second tab', () => {
    const names = [minted(PLEX), minted(PLEX), minted(PLEX)]

    expect(new Set(names).size).toBe(3)
  })

  it('is never the name of a tab of another kind', () => {
    expect(minted(PLEX)).not.toBe(minted(AGENT))
  })

  it('says what kind of thing the tab holds', () => {
    expect(minted(AGENT).startsWith(`${AGENT}:`)).toBe(true)
  })
})

describe('the name a conversation is answered under', () => {
  it('is a new one for every conversation opened', () => {
    const names = [minted(CONVERSATION), minted(CONVERSATION), minted(CONVERSATION)]

    expect(new Set(names).size).toBe(3)
  })

  /**
   * A reloaded page has forgotten every name it gave out, and the agent holds
   * the conversation each of them stands for. The next name is new to both.
   */
  it('is not one the page gave out before it was reloaded', async () => {
    const before = [minted(CONVERSATION), minted(CONVERSATION)]

    // The page again, with everything it held forgotten.
    vi.resetModules()
    const reloaded = await import('./workspace')
    const after = [reloaded.minted(CONVERSATION), reloaded.minted(CONVERSATION)]

    expect(new Set([...before, ...after]).size).toBe(4)
  })
})

describe('the layout the window opens with', () => {
  const layout = begun('plex:one', 'agent:one', 'files:tree')

  it('holds two panes, the plex alone and the agent over the files', () => {
    expect(panesOf(layout.root).map((one) => one.tabs)).toStrictEqual([
      ['plex:one'],
      ['agent:one', 'files:tree'],
    ])
  })

  it('stands the agent in front of the two the trailing pane holds', () => {
    expect(panesOf(layout.root).find((one) => one.id === 'aside')?.active).toBe('agent:one')
  })

  it('gives the plex the room, and the pane beside it the rest', () => {
    const sizes = layout.root.kind === 'branch' ? layout.root.sizes : []

    expect(sizes).toStrictEqual([0.72, 0.28])
    expect(Math.max(...sizes)).toBe(sizes[0])
  })

  it('divides the width, so the two stand side by side', () => {
    expect(layout.axis).toBe('horizontal')
  })

  it('leaves the person in the plex', () => {
    expect(layout.focus).toBe('main')
    expect(panesOf(layout.root).find((one) => one.id === 'main')?.tabs).toStrictEqual(['plex:one'])
  })
})

