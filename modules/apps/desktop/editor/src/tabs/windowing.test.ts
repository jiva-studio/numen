/**
 * The window keeping tabs of kinds it knows nothing about.
 *
 * Everything asked here is asked of a kind made up for the test: what the
 * window does with a kind is the whole of what a real one can count on.
 */
import { describe, expect, it } from 'vitest'
import { panesOf } from '@numen/ui'
import type { WorkspaceLayout } from '@numen/ui'
import { windowing, type AnyKind, type WindowHandle } from './windowing'

/**
 * A kind that records what it was asked to do, under the names it opened on. A
 * kind that keeps its tabs has something to finish and never lets one go.
 */
const kind = ({ keeps = false, ...over }: Partial<AnyKind> & { keeps?: boolean } = {}) => {
  const opened: string[] = []
  const shut: string[] = []
  const seen: string[] = []
  const one: AnyKind = {
    kind: 'thing',
    opens: (at: string) => {
      opened.push(at)
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
  return { declared: () => one, one, opened, shut, seen }
}

/** A window told what kinds it draws, each of them made with what it is given. */
const told = (declared: readonly ((handle: WindowHandle) => AnyKind)[]) => {
  const window = windowing()
  window.declares(declared.map((one) => one(window.handle)))
  return window
}

/** Every tab on the screen, whichever pane it is in. */
const onScreen = (layout: WorkspaceLayout) => panesOf(layout.root).flatMap((pane) => pane.tabs)

describe('a tab of a kind', () => {
  it('is opened by the kind, drawn, and called what the kind calls it', async () => {
    const thing = kind()
    const window = told([thing.declared])

    const id = await window.opens('thing', 'Note.md')

    expect(thing.opened).toEqual(['Note.md'])
    expect(onScreen(window.layout.value)).toContain(id)
    expect(window.tabs.value).toEqual([{ id, title: 'Note.md' }])
    expect(window.heldIn(id)?.held).toEqual({ at: 'Note.md', title: 'Note.md' })
  })

  it('carries the word its kind gives it, and none where the kind gives none', async () => {
    const marked = kind({ marked: (held: { at: string }) => (held.at ? 'unsaved' : undefined) })
    const window = told([marked.declared])

    const one = await window.opens('thing', 'Note.md')
    const other = await window.opens('thing')

    expect(window.tabs.value.find((tab) => tab.id === one)?.mark).toBe('unsaved')
    expect(window.tabs.value.find((tab) => tab.id === other)).not.toHaveProperty('mark')
  })

  it('is a tab of its own each time, for a kind that takes no identity', async () => {
    const thing = kind()
    const window = told([thing.declared])

    const one = await window.opens('thing')
    const other = await window.opens('thing')

    expect(one).not.toBe(other)
    expect(thing.opened).toHaveLength(2)
  })

  it('is the tab already open, for a kind whose identity is what it opens on', async () => {
    const thing = kind({ identity: (at: string) => at })
    const window = told([thing.declared])

    const one = await window.opens('thing', 'Note.md')
    const again = await window.opens('thing', 'Note.md')

    expect(again).toBe(one)
    expect(thing.opened).toEqual(['Note.md'])
  })

  it('is a tab of its own where another kind names one after the same thing', async () => {
    const thing = kind({ identity: (at: string) => at })
    const other = kind({ kind: 'other', identity: (at: string) => at })
    const window = told([thing.declared, other.declared])

    const one = await window.opens('thing', 'Note.md')
    const another = await window.opens('other', 'Note.md')

    expect(another).not.toBe(one)
    expect(window.heldIn(one)?.kind.kind).toBe('thing')
    expect(window.heldIn(another)?.kind.kind).toBe('other')
  })

  it('is nothing for a kind the window was never told about', async () => {
    const window = told([kind().declared])

    expect(await window.opens('nothing')).toBe('')
    expect(onScreen(window.layout.value)).toEqual([])
  })
})

describe('what a kind is given', () => {
  it('lets it open a tab of another kind, and put one it holds in front', async () => {
    const other = kind({ kind: 'other' })
    let handle: WindowHandle | null = null
    const thing = kind()
    const window = told([(given: WindowHandle) => ((handle = given), thing.one), other.declared])

    const opened = await handle!.opens('other', 'Note.md')
    const mine = await window.opens('thing')
    handle!.shows(opened)

    expect(other.opened).toEqual(['Note.md'])
    expect(onScreen(window.layout.value)).toContain(mine)
    expect(window.heldIn(opened)).not.toBeNull()
  })
})

describe('a tab that closes', () => {
  it('lets go of what it held', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const id = await window.opens('thing', 'Note.md')

    expect(window.shut(id)).toBe(true)
    expect(thing.shut).toEqual(['Note.md'])
    expect(window.heldIn(id)).toBeNull()
  })

  it('says it went, for an identity the window does not hold', () => {
    const window = told([kind().declared])

    expect(window.shut('gone')).toBe(true)
  })

  it('stays where it is while its kind has something to finish', async () => {
    const holding = kind({ keeps: true })
    const window = told([holding.declared])
    const id = await window.opens('thing', 'Note.md')

    expect(window.shut(id)).toBe(false)
    expect(window.heldIn(id)?.held).toBeTruthy()
    expect(onScreen(window.layout.value)).toContain(id)
  })

  it('goes when the kind that took it says it is done', async () => {
    const holding = kind({ keeps: true })
    const window = told([holding.declared])
    const id = await window.opens('thing', 'Note.md')
    window.shut(id)

    window.handle.closes(id)

    expect(window.heldIn(id)).toBeNull()
    expect(onScreen(window.layout.value)).not.toContain(id)
  })
})

describe('a tab let go of from outside', () => {
  it('lets go of what it held and comes off the screen', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const id = await window.opens('thing', 'Note.md')

    window.drops(id)

    expect(thing.shut).toEqual(['Note.md'])
    expect(window.heldIn(id)).toBeNull()
    expect(onScreen(window.layout.value)).not.toContain(id)
  })

  it('stays on the screen while its kind has something to finish', async () => {
    const holding = kind({ keeps: true })
    const window = told([holding.declared])
    const id = await window.opens('thing', 'Note.md')

    window.drops(id)

    expect(window.heldIn(id)?.held).toBeTruthy()
    expect(onScreen(window.layout.value)).toContain(id)
  })
})

describe('the tab now on screen', () => {
  it('is told, so what it holds has room to measure', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const id = await window.opens('thing', 'Note.md')

    window.shown(id)

    expect(thing.seen).toEqual(['Note.md'])
  })

  it('is nothing to a window that no longer holds it', () => {
    const thing = kind()
    const window = told([thing.declared])

    window.shown('gone')

    expect(thing.seen).toEqual([])
  })
})

describe('the tab the person is looking at', () => {
  it('is the one showing in the pane in front, with its kind and what it holds', async () => {
    const thing = kind()
    const window = told([thing.declared])
    await window.opens('thing', 'One.md')
    const two = await window.opens('thing', 'Two.md')

    expect(window.handle.front()).toEqual({
      id: two,
      kind: 'thing',
      held: { at: 'Two.md', title: 'Two.md' },
    })
  })

  it('is the tab of the pane in front, whichever pane reported itself last', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const one = await window.opens('thing', 'One.md')
    const two = await window.handle.beside('thing', 'Two.md')
    window.shows(one)
    // Every pane says what it is showing when it is drawn.
    window.shown(two)

    expect(window.handle.front()?.id).toBe(one)
    expect(window.handle.last('thing')?.id).toBe(two)
  })

  it('answers under no kind for a tab the window has let go of', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const id = await window.opens('thing', 'Note.md')
    window.shut(id)

    expect(window.handle.front()).toEqual({ id, kind: null, held: null })
  })

  it('is nothing while the pane in front holds no tab', () => {
    const window = told([kind().declared])

    expect(window.handle.front()).toBeNull()
  })
})

describe('the window going', () => {
  it('says so to every tab, and waits for none of them', async () => {
    const thing = kind({ keeps: true })
    const window = told([thing.declared])
    await window.opens('thing', 'One.md')
    await window.opens('thing', 'Two.md')

    window.close()

    expect(thing.shut).toEqual(['One.md', 'Two.md'])
  })

  it('lets a kind that has its own way of going take it, and asks no more', async () => {
    const going: string[] = []
    const thing = kind({ gone: (held: { at: string }) => going.push(held.at) })
    const window = told([thing.declared])
    await window.opens('thing', 'One.md')

    window.close()

    expect(going).toEqual(['One.md'])
    expect(thing.shut).toEqual([])
  })
})
