/**
 * The window keeping tabs of kinds it knows nothing about.
 *
 * Everything asked here is asked of a kind made up for the test: what the
 * window does with a kind is the whole of what a real one can count on.
 */
import { describe, expect, it } from 'vitest'
import { effectScope, onScopeDispose } from 'vue'
import { openTab, getPanes } from '@numen/ui'
import type { Workspace } from '@numen/ui'
import type { FileKind } from '@/entities/file/@x/tab'
import type { AnyTabKind, WindowHandle } from '../lib/kinds'
import { fileOpeners } from './openers'
import { useWindowTabs } from './windowTabs'

/**
 * A kind that records what it was asked to do, under the names it opened on. A
 * kind that keeps its tabs has something to finish and never lets one go.
 */
const kind = ({ isKept = false, ...over }: Partial<AnyTabKind> & { isKept?: boolean } = {}) => {
  const opened: string[] = []
  const closedPaths: string[] = []
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
      closedPaths.push(state.at)
      return !isKept
    },
    onShow: (state: { at: string }) => {
      seen.push(state.at)
    },
    ...over,
  }
  return { declared: () => one, one, opened, closedPaths, seen }
}

/** A window told what kinds it draws, each of them made with what it is given. */
const told = (declared: readonly ((handle: WindowHandle) => AnyTabKind)[]) => {
  const window = useWindowTabs()
  window.registerKinds(declared.map((one) => one(window.handle)))
  return window
}

/** Every tab on the screen, whichever pane it is in. */
const onScreen = (layout: Workspace) => getPanes(layout.root).flatMap((pane) => pane.tabs)

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

  it('is a plugin unavailable tab for a kind the window was never told about', async () => {
    const window = told([kind().declared])

    const id = await window.openTabOfKind('nothing', 'One.md')

    expect(id).toBe('nothing:One.md')
    expect(onScreen(window.layout.value)).toEqual([id])
    expect(window.getTab(id)?.kind.kind).toBe('nothing')
    expect(window.getTab(id)?.kind.getTitle?.(window.getTab(id)?.state)).toBe('Plugin unavailable')
    expect(window.getTab(id)?.kind.getOpenTab?.(window.getTab(id)?.state)).toEqual({ path: 'One.md' })
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

    expect(window.releaseTab(id)).toBe(true)
    expect(thing.closedPaths).toEqual(['Note.md'])
    expect(window.getTab(id)).toBeNull()
  })

  it('says it went, for an identity the window does not hold', () => {
    const window = told([kind().declared])

    expect(window.releaseTab('gone')).toBe(true)
  })

  it('stays where it is while its kind has something to finish', async () => {
    const holding = kind({ isKept: true })
    const window = told([holding.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.releaseTab(id)).toBe(false)
    expect(window.getTab(id)?.state).toBeTruthy()
    expect(onScreen(window.layout.value)).toContain(id)
  })

  it('goes when the kind that took it says it is done', async () => {
    const holding = kind({ isKept: true })
    const window = told([holding.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')
    window.releaseTab(id)

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

    expect(thing.closedPaths).toEqual(['Note.md'])
    expect(window.getTab(id)).toBeNull()
    expect(onScreen(window.layout.value)).not.toContain(id)
  })

  it('stays on the screen while its kind has something to finish', async () => {
    const holding = kind({ isKept: true })
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
    const two = await window.handle.openTabBeside('thing', 'Two.md')
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
    window.releaseTab(id)

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
    await window.handle.openTabBeside('thing', 'Two.epub')
    window.show(one)

    expect(window.onKeyPress(createKeyEvent('ArrowRight'))).toBe(true)

    expect(read.took).toStrictEqual(['One.epub ArrowRight'])
  })

  it('goes to the other pane once the person is in it', async () => {
    const read = reading()
    const window = told([read.one.declared])
    await window.openTabOfKind('thing', 'One.epub')
    const two = await window.handle.openTabBeside('thing', 'Two.epub')
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
    const thing = kind({ isKept: true })
    const window = told([thing.declared])
    await window.openTabOfKind('thing', 'One.md')
    await window.openTabOfKind('thing', 'Two.md')

    window.close()

    expect(thing.closedPaths).toEqual(['One.md', 'Two.md'])
  })

  it('lets a kind that has its own way of going take it, and asks no more', async () => {
    const going: string[] = []
    const thing = kind({ onDestroy: (state: { at: string }) => going.push(state.at) })
    const window = told([thing.declared])
    await window.openTabOfKind('thing', 'One.md')

    window.close()

    expect(going).toEqual(['One.md'])
    expect(thing.closedPaths).toEqual([])
  })
})

describe('reconciling tabs for a path', () => {
  it('closes existing tabs of a different kind for the same path', async () => {
    const noteKind = kind({
      kind: 'note',
      getOpenTab: (state: { at: string }) => ({ path: state.at }),
    })
    const deckKind = kind({
      kind: 'deck',
      getOpenTab: (state: { at: string }) => ({ path: state.at }),
    })
    const window = told([noteKind.declared, deckKind.declared])

    const noteTabId = await window.openTabOfKind('note', 'Cards.md')
    expect(window.tabs.value.map((t) => t.id)).toContain(noteTabId)

    window.reconcileTab('Cards.md', 'deck')

    expect(noteKind.closedPaths).toEqual(['Cards.md'])
    expect(window.getTab(noteTabId)).toBeNull()
  })

  it('keeps existing tabs when the kind matches', async () => {
    const noteKind = kind({
      kind: 'note',
      getOpenTab: (state: { at: string }) => ({ path: state.at }),
    })
    const window = told([noteKind.declared])

    const noteTabId = await window.openTabOfKind('note', 'Note.md')
    window.reconcileTab('Note.md', 'note')

    expect(noteKind.closedPaths).toEqual([])
    expect(window.getTab(noteTabId)).not.toBeNull()
  })

  it('preserves tool tabs like plex when reconciling file tabs', async () => {
    const plexKind = kind({
      kind: 'plex',
      getOpenTab: (state: { at: string }) => ({ path: state.at }),
    })
    const noteKind = kind({
      kind: 'note',
      getOpenTab: (state: { at: string }) => ({ path: state.at }),
    })
    const window = told([plexKind.declared, noteKind.declared])

    const plexTabId = await window.openTabOfKind('plex', 'Graph.md')
    const noteTabId = await window.openTabOfKind('note', 'Graph.md')

    window.reconcileTab('Graph.md', 'deck')

    expect(noteKind.closedPaths).toEqual(['Graph.md'])
    expect(plexKind.closedPaths).toEqual([])
    expect(window.getTab(plexTabId)).not.toBeNull()
    expect(window.getTab(noteTabId)).toBeNull()
  })

  it('reconciles and opens the new tab when a file kind changes on reopen', async () => {
    const noteKind = kind({
      kind: 'note',
      getOpenTab: (state: { at: string }) => ({ path: state.at }),
    })
    const deckKind = kind({
      kind: 'deck',
      getOpenTab: (state: { at: string }) => ({ path: state.at }),
    })
    const window = told([noteKind.declared, deckKind.declared])

    let fileKind: FileKind = { kind: 'note', type: 'note' }
    const openers = fileOpeners({
      fileKinds: async (paths) => new Map(paths.map((p) => [p, fileKind])),
      reconcileTab: window.reconcileTab,
    })
    openers.registerEditor('note', (path) => void window.openTabOfKind('note', path))
    openers.registerEditor('deck', (path) => void window.openTabOfKind('deck', path))

    await openers.openFile('Animals.md')
    expect(window.tabs.value).toHaveLength(1)
    expect(window.getTab(window.tabs.value[0]!.id)?.kind.kind).toBe('note')

    fileKind = { kind: 'note' as const, type: 'deck' as const }
    await openers.openFile('Animals.md')

    expect(window.tabs.value).toHaveLength(1)
    expect(window.getTab(window.tabs.value[0]!.id)?.kind.kind).toBe('deck')
    expect(noteKind.closedPaths).toEqual(['Animals.md'])
  })
})

describe('a window with nothing open', () => {
  it('has one pane named main, laid out horizontally and focused', () => {
    const window = told([])

    expect(window.layout.value.axis).toBe('horizontal')
    expect(window.layout.value.focus).toBe('main')
    expect(window.layout.value.root).toMatchObject({ kind: 'pane', id: 'main', tabs: [] })
  })

  it('calls a tab nothing where its kind gives no title', async () => {
    const bare = kind()
    delete bare.one.getTitle
    const window = told([bare.declared])

    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.tabs.value).toEqual([{ id, title: '' }])
  })
})

describe('what a tab of a kind holds, asked by kind', () => {
  it('is nothing for an identity the window does not hold', () => {
    const window = told([kind().declared])

    expect(window.getTabStateIn('missing', 'thing')).toBeNull()
    expect(window.handle.getTabState('thing', 'missing')).toBeNull()
  })

  it('is nothing where the tab is of another kind', async () => {
    const window = told([kind().declared, kind({ kind: 'other' }).declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.getTabStateIn(id, 'other')).toBeNull()
    expect(window.getTabStateIn(id, 'thing')).toEqual({ at: 'Note.md', title: 'Note.md' })
  })
})

describe('what a tab watches', () => {
  /** A kind whose tabs watch something that records when it is let go of. */
  const watching = () => {
    const stopped: string[] = []
    const one = kind({
      open: (at: string) => {
        onScopeDispose(() => stopped.push(at))
        return { at, title: at }
      },
    })
    return { stopped, one }
  }

  it('is let go of when the tab closes itself', async () => {
    const watch = watching()
    const window = told([watch.one.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.handle.closeTab(id)

    expect(watch.stopped).toEqual(['Note.md'])
  })

  it('is let go of when the tab is let go of', async () => {
    const watch = watching()
    const window = told([watch.one.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.releaseTab(id)

    expect(watch.stopped).toEqual(['Note.md'])
  })

  it('is let go of when the window goes', async () => {
    const watch = watching()
    const window = told([watch.one.declared])
    await window.openTabOfKind('thing', 'Note.md')

    window.close()

    expect(watch.stopped).toEqual(['Note.md'])
  })

  it('outlives the scope the tab was opened from', async () => {
    const watch = watching()
    const window = told([watch.one.declared])
    const parent = effectScope()

    await parent.run(() => window.openTabOfKind('thing', 'Note.md'))
    parent.stop()

    expect(watch.stopped).toEqual([])
  })

  it('is no trouble to close for an identity the window does not hold', () => {
    const window = told([kind().declared])

    expect(() => window.handle.closeTab('missing')).not.toThrow()
  })
})

describe('a tab opened beside', () => {
  it('lands in a new pane on the right, in a horizontal layout', async () => {
    const window = told([kind().declared])
    const one = await window.openTabOfKind('thing', 'One.md')

    const two = await window.handle.openTabBeside('thing', 'Two.md')

    expect(getPanes(window.layout.value.root).map((pane) => pane.tabs)).toEqual([[one], [two]])
    expect(window.layout.value.axis).toBe('horizontal')
  })

  it('draws a plugin unavailable tab beside for a kind the window was never told about', async () => {
    const window = told([kind().declared])
    await window.openTabOfKind('thing', 'One.md')

    const id = await window.handle.openTabBeside('nothing', 'Two.md')

    expect(id).toBe('nothing:Two.md')
    expect(onScreen(window.layout.value)).toContain(id)
  })
})

describe('the tab in front, with the pane in front unknown to the layout', () => {
  it('is nothing', async () => {
    const window = told([kind().declared])
    await window.openTabOfKind('thing', 'One.md')
    window.layout.value = { ...window.layout.value, focus: 'ghost' }

    expect(window.handle.front()).toBeNull()
  })
})

describe('a key struck on the window, offered in turn', () => {
  /** Tabs of a kind that records who was offered the key and takes it only where told. */
  const offering = (takes: string) => {
    const offered: string[] = []
    const one = kind({
      onKeyPress: (state: { at: string }) => {
        offered.push(state.at)
        return state.at === takes
      },
    })
    return { offered, one }
  }
  const strike = new KeyboardEvent('keydown', { key: 'a' })

  it('goes to the pane in front, then to the rest from the one the person was last in', async () => {
    const offer = offering('none')
    const window = told([offer.one.declared])
    const first = await window.openTabOfKind('thing', 'A')
    await window.handle.openTabBeside('thing', 'B')
    await window.handle.openTabBeside('thing', 'C')
    window.show(first)

    expect(window.onKeyPress(strike)).toBe(false)

    expect(offer.offered).toEqual(['A', 'C', 'B'])
  })

  it('is taken by the first tab that reads it, and by no later one', async () => {
    const offer = offering('C')
    const window = told([offer.one.declared])
    const first = await window.openTabOfKind('thing', 'A')
    await window.handle.openTabBeside('thing', 'B')
    await window.handle.openTabBeside('thing', 'C')
    window.show(first)

    expect(window.onKeyPress(strike)).toBe(true)

    expect(offer.offered).toEqual(['A', 'C'])
  })

  it('is not offered to a tab hidden behind another in its pane', async () => {
    const offer = offering('none')
    const window = told([offer.one.declared])
    await window.openTabOfKind('thing', 'Hidden')
    await window.openTabOfKind('thing', 'Shown')

    window.onKeyPress(strike)

    expect(offer.offered).toEqual(['Shown'])
  })

  it('is no trouble for a layout drawing a tab the window does not hold', () => {
    const window = told([kind().declared])
    window.layout.value = openTab(window.layout.value, 'ghost')

    expect(window.onKeyPress(strike)).toBe(false)
  })

  it('is offered to the tabs on screen while the pane in front is unknown to the layout', async () => {
    const offer = offering('A')
    const window = told([offer.one.declared])
    await window.openTabOfKind('thing', 'A')
    window.layout.value = { ...window.layout.value, focus: 'ghost' }

    expect(window.onKeyPress(strike)).toBe(true)
    expect(offer.offered).toEqual(['A'])
  })
})

describe('a tab let go of', () => {
  it('goes where its kind has nothing to finish', async () => {
    const bare = kind()
    delete bare.one.onClose
    const window = told([bare.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.releaseTab(id)).toBe(true)

    expect(window.getTab(id)).toBeNull()
  })
})

describe('reconciling tabs for a path, by what a tab stands on', () => {
  it('leaves a tab with no path alone when asked for no path', async () => {
    const bare = kind({ kind: 'bare' })
    const window = told([bare.declared])
    const id = await window.openTabOfKind('bare', 'x')

    window.reconcileTab('', 'deck')

    expect(window.getTab(id)).not.toBeNull()
    expect(bare.closedPaths).toEqual([])
  })

  it('leaves a tab of another kind alone where it stands on another path', async () => {
    const note = kind({
      kind: 'note',
      getOpenTab: (state: { at: string }) => ({ path: state.at }),
    })
    const window = told([note.declared])
    const id = await window.openTabOfKind('note', 'Other.md')

    window.reconcileTab('Cards.md', 'deck')

    expect(window.getTab(id)).not.toBeNull()
  })

  it('reads the path from the target where the kind has no open tab', async () => {
    const note = kind({
      kind: 'note',
      getTarget: (state: { at: string }) => ({ path: state.at }),
    })
    const window = told([note.declared])
    const id = await window.openTabOfKind('note', 'Cards.md')

    window.reconcileTab('Cards.md', 'deck')

    expect(window.getTab(id)).toBeNull()
  })

  it('reads the path from the target where the open tab has none', async () => {
    const note = kind({
      kind: 'note',
      getOpenTab: () => ({ path: '' }),
      getTarget: (state: { at: string }) => ({ path: state.at }),
    })
    const window = told([note.declared])
    const id = await window.openTabOfKind('note', 'Cards.md')

    window.reconcileTab('Cards.md', 'deck')

    expect(window.getTab(id)).toBeNull()
  })
})

describe('a tab opened on nothing', () => {
  it('is opened by its kind on the empty address, in front or beside', async () => {
    const thing = kind()
    const window = told([thing.declared])

    await window.openTabOfKind('thing')
    await window.handle.openTabBeside('thing')

    expect(thing.opened).toEqual(['', ''])
  })
})

describe('a window going, with kinds that say nothing about it', () => {
  it('goes without a word to a tab whose kind has neither way of going', async () => {
    const bare = kind()
    delete bare.one.onClose
    const window = told([bare.declared])
    await window.openTabOfKind('thing', 'x')

    expect(() => window.close()).not.toThrow()
  })
})

describe('reconciling tabs for a path, by what a tab names', () => {
  it('leaves a tab alone whose kind says nothing of where it stands', async () => {
    const bare = kind({ kind: 'bare' })
    const window = told([bare.declared])
    const id = await window.openTabOfKind('bare', 'x')

    expect(() => window.reconcileTab('x', 'deck')).not.toThrow()

    expect(window.getTab(id)).not.toBeNull()
  })

  it('reads the file from the target where it names no path', async () => {
    const note = kind({
      kind: 'note',
      getTarget: (state: { at: string }) => ({ file: state.at }),
    })
    const window = told([note.declared])
    const id = await window.openTabOfKind('note', 'Cards.md')

    window.reconcileTab('Cards.md', 'deck')

    expect(window.getTab(id)).toBeNull()
  })

  it('leaves a tab alone whose target names neither path nor file', async () => {
    const note = kind({ kind: 'note', getTarget: () => ({}) })
    const window = told([note.declared])
    const id = await window.openTabOfKind('note', 'Cards.md')

    window.reconcileTab('Cards.md', 'deck')

    expect(window.getTab(id)).not.toBeNull()
  })
})

describe('a tab created without being shown', () => {
  it('is opened by its kind on the empty address', async () => {
    const thing = kind()
    const window = told([thing.declared])

    const id = await window.createTab('thing')

    expect(thing.opened).toEqual([''])
    expect(onScreen(window.layout.value)).not.toContain(id)
  })
})

describe('the registration of a kind', () => {
  it('is removed by its own disposer', async () => {
    const window = useWindowTabs()
    const dispose = window.registerKind(kind().one)

    dispose()

    const id = await window.createTab('thing', 'a')
    expect(window.getTab(id)?.state).toEqual({ kind: 'thing', at: 'a' })
  })

  it('is kept when the disposer of a registration it replaced runs late', async () => {
    const window = useWindowTabs()
    const first = kind().one
    const second = kind().one
    const disposeFirst = window.registerKind(first)
    window.registerKind(second)

    disposeFirst()

    const id = await window.createTab('thing', 'a')
    expect(window.getTab(id)?.kind).toBe(second)
  })

  it('names the plugin of the kind of a tab', async () => {
    const window = useWindowTabs()
    window.registerKind({ ...kind().one, kind: 'reader:x', pluginId: 'md.numen.x' })

    const id = await window.createTab('reader:x', 'a.x')

    expect(window.getTab(id)?.kind.pluginId).toBe('md.numen.x')
  })
})

describe('a tab of a kind that is not registered', () => {
  it('opens on a restore without throwing and lists itself under the missing kind', async () => {
    const window = useWindowTabs()

    const id = await window.createTab('reader:reader-pdf', '/a.pdf')

    expect(id).toBe('reader:reader-pdf:/a.pdf')
    expect(window.tabs.value).toEqual([{ id, title: 'Plugin unavailable' }])
  })

  it('does not stop the kind being registered and opened later', async () => {
    const window = useWindowTabs()
    const missing = await window.createTab('thing', 'a')
    const thing = kind()
    window.registerKind(thing.one)

    const opened = await window.createTab('thing', 'b')

    expect(opened).not.toBe(missing)
    expect(window.getTab(opened)?.state).toEqual({ at: 'b', title: 'b' })
  })
})
