/**
 * The window keeping tabs of kinds it knows nothing about.
 *
 * Everything asked here is asked of a kind made up for the test: what the
 * window does with a kind is the whole of what a real one can count on.
 */
import { describe, expect, it } from 'vitest'
import { effectScope, onScopeDispose } from 'vue'
import { openTab, getPanes, paneById } from '@numen/shell'
import type { Workspace } from '@numen/shell'
import type { FileKind } from '@/entities/file/@x/tab'
import type { AnyTabKind, WindowHandle } from '../lib/kinds'
import { fileOpeners } from './openers'
import { useWindowTabs } from './windowTabs'

/**
 * A kind that records what it was asked to do, under the names it opened on. A
 * kind that keeps its tabs has something to finish and never lets one go.
 */
const kind = ({ isKept = false, ...options }: Partial<AnyTabKind> & { isKept?: boolean } = {}) => {
  const openedTab: string[] = []
  const closedPaths: string[] = []
  const observed: string[] = []
  const one: AnyTabKind = {
    kind: 'thing',
    open: (targetPath: string) => {
      openedTab.push(targetPath)
      return { targetPath, title: targetPath || 'a thing' }
    },
    getTitle: (state: { title: string }) => state.title,
    pane: {},
    onClose: (state: { targetPath: string }) => {
      closedPaths.push(state.targetPath)
      return !isKept
    },
    onShow: (state: { targetPath: string }) => {
      observed.push(state.targetPath)
    },
    ...options,
  }
  return { declared: () => one, one, opened: openedTab, closedPaths, inspected: observed }
}

/** A window told what kinds it draws, each of them made with what it is given. */
const notifiedCalls = (declared: readonly ((handle: WindowHandle) => AnyTabKind)[]) => {
  const window = useWindowTabs()
  window.registerKinds(declared.map((one) => one(window.handle)))
  return window
}

/** Every tab on the screen, whichever pane it is in. */
const onScreen = (layout: Workspace) => getPanes(layout.root).flatMap((pane) => pane.tabs)

describe('a tab of a kind', () => {
  it('is opened by the kind, drawn, and called what the kind calls it', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(thing.opened).toEqual(['Note.md'])
    expect(onScreen(window.layout.value)).toContain(id)
    expect(window.tabs.value).toEqual([{ id, title: 'Note.md' }])
    expect(window.getTab(id)?.state).toEqual({ targetPath: 'Note.md', title: 'Note.md' })
  })

  it('carries the word its kind gives it, and none where the kind gives none', async () => {
    const marked = kind({
      getMark: (state: { targetPath: string }) => (state.targetPath ? 'unsaved' : undefined),
    })
    const window = notifiedCalls([marked.declared])

    const one = await window.openTabOfKind('thing', 'Note.md')
    const other = await window.openTabOfKind('thing')

    expect(window.tabs.value.find((tab) => tab.id === one)?.mark).toBe('unsaved')
    expect(window.tabs.value.find((tab) => tab.id === other)).not.toHaveProperty('mark')
  })

  it('is a tab of its own each time, for a kind that takes no identity', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

    const one = await window.openTabOfKind('thing')
    const other = await window.openTabOfKind('thing')

    expect(one).not.toBe(other)
    expect(thing.opened).toHaveLength(2)
  })

  it('is the tab already open, for a kind whose identity is what it opens on', async () => {
    const thing = kind({ identity: (targetPath: string) => targetPath })
    const window = notifiedCalls([thing.declared])

    const one = await window.openTabOfKind('thing', 'Note.md')
    const again = await window.openTabOfKind('thing', 'Note.md')

    expect(again).toBe(one)
    expect(thing.opened).toEqual(['Note.md'])
  })

  it('is a tab of its own where another kind names one after the same thing', async () => {
    const thing = kind({ identity: (targetPath: string) => targetPath })
    const other = kind({ kind: 'other', identity: (targetPath: string) => targetPath })
    const window = notifiedCalls([thing.declared, other.declared])

    const one = await window.openTabOfKind('thing', 'Note.md')
    const another = await window.openTabOfKind('other', 'Note.md')

    expect(another).not.toBe(one)
    expect(window.getTab(one)?.kind.kind).toBe('thing')
    expect(window.getTab(another)?.kind.kind).toBe('other')
  })

  it('is a placeholder tab for a kind the window was never told about', async () => {
    const window = notifiedCalls([kind().declared])

    const id = await window.openTabOfKind('nothing', 'One.md')

    expect(id).toBe('nothing:One.md')
    expect(onScreen(window.layout.value)).toEqual([id])
    expect(window.getTab(id)?.kind.kind).toBe('nothing')
    expect(window.getTab(id)?.kind.getTitle?.(window.getTab(id)?.state)).toBe('One.md')
    expect(window.getTab(id)?.kind.getOpenTab?.(window.getTab(id)?.state)).toEqual({
      path: 'One.md',
    })
  })
})

describe('what a kind is given', () => {
  it('lets it open a tab of another kind, and put one it holds in front', async () => {
    const other = kind({ kind: 'other' })
    let handle: WindowHandle | null = null
    const thing = kind()
    const window = notifiedCalls([
      (provided: WindowHandle) => ((handle = provided), thing.one),
      other.declared,
    ])

    const openedTab = await handle!.openTab('other', 'Note.md')
    const mine = await window.openTabOfKind('thing')
    handle!.show(openedTab)

    expect(other.opened).toEqual(['Note.md'])
    expect(onScreen(window.layout.value)).toContain(mine)
    expect(window.getTab(openedTab)).not.toBeNull()
  })
})

describe('a tab that closes', () => {
  it('lets go of what it retained', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.releaseTab(id)).toBe(true)
    expect(thing.closedPaths).toEqual(['Note.md'])
    expect(window.getTab(id)).toBeNull()
  })

  it('says it went, for an identity the window does not hold', () => {
    const window = notifiedCalls([kind().declared])

    expect(window.releaseTab('gone')).toBe(true)
  })

  it('stays where it is while its kind has something to finish', async () => {
    const heldKind = kind({ isKept: true })
    const window = notifiedCalls([heldKind.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.releaseTab(id)).toBe(false)
    expect(window.getTab(id)?.state).toBeTruthy()
    expect(onScreen(window.layout.value)).toContain(id)
  })

  it('closes when the kind that took it says it is done', async () => {
    const heldKind = kind({ isKept: true })
    const window = notifiedCalls([heldKind.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')
    window.releaseTab(id)

    window.handle.closeTab(id)

    expect(window.getTab(id)).toBeNull()
    expect(onScreen(window.layout.value)).not.toContain(id)
  })
})

describe('a tab release from outside', () => {
  it('lets go of what it retained and arrives off the screen', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.requestClose(id)

    expect(thing.closedPaths).toEqual(['Note.md'])
    expect(window.getTab(id)).toBeNull()
    expect(onScreen(window.layout.value)).not.toContain(id)
  })

  it('stays on the screen while its kind has something to finish', async () => {
    const heldKind = kind({ isKept: true })
    const window = notifiedCalls([heldKind.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.requestClose(id)

    expect(window.getTab(id)?.state).toBeTruthy()
    expect(onScreen(window.layout.value)).toContain(id)
  })
})

describe('the tab now on screen', () => {
  it('is told, so what it holds has room to measure', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.onTabShown(id)

    expect(thing.inspected).toEqual(['Note.md'])
  })

  it('is nothing to a window that no longer holds it', () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

    window.onTabShown('gone')

    expect(thing.inspected).toEqual([])
  })
})

describe('the tab the person is looking at', () => {
  it('is the one showing in the pane in front, with its kind and what it holds', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    await window.openTabOfKind('thing', 'One.md')
    const two = await window.openTabOfKind('thing', 'Two.md')

    expect(window.handle.front()).toEqual({
      id: two,
      kind: 'thing',
      state: { targetPath: 'Two.md', title: 'Two.md' },
    })
  })

  it('is the tab of the pane in front, whichever pane reported itself last', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    const one = await window.openTabOfKind('thing', 'One.md')
    const two = await window.handle.openTabBeside('thing', 'Two.md')
    window.show(one)
    // Every pane says what it is showing when it is drawn.
    window.onTabShown(two)

    expect(window.handle.front()?.id).toBe(one)
    expect(window.handle.last('thing')?.id).toBe(two)
  })

  it('answers under no kind for a tab the window has release', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')
    window.releaseTab(id)

    expect(window.handle.front()).toEqual({ id, kind: null, state: null })
  })

  it('is nothing while the pane in front holds no tab', () => {
    const window = notifiedCalls([kind().declared])

    expect(window.handle.front()).toBeNull()
  })
})

describe('a key struck on the window', () => {
  /** A kind whose tabs take the arrows and record what they were struck with. */
  const readTabs = () => {
    const took: string[] = []
    return {
      took,
      one: kind({
        onKeyPress: (state: { targetPath: string }, event: KeyboardEvent) => {
          if (!event.key.startsWith('Arrow')) return false
          took.push(`${state.targetPath} ${event.key}`)
          return true
        },
      }),
    }
  }

  const createKeyEvent = (key: string) => new KeyboardEvent('keydown', { key })

  it('routes to the tab of the pane the person is in, and to no other', async () => {
    // Two panes are drawn at once, so a tab that listened for itself would have
    // both of them answering the one key.
    const readItem = readTabs()
    const window = notifiedCalls([readItem.one.declared])
    const one = await window.openTabOfKind('thing', 'One.epub')
    await window.handle.openTabBeside('thing', 'Two.epub')
    window.show(one)

    expect(window.onKeyPress(createKeyEvent('ArrowRight'))).toBe(true)

    expect(readItem.took).toStrictEqual(['One.epub ArrowRight'])
  })

  it('routes to the other pane once the person is in it', async () => {
    const readItem = readTabs()
    const window = notifiedCalls([readItem.one.declared])
    await window.openTabOfKind('thing', 'One.epub')
    const two = await window.handle.openTabBeside('thing', 'Two.epub')
    window.show(two)

    window.onKeyPress(createKeyEvent('ArrowLeft'))

    expect(readItem.took).toStrictEqual(['Two.epub ArrowLeft'])
  })

  it('is left alone where the tab in front does not read it', async () => {
    const readItem = readTabs()
    const window = notifiedCalls([readItem.one.declared])
    await window.openTabOfKind('thing', 'One.epub')

    expect(window.onKeyPress(createKeyEvent('k'))).toBe(false)
    expect(readItem.took).toStrictEqual([])
  })

  it('is left alone where the kind in front reads no key at all', async () => {
    const window = notifiedCalls([kind().declared])
    await window.openTabOfKind('thing', 'One.md')

    expect(window.onKeyPress(createKeyEvent('ArrowRight'))).toBe(false)
  })

  it('is left alone while the pane in front holds no tab', () => {
    const window = notifiedCalls([readTabs().one.declared])

    expect(window.onKeyPress(createKeyEvent('ArrowRight'))).toBe(false)
  })
})

describe('the window going', () => {
  it('says so to every tab, and waits for none of them', async () => {
    const thing = kind({ isKept: true })
    const window = notifiedCalls([thing.declared])
    await window.openTabOfKind('thing', 'One.md')
    await window.openTabOfKind('thing', 'Two.md')

    window.close()

    expect(thing.closedPaths).toEqual(['One.md', 'Two.md'])
  })

  it('lets a kind that has its own way of going take it, and asks no more', async () => {
    const going: string[] = []
    const thing = kind({
      onDestroy: (state: { targetPath: string }) => going.push(state.targetPath),
    })
    const window = notifiedCalls([thing.declared])
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
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const sampleKind = kind({
      kind: 'sample',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([noteKind.declared, sampleKind.declared])

    const noteTabId = await window.openTabOfKind('note', 'Shapes.md')
    expect(window.tabs.value.map((t) => t.id)).toContain(noteTabId)

    window.reconcileTab('Shapes.md', 'sample')

    expect(noteKind.closedPaths).toEqual(['Shapes.md'])
    expect(window.getTab(noteTabId)).toBeNull()
  })

  it('keeps existing tabs when the kind matches', async () => {
    const noteKind = kind({
      kind: 'note',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([noteKind.declared])

    const noteTabId = await window.openTabOfKind('note', 'Note.md')
    window.reconcileTab('Note.md', 'note')

    expect(noteKind.closedPaths).toEqual([])
    expect(window.getTab(noteTabId)).not.toBeNull()
  })

  it('preserves the tabs of a kind that declares itself a tool', async () => {
    const plexKind = kind({
      kind: 'plex',
      isTool: true,
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const noteKind = kind({
      kind: 'note',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([plexKind.declared, noteKind.declared])

    const plexTabId = await window.openTabOfKind('plex', 'Graph.md')
    const noteTabId = await window.openTabOfKind('note', 'Graph.md')

    window.reconcileTab('Graph.md', 'sample')

    expect(noteKind.closedPaths).toEqual(['Graph.md'])
    expect(plexKind.closedPaths).toEqual([])
    expect(window.getTab(plexTabId)).not.toBeNull()
    expect(window.getTab(noteTabId)).toBeNull()
  })

  it('reconciles and opens the new tab when a file kind changes on reopen', async () => {
    const noteKind = kind({
      kind: 'note',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const sampleKind = kind({
      kind: 'sample',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([noteKind.declared, sampleKind.declared])

    let fileKind: FileKind = { isNote: true, type: 'note' }
    const openers = fileOpeners({
      fileKinds: async (paths) => new Map(paths.map((p) => [p, fileKind])),
      reconcileTab: window.reconcileTab,
    })
    openers.registerEditor('note', (path) => void window.openTabOfKind('note', path))
    openers.registerEditor('sample', (path) => void window.openTabOfKind('sample', path))

    await openers.openFile('Animals.md')
    expect(window.tabs.value).toHaveLength(1)
    expect(window.getTab(window.tabs.value[0]!.id)?.kind.kind).toBe('note')

    fileKind = { isNote: true, type: 'sample' as const }
    await openers.openFile('Animals.md')

    expect(window.tabs.value).toHaveLength(1)
    expect(window.getTab(window.tabs.value[0]!.id)?.kind.kind).toBe('sample')
    expect(noteKind.closedPaths).toEqual(['Animals.md'])
  })
})

describe('a window with nothing open', () => {
  it('has one pane named main, laid out horizontally and focused', () => {
    const window = notifiedCalls([])

    expect(window.layout.value.axis).toBe('horizontal')
    expect(window.layout.value.focus).toBe('main')
    expect(window.layout.value.root).toMatchObject({ kind: 'pane', id: 'main', tabs: [] })
  })

  it('calls a tab nothing where its kind gives no title', async () => {
    const untitledKind = kind()
    delete untitledKind.one.getTitle
    const window = notifiedCalls([untitledKind.declared])

    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.tabs.value).toEqual([{ id, title: '' }])
  })
})

describe('what a tab of a kind holds, requested by kind', () => {
  it('is nothing for an identity the window does not hold', () => {
    const window = notifiedCalls([kind().declared])

    expect(window.getTabStateIn('missing', 'thing')).toBeNull()
    expect(window.handle.getTabState('thing', 'missing')).toBeNull()
  })

  it('is nothing where the tab is of another kind', async () => {
    const window = notifiedCalls([kind().declared, kind({ kind: 'other' }).declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.getTabStateIn(id, 'other')).toBeNull()
    expect(window.getTabStateIn(id, 'thing')).toEqual({ targetPath: 'Note.md', title: 'Note.md' })
  })
})

describe('what a tab watches', () => {
  /** A kind whose tabs watch something that records when it is release. */
  const watching = () => {
    const stoppedTabs: string[] = []
    const one = kind({
      open: (targetPath: string) => {
        onScopeDispose(() => stoppedTabs.push(targetPath))
        return { targetPath, title: targetPath }
      },
    })
    return { stopped: stoppedTabs, one }
  }

  it('is release when the tab closes itself', async () => {
    const watch = watching()
    const window = notifiedCalls([watch.one.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.handle.closeTab(id)

    expect(watch.stopped).toEqual(['Note.md'])
  })

  it('is release when the tab is release', async () => {
    const watch = watching()
    const window = notifiedCalls([watch.one.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    window.releaseTab(id)

    expect(watch.stopped).toEqual(['Note.md'])
  })

  it('is released when the window closes', async () => {
    const watch = watching()
    const window = notifiedCalls([watch.one.declared])
    await window.openTabOfKind('thing', 'Note.md')

    window.close()

    expect(watch.stopped).toEqual(['Note.md'])
  })

  it('outlives the scope the tab was opened from', async () => {
    const watch = watching()
    const window = notifiedCalls([watch.one.declared])
    const parent = effectScope()

    await parent.run(() => window.openTabOfKind('thing', 'Note.md'))
    parent.stop()

    expect(watch.stopped).toEqual([])
  })

  it('is no trouble to close for an identity the window does not hold', () => {
    const window = notifiedCalls([kind().declared])

    expect(() => window.handle.closeTab('missing')).not.toThrow()
  })
})

describe('a tab opened alongside', () => {
  it('lands in a new pane on the right, in a horizontal layout', async () => {
    const window = notifiedCalls([kind().declared])
    const one = await window.openTabOfKind('thing', 'One.md')

    const two = await window.handle.openTabBeside('thing', 'Two.md')

    expect(getPanes(window.layout.value.root).map((pane) => pane.tabs)).toEqual([[one], [two]])
    expect(window.layout.value.axis).toBe('horizontal')
  })

  it('draws a plugin unavailable tab alongside for a kind the window was never told about', async () => {
    const window = notifiedCalls([kind().declared])
    await window.openTabOfKind('thing', 'One.md')

    const id = await window.handle.openTabBeside('nothing', 'Two.md')

    expect(id).toBe('nothing:Two.md')
    expect(onScreen(window.layout.value)).toContain(id)
  })
})

describe('the tab in front, with the pane in front unknown to the layout', () => {
  it('is nothing', async () => {
    const window = notifiedCalls([kind().declared])
    await window.openTabOfKind('thing', 'One.md')
    window.layout.value = { ...window.layout.value, focus: 'ghost' }

    expect(window.handle.front()).toBeNull()
  })
})

describe('a key struck on the window, offered in turn', () => {
  /** Tabs of a kind that records who was offered the key and takes it only where told. */
  const offerTab = (takes: string) => {
    const proposed: string[] = []
    const one = kind({
      onKeyPress: (state: { targetPath: string }) => {
        proposed.push(state.targetPath)
        return state.targetPath === takes
      },
    })
    return { actions: proposed, one }
  }
  const strike = new KeyboardEvent('keydown', { key: 'a' })

  it('routes to the pane in front, then to the rest from the one the person was last in', async () => {
    const sampleTab = offerTab('none')
    const window = notifiedCalls([sampleTab.one.declared])
    const first = await window.openTabOfKind('thing', 'A')
    await window.handle.openTabBeside('thing', 'B')
    await window.handle.openTabBeside('thing', 'C')
    window.show(first)

    expect(window.onKeyPress(strike)).toBe(false)

    expect(sampleTab.actions).toEqual(['A', 'C', 'B'])
  })

  it('is taken by the first tab that reads it, and by no later one', async () => {
    const offer = offerTab('C')
    const window = notifiedCalls([offer.one.declared])
    const first = await window.openTabOfKind('thing', 'A')
    await window.handle.openTabBeside('thing', 'B')
    await window.handle.openTabBeside('thing', 'C')
    window.show(first)

    expect(window.onKeyPress(strike)).toBe(true)

    expect(offer.actions).toEqual(['A', 'C'])
  })

  it('is not offered to a tab hidden behind another in its pane', async () => {
    const offer = offerTab('none')
    const window = notifiedCalls([offer.one.declared])
    await window.openTabOfKind('thing', 'Hidden')
    await window.openTabOfKind('thing', 'Shown')

    window.onKeyPress(strike)

    expect(offer.actions).toEqual(['Shown'])
  })

  it('is no trouble for a layout drawing a tab the window does not hold', () => {
    const window = notifiedCalls([kind().declared])
    window.layout.value = openTab(window.layout.value, 'ghost')

    expect(window.onKeyPress(strike)).toBe(false)
  })

  it('is offered to the tabs on screen while the pane in front is unknown to the layout', async () => {
    const offer = offerTab('A')
    const window = notifiedCalls([offer.one.declared])
    await window.openTabOfKind('thing', 'A')
    window.layout.value = { ...window.layout.value, focus: 'ghost' }

    expect(window.onKeyPress(strike)).toBe(true)
    expect(offer.actions).toEqual(['A'])
  })
})

describe('a tab release', () => {
  it('closes where its kind has nothing to finish', async () => {
    const simpleKind = kind()
    delete simpleKind.one.onClose
    const window = notifiedCalls([simpleKind.declared])
    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(window.releaseTab(id)).toBe(true)

    expect(window.getTab(id)).toBeNull()
  })
})

describe('reconciling tabs for a path, by what a tab relies on', () => {
  it('leaves a tab with no path alone when requested for no path', async () => {
    const pathlessKind = kind({ kind: 'bare' })
    const window = notifiedCalls([pathlessKind.declared])
    const id = await window.openTabOfKind('bare', 'x')

    window.reconcileTab('', 'sample')

    expect(window.getTab(id)).not.toBeNull()
    expect(pathlessKind.closedPaths).toEqual([])
  })

  it('leaves a tab of another kind alone where it relies on another path', async () => {
    const note = kind({
      kind: 'note',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([note.declared])
    const id = await window.openTabOfKind('note', 'Other.md')

    window.reconcileTab('Shapes.md', 'sample')

    expect(window.getTab(id)).not.toBeNull()
  })

  it('reads the path from the target where the kind has no open tab', async () => {
    const note = kind({
      kind: 'note',
      getTarget: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([note.declared])
    const id = await window.openTabOfKind('note', 'Shapes.md')

    window.reconcileTab('Shapes.md', 'sample')

    expect(window.getTab(id)).toBeNull()
  })

  it('reads the path from the target where the open tab has none', async () => {
    const note = kind({
      kind: 'note',
      getOpenTab: () => ({ path: '' }),
      getTarget: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([note.declared])
    const id = await window.openTabOfKind('note', 'Shapes.md')

    window.reconcileTab('Shapes.md', 'sample')

    expect(window.getTab(id)).toBeNull()
  })
})

describe('a tab opened on nothing', () => {
  it('is opened by its kind on the empty address, in front or alongside', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

    await window.openTabOfKind('thing')
    await window.handle.openTabBeside('thing')

    expect(thing.opened).toEqual(['', ''])
  })
})

describe('a window going, with kinds that say nothing about it', () => {
  it('closes without a word to a tab whose kind has neither way of going', async () => {
    const closingKind = kind()
    delete closingKind.one.onClose
    const window = notifiedCalls([closingKind.declared])
    await window.openTabOfKind('thing', 'x')

    expect(() => window.close()).not.toThrow()
  })
})

describe('reconciling tabs for a path, by what a tab names', () => {
  it('leaves a tab alone whose kind says nothing of at its current position', async () => {
    const pathlessKind = kind({ kind: 'bare' })
    const window = notifiedCalls([pathlessKind.declared])
    const id = await window.openTabOfKind('bare', 'x')

    expect(() => window.reconcileTab('x', 'sample')).not.toThrow()

    expect(window.getTab(id)).not.toBeNull()
  })

  it('reads the file from the target where it names no path', async () => {
    const note = kind({
      kind: 'note',
      getTarget: (state: { targetPath: string }) => ({ file: state.targetPath }),
    })
    const window = notifiedCalls([note.declared])
    const id = await window.openTabOfKind('note', 'Shapes.md')

    window.reconcileTab('Shapes.md', 'sample')

    expect(window.getTab(id)).toBeNull()
  })

  it('leaves a tab alone whose target names neither path nor file', async () => {
    const note = kind({ kind: 'note', getTarget: () => ({}) })
    const window = notifiedCalls([note.declared])
    const id = await window.openTabOfKind('note', 'Shapes.md')

    window.reconcileTab('Shapes.md', 'sample')

    expect(window.getTab(id)).not.toBeNull()
  })
})

describe('a tab created without being shown', () => {
  it('is opened by its kind on the empty address', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

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
    expect(window.getTab(id)?.state).toEqual({ kind: 'thing', targetPath: 'a' })
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

    const id = await window.createTab('reader:viewer', '/a.bin')

    expect(id).toBe('reader:viewer:/a.bin')
    expect(window.tabs.value).toEqual([{ id, title: 'a.bin' }])
  })

  it('does not stop the kind being registered and opened later', async () => {
    const window = useWindowTabs()
    const missing = await window.createTab('thing', 'a')
    const thing = kind()
    window.registerKind(thing.one)

    const openedTab = await window.createTab('thing', 'b')

    expect(openedTab).not.toBe(missing)
    expect(window.getTab(openedTab)?.state).toEqual({ targetPath: 'b', title: 'b' })
  })
})

describe('the identity a kind is told as its tab opens', () => {
  it('is the one the window files the tab under', async () => {
    const provided: string[] = []
    const thing = kind({
      open: (targetPath: string, id: string) => {
        provided.push(id)
        return { targetPath, title: targetPath }
      },
    })
    const window = useWindowTabs()
    window.registerKinds([thing.one])

    const id = await window.openTabOfKind('thing', 'Note.md')

    expect(provided).toEqual([id])
  })
})

describe('the kinds the window holds', () => {
  it('are the ones registered, and a kind removed is no longer listed', () => {
    const window = useWindowTabs()
    const release = window.registerKind(kind().one)

    expect(window.hasKind('thing')).toBe(true)
    expect(window.hasKind('other')).toBe(false)
    release()
    expect(window.hasKind('thing')).toBe(false)
  })
  it('finds the first registered that answers, and none where none does', () => {
    const window = useWindowTabs()
    window.registerKind(kind().one)
    window.registerKind(kind({ kind: 'chat', receiveText: () => {} }).one)

    expect(window.findKind((one) => one.receiveText !== undefined)?.kind).toBe('chat')
    expect(window.findKind((one) => one.isTool === true)).toBeUndefined()
  })
})

describe('a window opened with nothing to restore', () => {
  const placedTab = (name: string, pane: 'main' | 'aside', order: number) =>
    kind({ kind: name, placement: { pane, order } }).one

  it('opens one tab of every placed kind, in the pane and the order each declares', async () => {
    const window = useWindowTabs()
    window.registerKinds([
      placedTab('files', 'aside', 1),
      kind({ kind: 'unplaced' }).one,
      placedTab('agent', 'aside', 0),
      placedTab('plex', 'main', 0),
    ])

    await window.openPlacedTabs(() => {
      throw new Error('no placed tab fails here')
    })

    const panes = getPanes(window.layout.value.root)
    expect(panes.map((one) => [one.id, one.tabs.map((id) => id.split(':')[0])])).toStrictEqual([
      ['main', ['plex']],
      ['aside', ['agent', 'files']],
    ])
    expect(window.tabs.value.map((tab) => tab.id.split(':')[0])).toStrictEqual([
      'plex',
      'agent',
      'files',
    ])
  })

  it('keeps a tab the person opened before the placed tabs were laid out, in front', async () => {
    const window = useWindowTabs()
    window.registerKinds([placedTab('plex', 'main', 0), kind({ kind: 'settings' }).one])
    const early = await window.openTabOfKind('settings')

    await window.openPlacedTabs(() => {
      throw new Error('no placed tab fails here')
    })

    const [main] = getPanes(window.layout.value.root)
    expect(main?.tabs.map((id) => id.split(':')[0])).toStrictEqual(['plex', 'settings'])
    expect(main?.active).toBe(early)
  })

  it('gives the whole window to the aside kinds where no kind is placed in the main pane', async () => {
    const window = useWindowTabs()
    window.registerKinds([placedTab('agent', 'aside', 0)])

    await window.openPlacedTabs(() => {
      throw new Error('no placed tab fails here')
    })

    expect(getPanes(window.layout.value.root).map((one) => one.id)).toStrictEqual(['main'])
    expect(onScreen(window.layout.value).map((id) => id.split(':')[0])).toStrictEqual(['agent'])
  })
})

describe('a placed kind whose tab fails to open', () => {
  it('is told, left out of the layout, and lets go of what it started', async () => {
    const failure = new Error('the plugin is gone')
    const stoppedTabs: string[] = []
    const window = useWindowTabs()
    window.registerKinds([
      kind({ kind: 'agent', placement: { pane: 'aside', order: 0 } }).one,
      kind({
        kind: 'plex',
        placement: { pane: 'main', order: 0 },
        open: () => {
          onScopeDispose(() => stoppedTabs.push('plex'))
          throw failure
        },
      }).one,
    ])
    const notifiedCalls: [string, unknown][] = []

    await window.openPlacedTabs((name, error) => notifiedCalls.push([name, error]))

    expect(notifiedCalls).toStrictEqual([['plex', failure]])
    expect(stoppedTabs).toStrictEqual(['plex'])
    expect(onScreen(window.layout.value).map((id) => id.split(':')[0])).toStrictEqual(['agent'])
    expect(window.tabs.value.map((tab) => tab.id.split(':')[0])).toStrictEqual(['agent'])
  })
})

describe('a tab of a kind moved to a path', () => {
  /** A kind whose tabs stand on a path, and move when asked. */
  const walker = (name = 'walker') =>
    kind({
      kind: name,
      open: (targetPath: string) => ({ targetPath, title: targetPath }),
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
      navigate: (state: { targetPath: string; title: string }, path: string) => {
        state.targetPath = path
      },
    })

  it('is the one the person was last in, put in front', async () => {
    const one = walker()
    const window = useWindowTabs()
    window.registerKinds([one.one, kind().one])
    const first = await window.openTabOfKind('walker', 'One.md')
    const second = await window.openTabOfKind('walker', 'Two.md')
    window.onTabShown(first)
    await window.openTabOfKind('thing', 'Else.md')

    const moved = await window.navigateTab('walker', 'Wanted.md')

    expect(moved).toBe(first)
    expect(window.getTab(first)?.state).toMatchObject({ targetPath: 'Wanted.md' })
    expect(window.getTab(second)?.state).toMatchObject({ targetPath: 'Two.md' })
    expect(paneById(window.layout.value.root, window.layout.value.focus)?.active).toBe(first)
  })

  it('is a new one opened there where the window holds none', async () => {
    const window = useWindowTabs()
    window.registerKinds([walker().one])

    const id = await window.navigateTab('walker', 'Wanted.md')

    expect(window.getTab(id)?.state).toMatchObject({ targetPath: 'Wanted.md' })
  })
})

describe('the tabs standing on a path that went', () => {
  it('move to another, every one of a kind that navigates, and no other', async () => {
    const window = useWindowTabs()
    const fixed = kind({
      kind: 'fixed',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    window.registerKinds([
      kind({
        kind: 'walker',
        getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
        navigate: (state: { targetPath: string }, path: string) => {
          state.targetPath = path
        },
      }).one,
      fixed.one,
    ])
    const removed = await window.openTabOfKind('walker', 'Gone.md')
    const survivor = await window.openTabOfKind('walker', 'Elsewhere.md')
    const stationary = await window.openTabOfKind('fixed', 'Gone.md')

    await window.navigateTabsAt('Gone.md', 'Root.md')

    expect(window.getTab(removed)?.state).toMatchObject({ targetPath: 'Root.md' })
    expect(window.getTab(survivor)?.state).toMatchObject({ targetPath: 'Elsewhere.md' })
    expect(window.getTab(stationary)?.state).toMatchObject({ targetPath: 'Gone.md' })
  })
})

describe('text put to the tabs of a kind', () => {
  /** A kind whose tabs keep the text they were handed. */
  const listener = () =>
    kind({
      kind: 'listener',
      open: () => ({ received: [] as string[], title: '' }),
      receiveText: (state: { received: string[] }, text: string) => {
        state.received.push(text)
      },
    })

  it('switches to the tab the person was last in, which is put in front', async () => {
    const window = useWindowTabs()
    window.registerKinds([listener().one])
    const first = await window.openTabOfKind('listener')
    const second = await window.openTabOfKind('listener')
    window.onTabShown(first)
    window.onTabShown(second)

    const dispatched = await window.sendText('listener', 'what is this?')

    expect(dispatched).toBe(second)
    expect(window.getTab(first)?.state).toMatchObject({ received: [] })
    expect(window.getTab(second)?.state).toMatchObject({ received: ['what is this?'] })
    expect(paneById(window.layout.value.root, window.layout.value.focus)?.active).toBe(second)
  })

  it('opens a tab to take it where the window holds none', async () => {
    const window = useWindowTabs()
    window.registerKinds([listener().one])

    const id = await window.sendText('listener', 'what is this?')

    expect(window.getTab(id)?.state).toMatchObject({ received: ['what is this?'] })
  })

  it('is left unheard by a tab of a kind that takes no text', async () => {
    const window = useWindowTabs()
    window.registerKinds([kind().one])
    const id = await window.openTabOfKind('thing')

    await expect(window.sendText('thing', 'hello')).resolves.toBe(id)
  })
})

describe('a file made as the kind that edits it', () => {
  it('is requested of that kind, under the folder and the name given', async () => {
    const createdItem: string[][] = []
    const window = notifiedCalls([
      kind({
        createFile: async (folder, name) => {
          createdItem.push([folder, name])
          return { ok: true, value: { path: `${folder}/${name}.md` } }
        },
      }).declared,
    ])

    expect(await window.createFileOfKind('thing', 'dir', 'Name')).toStrictEqual({
      ok: true,
      value: { path: 'dir/Name.md' },
    })
    expect(createdItem).toStrictEqual([['dir', 'Name']])
  })

  it('is requested of nothing where the kind makes no file or is not registered', () => {
    const window = notifiedCalls([kind().declared])

    expect(window.createFileOfKind('thing', 'dir', 'Name')).toBeUndefined()
    expect(window.createFileOfKind('nothing', 'dir', 'Name')).toBeUndefined()
  })
})

/** A kind of a plugin that keeps a page in each tab and saves it. */
const pagedKind = (pluginId = 'viewer.plugin'): AnyTabKind => ({
  kind: 'paged',
  pluginId,
  open: (targetPath: string, _id: string, saved?: unknown) => ({
    targetPath,
    page: (saved as { page?: number } | undefined)?.page ?? 1,
  }),
  getState: (state: { page: number }) => ({ page: state.page }),
  getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
  identity: (targetPath: string) => targetPath,
  pane: {},
})

describe('the tabs the core kept', () => {
  it('open again under their ids with the state their kind saved, the front one in front', () => {
    const window = useWindowTabs()
    window.registerKind(pagedKind())

    window.restoreTabs(
      [
        { id: 'paged:a.bin', kind: 'paged', path: 'a.bin', saved: { page: 4 } },
        { id: 'paged:b.bin', kind: 'paged', path: 'b.bin' },
      ],
      'paged:a.bin',
    )

    expect(window.getTabStateIn('paged:a.bin', 'paged')).toMatchObject({ page: 4 })
    expect(window.getTabStateIn('paged:b.bin', 'paged')).toMatchObject({ page: 1 })
    expect(window.handle.front()?.id).toBe('paged:a.bin')
  })

  it('open as placeholders that keep the saved state where no plugin registers the kind', () => {
    const window = useWindowTabs()

    window.restoreTabs(
      [{ id: 'paged:a.bin', kind: 'paged', path: 'a.bin', saved: { page: 4 } }],
      '',
    )

    const one = window.getTab('paged:a.bin')
    expect(one?.kind.getState?.(one.state)).toStrictEqual({ page: 4 })
    expect(window.tabs.value[0]?.title).toBe('a.bin')
  })
})

describe('the tabs of a plugin that stops', () => {
  it('turn into placeholders that keep the state, and back once the kind is registered again', () => {
    const window = useWindowTabs()
    const unregister = window.registerKind(pagedKind())
    window.restoreTabs(
      [{ id: 'paged:a.bin', kind: 'paged', path: 'a.bin', saved: { page: 4 } }],
      '',
    )

    window.closeTabsOfPlugin('viewer.plugin')
    unregister()
    const placeholder = window.getTab('paged:a.bin')
    expect(placeholder?.kind.pluginId).toBeUndefined()
    expect(placeholder?.kind.getState?.(placeholder.state)).toStrictEqual({ page: 4 })

    window.registerKind(pagedKind())
    expect(window.getTabStateIn('paged:a.bin', 'paged')).toMatchObject({ page: 4 })
  })

  it('leaves the tabs of other plugins as they are', () => {
    const window = useWindowTabs()
    window.registerKind(pagedKind('other.plugin'))
    window.restoreTabs([{ id: 'paged:a.bin', kind: 'paged', path: 'a.bin' }], '')

    window.closeTabsOfPlugin('viewer.plugin')

    expect(window.getTab('paged:a.bin')?.kind.pluginId).toBe('other.plugin')
  })
})
