/**
 * The window keeping tabs of kinds it knows nothing about.
 *
 * Everything asked here is asked of a kind made up for the test: what the
 * window does with a kind is the whole of what a real one can count on.
 */
import { describe, expect, it } from 'vitest'
import { panesOf } from '@numen/ui'
import type { WorkspaceLayout } from '@numen/ui'
import { windowing, type Host, type Kept } from './windowing'

const words = { newTab: 'New tab' }

/**
 * A kind that records what it was asked to do, under the names it opened on. A
 * kind that keeps its tabs has something to finish and never lets one go.
 */
const kind = ({ keeps = false, ...over }: Partial<Kept> & { keeps?: boolean } = {}) => {
  const opened: string[] = []
  const shut: string[] = []
  const seen: string[] = []
  const ids: string[] = []
  const one: Kept = {
    kind: 'thing',
    opens: (at: string, id: string) => {
      opened.push(at)
      ids.push(id)
      return { at, title: at || 'a thing' }
    },
    called: (held: { title: string }) => held.title,
    draws: {},
    shuts: (held: { at: string }) => {
      shut.push(held.at)
      return !keeps
    },
    shown: (held: { at: string }) => {
      seen.push(held.at)
    },
    ...over,
  }
  return { declared: () => one, one, opened, ids, shut, seen }
}

/** Every tab on the screen, whichever pane it is in. */
const onScreen = (layout: WorkspaceLayout) => panesOf(layout.root).flatMap((pane) => pane.tabs)

describe('a tab of a kind', () => {
  it('is opened by the kind, drawn, and called what the kind calls it', async () => {
    const thing = kind()
    const window = windowing([thing.declared], words)

    const id = await window.opens('thing', 'Note.md')

    expect(thing.opened).toEqual(['Note.md'])
    expect(onScreen(window.layout.value)).toContain(id)
    expect(window.tabs.value).toEqual([{ id, title: 'Note.md' }])
    expect(window.heldIn(id)?.held).toEqual({ at: 'Note.md', title: 'Note.md' })
  })

  it('carries the word its kind gives it, and none where the kind gives none', async () => {
    const marked = kind({ marked: (held: { at: string }) => (held.at ? 'unsaved' : undefined) })
    const window = windowing([marked.declared], words)

    const one = await window.opens('thing', 'Note.md')
    const other = await window.opens('thing')

    expect(window.tabs.value.find((tab) => tab.id === one)?.mark).toBe('unsaved')
    expect(window.tabs.value.find((tab) => tab.id === other)).not.toHaveProperty('mark')
  })

  it('is a tab of its own each time, for a kind that takes no identity', async () => {
    const thing = kind()
    const window = windowing([thing.declared], words)

    const one = await window.opens('thing')
    const other = await window.opens('thing')

    expect(one).not.toBe(other)
    expect(thing.opened).toHaveLength(2)
  })

  it('is the tab already open, for a kind whose identity is what it opens on', async () => {
    const thing = kind({ identity: (at: string) => at })
    const window = windowing([thing.declared], words)

    const one = await window.opens('thing', 'Note.md')
    const again = await window.opens('thing', 'Note.md')

    expect(again).toBe(one)
    expect(thing.opened).toEqual(['Note.md'])
  })

  it('is nothing for a kind the window was never told about', async () => {
    const window = windowing([kind().declared], words)

    expect(await window.opens('nothing')).toBe('')
    expect(onScreen(window.layout.value)).toEqual([])
  })

  it('is told the identity it will carry, so its kind can find it again', async () => {
    const thing = kind()
    const window = windowing([thing.declared], words)

    const id = await window.opens('thing', 'Note.md')

    expect(thing.ids).toEqual([id])
  })
})

describe('what a kind is given', () => {
  it('lets it open a tab of another kind, and put one it holds in front', async () => {
    const other = kind({ kind: 'other' })
    let host: Host | null = null
    const thing = kind()
    const window = windowing([(given: Host) => ((host = given), thing.one), other.declared], words)

    const opened = await host!.opens('other', 'Note.md')
    const mine = await window.opens('thing')
    host!.shows(opened)

    expect(other.opened).toEqual(['Note.md'])
    expect(onScreen(window.layout.value)).toContain(mine)
    expect(window.heldIn(opened)).not.toBeNull()
  })
})

describe('a tab that closes', () => {
  it('lets go of what it held', async () => {
    const thing = kind()
    const window = windowing([thing.declared], words)
    const id = await window.opens('thing', 'Note.md')

    expect(window.shut(id)).toBe(true)
    expect(thing.shut).toEqual(['Note.md'])
    expect(window.heldIn(id)).toBeNull()
  })

  it('stays where it is while its kind has something to finish', async () => {
    const holding = kind({ keeps: true })
    const window = windowing([holding.declared], words)
    const id = await window.opens('thing', 'Note.md')

    expect(window.shut(id)).toBe(false)
    expect(window.heldIn(id)?.held).toBeTruthy()
    expect(onScreen(window.layout.value)).toContain(id)
  })

  it('goes when the kind that took it says it is done', async () => {
    const holding = kind({ keeps: true })
    const window = windowing([holding.declared], words)
    const id = await window.opens('thing', 'Note.md')
    window.shut(id)

    window.host.closes(id)

    expect(window.heldIn(id)).toBeNull()
    expect(onScreen(window.layout.value)).not.toContain(id)
  })
})

describe('a tab with nothing in it yet', () => {
  it('is called the word for a new tab, and offers what the kinds offer', async () => {
    const thing = kind({ offers: 'New thing' })
    const other = kind({ kind: 'other', offers: 'New other' })
    const quiet = kind({ kind: 'quiet' })
    const window = windowing([thing.declared, other.declared, quiet.declared], words)

    window.blanked('main')

    expect(window.tabs.value).toEqual([{ id: window.blanks.value[0], title: 'New tab' }])
    expect(window.becomes.value).toEqual([
      { id: 'thing', title: 'New thing' },
      { id: 'other', title: 'New other' },
    ])
  })

  it('is what it was told to be, standing where it stood', async () => {
    const thing = kind({ offers: 'New thing' })
    const window = windowing([thing.declared], words)
    window.blanked('main')
    const blank = window.blanks.value[0] ?? ''

    await window.becomeIt(blank, 'thing')

    expect(window.blanks.value).toEqual([])
    expect(onScreen(window.layout.value)).not.toContain(blank)
    expect(onScreen(window.layout.value)).toHaveLength(1)
  })

  it('opens on what its kind had to make first', async () => {
    const thing = kind({ offers: 'New thing', makes: async () => 'Made.md' })
    const window = windowing([thing.declared], words)
    window.blanked('main')

    await window.becomeIt(window.blanks.value[0] ?? '', 'thing')

    expect(thing.opened).toEqual(['Made.md'])
  })

  it('stays blank when its kind made nothing', async () => {
    const thing = kind({ offers: 'New thing', makes: async () => '' })
    const window = windowing([thing.declared], words)
    window.blanked('main')
    const blank = window.blanks.value[0] ?? ''

    await window.becomeIt(blank, 'thing')

    expect(thing.opened).toEqual([])
    expect(window.blanks.value).toEqual([blank])
  })

  it('goes on its own when it is closed with nothing in it', () => {
    const window = windowing([kind().declared], words)
    window.blanked('main')
    const blank = window.blanks.value[0] ?? ''

    expect(window.shut(blank)).toBe(true)
    expect(window.blanks.value).toEqual([])
  })
})

describe('the tab now on screen', () => {
  it('is told, so what it holds has room to measure', async () => {
    const thing = kind()
    const window = windowing([thing.declared], words)
    const id = await window.opens('thing', 'Note.md')

    window.shown(id)

    expect(thing.seen).toEqual(['Note.md'])
  })

  it('is nothing to a window that no longer holds it', () => {
    const thing = kind()
    const window = windowing([thing.declared], words)

    window.shown('gone')

    expect(thing.seen).toEqual([])
  })
})

describe('the window going', () => {
  it('says so to every tab, and waits for none of them', async () => {
    const thing = kind({ keeps: true })
    const window = windowing([thing.declared], words)
    await window.opens('thing', 'One.md')
    await window.opens('thing', 'Two.md')

    window.close()

    expect(thing.shut).toEqual(['One.md', 'Two.md'])
  })

  it('lets a kind that has its own way of going take it, and asks no more', async () => {
    const going: string[] = []
    const thing = kind({ gone: (held: { at: string }) => going.push(held.at) })
    const window = windowing([thing.declared], words)
    await window.opens('thing', 'One.md')

    window.close()

    expect(going).toEqual(['One.md'])
    expect(thing.shut).toEqual([])
  })
})
