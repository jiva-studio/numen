/**
 * The window keeping tabs of kinds it knows nothing about.
 *
 * Everything asked here is asked of a kind made up for the test: what the
 * window does with a kind is the whole of what a real one can count on.
 */
import { describe, expect, it } from 'vitest'
import { panesOf } from '@numen/ui'
import type { Workspace } from '@numen/ui'
import type { AnyTabKind, WindowHandle } from './kinds'
import { useWindowTabs } from './windowTabs'

/**
 * A kind that records what it was asked to do, under the names it opened on. A
 * kind that keeps its tabs has something to finish and never lets one go.
 */
const kind = ({ keeps = false, ...over }: Partial<AnyTabKind> & { keeps?: boolean } = {}) => {
  const opened: string[] = []
  const shut: string[] = []
  const seen: string[] = []
  const one: AnyTabKind = {
    kind: 'thing',
    open: (at: string) => {
      opened.push(at)
      return { at, title: at || 'a thing' }
    },
    getTitle: (state: { title: string }) => state.title,
    pane: {},
    onClose: (state: { at: string }) => {
      shut.push(state.at)
      return !keeps
    },
    onShow: (state: { at: string }) => {
      seen.push(state.at)
    },
    ...over,
  }
  return { declared: () => one, one, opened, shut, seen }
}

/** A window told what kinds it draws, each of them made with what it is given. */
const told = (declared: readonly ((handle: WindowHandle) => AnyTabKind)[]) => {
  const window = useWindowTabs()
  window.registerKinds(declared.map((one) => one(window.handle)))
  return window
}

/** Every tab on the screen, whichever pane it is in. */
const onScreen = (layout: Workspace) => panesOf(layout.root).flatMap((pane) => pane.tabs)

describe('a tab of a kind', () => {
  it('is opened by the kind, drawn, and called what the kind calls it', async () => {
    const thing = kind()
    const window = told([thing.declared])

    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(thing.opened).toEqual(['Note.md'])
    expect(onScreen(window.layout.value)).toContain(id)
    expect(window.tabs.value).toEqual([{ id, title: 'Note.md' }])
    expect(window.getTab(id)?.state).toEqual({ at: 'Note.md', title: 'Note.md' })
  })

  it('carries the word its kind gives it, and none where the kind gives none', async () => {
    const marked = kind({ getMark: (state: { at: string }) => (state.at ? 'unsaved' : undefined) })
    const window = told([marked.declared])

    const one = await window.openTabOfKind('thing', 'Note.md')
    const other = await window.openTabOfKind('thing')

    expect(window.tabs.value.find((tab) => tab.id === one)?.mark).toBe('unsaved')
    expect(window.tabs.value.find((tab) => tab.id === other)).not.toHaveProperty('mark')
  })

  it('is a tab of its own each time, for a kind that takes no identity', async () => {
    const thing = kind()
    const window = told([thing.declared])

    const one = await window.openTabOfKind('thing')
    const other = await window.openTabOfKind('thing')

    expect(one).not.toBe(other)
    expect(thing.opened).toHaveLength(2)
  })

  it('is the tab already open, for a kind whose identity is what it opens on', async () => {
    const thing = kind({ identity: (at: string) => at })
    const window = told([thing.declared])

    const one = await window.openTabOfKind('thing', 'Note.md')
    const again = await window.openTabOfKind('thing', 'Note.md')

    expect(again).toBe(one)
    expect(thing.opened).toEqual(['Note.md'])
  })

  it('is a tab of its own where another kind names one after the same thing', async () => {
    const thing = kind({ identity: (at: string) => at })
    const other = kind({ kind: 'other', identity: (at: string) => at })
    const window = told([thing.declared, other.declared])

    const one = await window.openTabOfKind('thing', 'Note.md')
    const another = await window.openTabOfKind('other', 'Note.md')

    expect(another).not.toBe(one)
    expect(window.getTab(one)?.kind.kind).toBe('thing')
    expect(window.getTab(another)?.kind.kind).toBe('other')
  })

  it('is nothing for a kind the window was never told about', async () => {
    const window = told([kind().declared])

    expect(await window.openTabOfKind('nothing')).toBe('')
    expect(onScreen(window.layout.value)).toEqual([])
  })
})

describe('what a kind is given', () => {
  it('lets it open a tab of another kind, and put one it holds in front', async () => {
    const other = kind({ kind: 'other' })
    let handle: WindowHandle | null = null
    const thing = kind()
    const window = told([(given: WindowHandle) => ((handle = given), thing.one), other.declared])

    const opened = await handle!.openTab('other', 'Note.md')
    const mine = await window.openTabOfKind('thing')
    handle!.show(opened)

    expect(other.opened).toEqual(['Note.md'])
    expect(onScreen(window.layout.value)).toContain(mine)
    expect(window.getTab(opened)).not.toBeNull()
  })
})

describe('a tab that closes', () => {
  it('lets go of what it held', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.shut(id)).toBe(true)
    expect(thing.shut).toEqual(['Note.md'])
    expect(window.getTab(id)).toBeNull()
  })

  it('says it went, for an identity the window does not hold', () => {
    const window = told([kind().declared])

    expect(window.shut('gone')).toBe(true)
  })

  it('stays where it is while its kind has something to finish', async () => {
    const holding = kind({ keeps: true })
    const window = told([holding.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.shut(id)).toBe(false)
    expect(window.getTab(id)?.state).toBeTruthy()
    expect(onScreen(window.layout.value)).toContain(id)
  })

  it('goes when the kind that took it says it is done', async () => {
    const holding = kind({ keeps: true })
    const window = told([holding.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')
    window.shut(id)

    window.handle.closeTab(id)

    expect(window.getTab(id)).toBeNull()
    expect(onScreen(window.layout.value)).not.toContain(id)
  })
})

describe('a tab let go of from outside', () => {
  it('lets go of what it held and comes off the screen', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.requestClose(id)

    expect(thing.shut).toEqual(['Note.md'])
    expect(window.getTab(id)).toBeNull()
    expect(onScreen(window.layout.value)).not.toContain(id)
  })

  it('stays on the screen while its kind has something to finish', async () => {
    const holding = kind({ keeps: true })
    const window = told([holding.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.requestClose(id)

    expect(window.getTab(id)?.state).toBeTruthy()
    expect(onScreen(window.layout.value)).toContain(id)
  })
})

describe('the tab now on screen', () => {
  it('is told, so what it holds has room to measure', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.onTabShown(id)

    expect(thing.seen).toEqual(['Note.md'])
  })

  it('is nothing to a window that no longer holds it', () => {
    const thing = kind()
    const window = told([thing.declared])

    window.onTabShown('gone')

    expect(thing.seen).toEqual([])
  })
})

describe('the tab the person is looking at', () => {
  it('is the one showing in the pane in front, with its kind and what it holds', async () => {
    const thing = kind()
    const window = told([thing.declared])
    await window.openTabOfKind('thing', 'One.md')
    const two = await window.openTabOfKind('thing', 'Two.md')

    expect(window.handle.front()).toEqual({
      id: two,
      kind: 'thing',
      state: { at: 'Two.md', title: 'Two.md' },
    })
  })

  it('is the tab of the pane in front, whichever pane reported itself last', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const one = await window.openTabOfKind('thing', 'One.md')
    const two = await window.handle.beside('thing', 'Two.md')
    window.show(one)
    // Every pane says what it is showing when it is drawn.
    window.onTabShown(two)

    expect(window.handle.front()?.id).toBe(one)
    expect(window.handle.last('thing')?.id).toBe(two)
  })

  it('answers under no kind for a tab the window has let go of', async () => {
    const thing = kind()
    const window = told([thing.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')
    window.shut(id)

    expect(window.handle.front()).toEqual({ id, kind: null, state: null })
  })

  it('is nothing while the pane in front holds no tab', () => {
    const window = told([kind().declared])

    expect(window.handle.front()).toBeNull()
  })
})

describe('a key struck on the window', () => {
  /** A kind whose tabs take the arrows and record what they were struck with. */
  const reading = () => {
    const took: string[] = []
    return {
      took,
      one: kind({
        onKeyPress: (state: { at: string }, event: KeyboardEvent) => {
          if (!event.key.startsWith('Arrow')) return false
          took.push(`${state.at} ${event.key}`)
          return true
        },
      }),
    }
  }

  const createKeyEvent = (key: string) => new KeyboardEvent('keydown', { key })

  it('goes to the tab of the pane the person is in, and to no other', async () => {
    // Two panes are drawn at once, so a tab that listened for itself would have
    // both of them answering the one key.
    const read = reading()
    const window = told([read.one.declared])
    const one = await window.openTabOfKind('thing', 'One.epub')
    await window.handle.beside('thing', 'Two.epub')
    window.show(one)

    expect(window.onKeyPress(createKeyEvent('ArrowRight'))).toBe(true)

    expect(read.took).toStrictEqual(['One.epub ArrowRight'])
  })

  it('goes to the other pane once the person is in it', async () => {
    const read = reading()
    const window = told([read.one.declared])
    await window.openTabOfKind('thing', 'One.epub')
    const two = await window.handle.beside('thing', 'Two.epub')
    window.show(two)

    window.onKeyPress(createKeyEvent('ArrowLeft'))

    expect(read.took).toStrictEqual(['Two.epub ArrowLeft'])
  })

  it('is left alone where the tab in front does not read it', async () => {
    const read = reading()
    const window = told([read.one.declared])
    await window.openTabOfKind('thing', 'One.epub')

    expect(window.onKeyPress(createKeyEvent('k'))).toBe(false)
    expect(read.took).toStrictEqual([])
  })

  it('is left alone where the kind in front reads no key at all', async () => {
    const window = told([kind().declared])
    await window.openTabOfKind('thing', 'One.md')

    expect(window.onKeyPress(createKeyEvent('ArrowRight'))).toBe(false)
  })

  it('is left alone while the pane in front holds no tab', () => {
    const window = told([reading().one.declared])

    expect(window.onKeyPress(createKeyEvent('ArrowRight'))).toBe(false)
  })
})

describe('the window going', () => {
  it('says so to every tab, and waits for none of them', async () => {
    const thing = kind({ keeps: true })
    const window = told([thing.declared])
    await window.openTabOfKind('thing', 'One.md')
    await window.openTabOfKind('thing', 'Two.md')

    window.close()

    expect(thing.shut).toEqual(['One.md', 'Two.md'])
  })

  it('lets a kind that has its own way of going take it, and asks no more', async () => {
    const going: string[] = []
    const thing = kind({ onDestroy: (state: { at: string }) => going.push(state.at) })
    const window = told([thing.declared])
    await window.openTabOfKind('thing', 'One.md')

    window.close()

    expect(going).toEqual(['One.md'])
    expect(thing.shut).toEqual([])
  })
})
