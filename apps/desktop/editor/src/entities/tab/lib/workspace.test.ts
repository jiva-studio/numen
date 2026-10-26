/**
 * The names tabs and conversations are filed under, and what a tab is called.
 *
 * Two tabs under one name are one tab to the workspace, and two conversations
 * under one name are one conversation to the agent. The second is the worse of
 * the two: a question lands in a talk nobody can see.
 */
import { describe, expect, it } from 'vitest'
import { getPanes } from '@numen/ui'
import {
  BOOK,
  CONVERSATION,
  DOCUMENT,
  NOTE,
  RECORDING,
  SETTINGS,
  SETTINGS_FILE,
  URL,
  createWorkspace,
  generateId,
} from './workspace'

describe('the identity a tab is filed under', () => {
  it('is a new one every time, so a second of a kind is a second tab', () => {
    const names = [generateId(NOTE), generateId(NOTE), generateId(NOTE)]

    expect(new Set(names).size).toBe(3)
  })

  it('is never the name of a tab of another kind', () => {
    expect(generateId(NOTE)).not.toBe(generateId(DOCUMENT))
  })

  it('says what kind of thing the tab holds', () => {
    expect(generateId(SETTINGS).startsWith(`${SETTINGS}:`)).toBe(true)
  })
})

describe('the kinds a tab is filed under', () => {
  const kinds = [NOTE, DOCUMENT, BOOK, RECORDING, URL, SETTINGS, SETTINGS_FILE, CONVERSATION]

  it('are each a word, so an identity names its kind', () => {
    for (const kind of kinds) expect(kind).not.toBe('')
  })

  it('are each their own, so two kinds never share an identity', () => {
    expect(new Set(kinds).size).toBe(kinds.length)
  })
})

describe('the name a conversation is answered under', () => {
  it('is a new one for every conversation opened', () => {
    const names = [generateId(CONVERSATION), generateId(CONVERSATION), generateId(CONVERSATION)]

    expect(new Set(names).size).toBe(3)
  })
})

describe('the layout the window opens with', () => {
  const createLayout = () => createWorkspace(['plex:one'], ['agent:one', 'files:tree'])

  it('holds two panes, the plex alone and the agent over the files', () => {
    const layout = createLayout()
    expect(getPanes(layout.root).map((one) => one.tabs)).toStrictEqual([
      ['plex:one'],
      ['agent:one', 'files:tree'],
    ])
  })

  it('remains the agent in front of the two the trailing pane holds', () => {
    const layout = createLayout()
    expect(getPanes(layout.root).find((one) => one.id === 'aside')?.active).toBe('agent:one')
  })

  it('gives the plex the room, and the pane alongside it the rest', () => {
    const layout = createLayout()
    const sizes = layout.root.kind === 'branch' ? layout.root.sizes : []

    expect(sizes).toStrictEqual([0.72, 0.28])
    expect(Math.max(...sizes)).toBe(sizes[0])
  })

  it('divides the width, so the two stand side by side', () => {
    const layout = createLayout()
    expect(layout.axis).toBe('horizontal')
  })

  it('names the root after what it divides, and the panes after their places', () => {
    const layout = createLayout()
    expect(layout.root.id).toBe('root')
    expect(getPanes(layout.root).map((one) => one.id)).toStrictEqual(['main', 'aside'])
  })

  it('leaves the person in the plex', () => {
    const layout = createLayout()
    expect(layout.focus).toBe('main')
    expect(getPanes(layout.root).find((one) => one.id === 'main')?.tabs).toStrictEqual(['plex:one'])
  })
})

describe('the layout of a window with one side empty', () => {
  it('is the main tabs alone where nothing remains aside', () => {
    const layout = createWorkspace(['plex:one'], [])
    expect(getPanes(layout.root).map((one) => [one.id, one.tabs])).toStrictEqual([
      ['main', ['plex:one']],
    ])
    expect(layout.focus).toBe('main')
  })

  it('is the aside tabs, as the whole window, where nothing remains in the main pane', () => {
    const layout = createWorkspace([], ['agent:one', 'files:tree'])
    expect(getPanes(layout.root).map((one) => [one.id, one.tabs, one.active])).toStrictEqual([
      ['main', ['agent:one', 'files:tree'], 'agent:one'],
    ])
  })

  it('is one empty pane where no kind is placed', () => {
    const layout = createWorkspace([], [])
    expect(getPanes(layout.root).map((one) => one.tabs)).toStrictEqual([[]])
  })
})
