/**
 * The window managing tabs of tab kinds it has no knowledge of.
 *
 * The tests use a kind created for the test: the window's behavior toward a
 * kind is all a real kind can rely on.
 */
import { describe, expect, it, vi } from 'vitest'
import { effectScope, onScopeDispose } from 'vue'
import { openTab, getPanes, paneById } from '@numen/workbench'
import type { Workspace } from '@numen/workbench'
import type { FileKind } from '@/shared/model/fileTypes'
import type { AnyTabKind, WindowHandle } from '../lib/kinds'
import { fileOpeners } from './openers'
import { createTestTabs, NO_TAB_KIND_PLUGINS } from '@/testing/tabs'

function rethrow(error: unknown): never {
  throw error
}

type Hooks = NonNullable<AnyTabKind['display']> &
  NonNullable<AnyTabKind['events']> &
  NonNullable<AnyTabKind['address']>
type KindOptions = Partial<
  Pick<AnyTabKind, 'kind' | 'pluginId' | 'open' | 'pane' | 'getTitle'> & Hooks
>

const PART_OF_HOOK: Readonly<Record<keyof Hooks, 'display' | 'events' | 'address'>> = {
  onShow: 'display',
  getMark: 'display',
  icon: 'display',
  placement: 'display',
  isTool: 'display',
  onClose: 'events',
  onDestroy: 'events',
  flushPending: 'events',
  receiveText: 'events',
  onKeyPress: 'events',
  isAsking: 'events',
  getState: 'address',
  identity: 'address',
  navigate: 'address',
  getTarget: 'address',
  getOpenTab: 'address',
  createFile: 'address',
}

const groupHooks = (hooks: Partial<Hooks>) => {
  const parts: Record<string, Record<string, unknown>> = { display: {}, events: {}, address: {} }
  for (const [name, hook] of Object.entries(hooks)) {
    parts[PART_OF_HOOK[name as keyof Hooks]]![name] = hook
  }
  return parts as unknown as Pick<AnyTabKind, 'display' | 'events' | 'address'>
}

const kind = ({ isKept = false, ...options }: KindOptions & { isKept?: boolean } = {}) => {
  const openedTab: string[] = []
  const closedPaths: string[] = []
  const observed: string[] = []
  const { kind: kindId, pluginId, open, pane, getTitle, ...hooks } = options
  const one: AnyTabKind = {
    kind: kindId ?? 'thing',
    ...(pluginId === undefined ? {} : { pluginId }),
    open:
      open ??
      ((targetPath: string) => {
        openedTab.push(targetPath)
        return { targetPath, title: targetPath || 'a thing' }
      }),
    pane: pane ?? {},
    getTitle: getTitle ?? ((state: { title: string }) => state.title),
    ...groupHooks({
      onClose: (state: { targetPath: string }) => {
        closedPaths.push(state.targetPath)
        return !isKept
      },
      onShow: (state: { targetPath: string }) => {
        observed.push(state.targetPath)
      },
      ...hooks,
    }),
  }
  return { declared: () => one, one, opened: openedTab, closedPaths, inspected: observed }
}

const notifiedCalls = (declared: readonly ((handle: WindowHandle) => AnyTabKind)[]) => {
  const window = createTestTabs()
  window.groups.registerKinds(declared.map((one) => one(window.groups.handle)))
  return window
}

const onScreen = (layout: Workspace) => getPanes(layout.root).flatMap((pane) => pane.tabs)

describe('a tab of a kind', () => {
  it('is opened by the kind, drawn, and called what the kind calls it', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    expect(thing.opened).toEqual(['Note.md'])
    expect(onScreen(window.panes.layout.value)).toContain(id)
    expect(window.tabs.tabs.value).toEqual([{ id, title: 'Note.md' }])
    expect(window.tabs.getTab(id)?.state).toEqual({ targetPath: 'Note.md', title: 'Note.md' })
  })

  it('carries the word its kind gives it, and none where the kind gives none', async () => {
    const marked = kind({
      getMark: (state: { targetPath: string }) => (state.targetPath ? 'unsaved' : undefined),
    })
    const window = notifiedCalls([marked.declared])

    const one = await window.tabs.openTabOfKind('thing', 'Note.md')
    const other = await window.tabs.openTabOfKind('thing')

    expect(window.tabs.tabs.value.find((tab) => tab.id === one)?.mark).toBe('unsaved')
    expect(window.tabs.tabs.value.find((tab) => tab.id === other)).not.toHaveProperty('mark')
  })

  it('is a tab of its own each time, for a kind that takes no identity', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

    const one = await window.tabs.openTabOfKind('thing')
    const other = await window.tabs.openTabOfKind('thing')

    expect(one).not.toBe(other)
    expect(thing.opened).toHaveLength(2)
  })

  it('is the tab already open, for a kind whose identity is what it opens on', async () => {
    const thing = kind({
      identity: (targetPath: string) => targetPath,
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
      navigate: (state: { targetPath: string }, path: string) => {
        state.targetPath = path
      },
    })
    const other = kind({
      kind: 'other',
      identity: (targetPath: string) => targetPath,
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const singleton = kind({
      kind: 'singleton',
      identity: () => 'only',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([thing.declared, other.declared, singleton.declared])

    await window.tabs.openTabOfKind('other', 'Note.md')
    await window.tabs.openTabOfKind('thing', 'Other.md')
    const one = await window.tabs.openTabOfKind('thing', 'Note.md')
    const again = await window.tabs.openTabOfKind('thing', 'Note.md')
    expect(again).toBe(one)
    expect(thing.opened).toEqual(['Other.md', 'Note.md'])

    // After navigating `one` ('thing:Note.md') to 'Renamed.md', opening 'Renamed.md' finds `one` via findOpenTabAt
    const moved: string[] = []
    await window.tabs.navigateTabsAt('Note.md', 'Renamed.md', (id) => moved.push(id))
    expect(moved).toEqual([one])
    expect(await window.tabs.openTabOfKind('thing', 'Renamed.md')).toBe(one)

    // Opening 'Note.md' again when 'thing:Note.md' points to 'Renamed.md' allocates a new id
    const secondId = await window.tabs.openTabOfKind('thing', 'Note.md')
    expect(secondId).not.toBe(one)

    // Singleton with empty targetPath reuses candidateId even when existingPath is non-empty
    const singleFirst = await window.tabs.openTabOfKind('singleton', 'First.md')
    expect(await window.tabs.openTabOfKind('singleton', '')).toBe(singleFirst)
  })

  it('is a tab of its own where another kind names one after the same thing', async () => {
    const thing = kind({ identity: (targetPath: string) => targetPath })
    const other = kind({ kind: 'other', identity: (targetPath: string) => targetPath })
    const window = notifiedCalls([thing.declared, other.declared])

    const one = await window.tabs.openTabOfKind('thing', 'Note.md')
    const another = await window.tabs.openTabOfKind('other', 'Note.md')

    expect(another).not.toBe(one)
    expect(window.tabs.getTab(one)?.kind.kind).toBe('thing')
    expect(window.tabs.getTab(another)?.kind.kind).toBe('other')
  })

  it('is a placeholder tab for a kind the window was never notified about', async () => {
    const window = notifiedCalls([kind().declared])

    const id = await window.tabs.openTabOfKind('nothing', 'One.md')

    expect(id).toBe('nothing:One.md')
    expect(onScreen(window.panes.layout.value)).toEqual([id])
    expect(window.tabs.getTab(id)?.kind.kind).toBe('nothing')
    expect(window.tabs.getTab(id)?.kind.getTitle?.(window.tabs.getTab(id)!.state)).toBe('One.md')
    expect(
      window.tabs.getTab(id)?.kind.address?.getOpenTab?.(window.tabs.getTab(id)!.state),
    ).toEqual({
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
    const mine = await window.tabs.openTabOfKind('thing')
    expect(window.groups.handle.front()?.id).toBe(mine)
    handle!.show(openedTab)

    expect(window.groups.handle.front()?.id).toBe(openedTab)
    expect(other.opened).toEqual(['Note.md'])
    expect(onScreen(window.panes.layout.value)).toContain(mine)
    expect(window.tabs.getTab(openedTab)).not.toBeNull()
    expect(handle!.each('other')).toStrictEqual([
      { id: openedTab, state: { targetPath: 'Note.md', title: 'Note.md' } },
    ])
  })
})

describe('a tab that closes', () => {
  it('releases what it retained', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    expect(window.panes.releaseTab(id)).toBe(true)
    expect(thing.closedPaths).toEqual(['Note.md'])
    expect(window.tabs.getTab(id)).toBeNull()
  })

  it('says it went, for an identity the window does not hold', () => {
    const window = notifiedCalls([kind().declared])

    expect(window.panes.releaseTab('gone')).toBe(true)
  })

  it('keeps its position while its kind has something to finish', async () => {
    const pendingKind = kind({ isKept: true })
    const window = notifiedCalls([pendingKind.declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    expect(window.panes.releaseTab(id)).toBe(false)
    expect(window.tabs.getTab(id)?.state).toBeTruthy()
    expect(onScreen(window.panes.layout.value)).toContain(id)
  })

  it('closes when the kind that took it says it is done', async () => {
    const pendingKind = kind({ isKept: true })
    const window = notifiedCalls([pendingKind.declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')
    window.panes.releaseTab(id)

    window.groups.handle.closeTab(id)

    expect(window.tabs.getTab(id)).toBeNull()
    expect(onScreen(window.panes.layout.value)).not.toContain(id)
  })
})

describe('a tab release from outside', () => {
  it('releases what it retained and arrives off the screen', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    window.panes.requestClose(id)

    expect(thing.closedPaths).toEqual(['Note.md'])
    expect(window.tabs.getTab(id)).toBeNull()
    expect(onScreen(window.panes.layout.value)).not.toContain(id)
  })

  it('stays on the screen while its kind has something to finish', async () => {
    const pendingKind = kind({ isKept: true })
    const window = notifiedCalls([pendingKind.declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    window.panes.requestClose(id)

    expect(window.tabs.getTab(id)?.state).toBeTruthy()
    expect(onScreen(window.panes.layout.value)).toContain(id)
  })
})

describe('the tab now on screen', () => {
  it('is notified, so its contents have room to measure', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    window.panes.onTabShown(id)

    expect(thing.inspected).toEqual(['Note.md'])
  })

  it('is nothing to a window that no longer holds it', () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

    window.panes.onTabShown('gone')

    expect(thing.inspected).toEqual([])
  })
})

describe('the tab the person is looking at', () => {
  it('is the one showing in the pane in front, with its kind and its contents', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    await window.tabs.openTabOfKind('thing', 'One.md')
    const two = await window.tabs.openTabOfKind('thing', 'Two.md')

    expect(window.groups.handle.front()).toEqual({
      id: two,
      kind: 'thing',
      state: { targetPath: 'Two.md', title: 'Two.md' },
    })
  })

  it('is the tab of the pane in front, whichever pane reported itself last', async () => {
    const thing = kind()
    const other = kind({ kind: 'other' })
    const window = notifiedCalls([thing.declared, other.declared])
    const zero = await window.tabs.openTabOfKind('thing', 'Zero.md')
    const one = await window.tabs.openTabOfKind('thing', 'One.md')
    const two = await window.groups.handle.openTabAdjacent('thing', 'Two.md')
    await window.tabs.openTabOfKind('other', 'Other.md')
    window.panes.show(one)
    window.panes.onTabShown(zero)
    // Every pane reports the tab it is showing when it renders.
    window.panes.onTabShown(two)

    expect(window.groups.handle.front()?.id).toBe(one)
    expect(window.groups.handle.last('thing')?.id).toBe(two)
    expect(window.groups.handle.each('thing').map((t) => t.id)).toStrictEqual([one, zero, two])
  })

  it('answers under no kind for a tab the window has release', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')
    window.panes.releaseTab(id)

    expect(window.groups.handle.front()).toEqual({ id, kind: null, state: null })
  })

  it('is nothing while the pane in front holds no tab', () => {
    const window = notifiedCalls([kind().declared])

    expect(window.groups.handle.front()).toBeNull()
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
    // Two panes render at once, so a tab that listened on its own would cause both
    // to respond to the same key.
    const readItem = readTabs()
    const window = notifiedCalls([readItem.one.declared])
    const one = await window.tabs.openTabOfKind('thing', 'One.epub')
    await window.groups.handle.openTabAdjacent('thing', 'Two.epub')
    window.panes.show(one)

    expect(window.panes.onKeyPress(createKeyEvent('ArrowRight'))).toBe(true)

    expect(readItem.took).toStrictEqual(['One.epub ArrowRight'])
  })

  it('routes to the other pane once the person is in it', async () => {
    const readItem = readTabs()
    const window = notifiedCalls([readItem.one.declared])
    await window.tabs.openTabOfKind('thing', 'One.epub')
    const two = await window.groups.handle.openTabAdjacent('thing', 'Two.epub')
    window.panes.show(two)

    window.panes.onKeyPress(createKeyEvent('ArrowLeft'))

    expect(readItem.took).toStrictEqual(['Two.epub ArrowLeft'])
  })

  it('is left alone where the tab in front does not read it', async () => {
    const readItem = readTabs()
    const window = notifiedCalls([readItem.one.declared])
    await window.tabs.openTabOfKind('thing', 'One.epub')

    expect(window.panes.onKeyPress(createKeyEvent('k'))).toBe(false)
    expect(readItem.took).toStrictEqual([])
  })

  it('is left alone where the kind in front reads no key at all', async () => {
    const window = notifiedCalls([kind().declared])
    await window.tabs.openTabOfKind('thing', 'One.md')

    expect(window.panes.onKeyPress(createKeyEvent('ArrowRight'))).toBe(false)
  })

  it('is left alone while the pane in front holds no tab', () => {
    const window = notifiedCalls([readTabs().one.declared])

    expect(window.panes.onKeyPress(createKeyEvent('ArrowRight'))).toBe(false)
  })
})

describe('the window going', () => {
  it('says so to every tab, and waits for none of them', async () => {
    const thing = kind({ isKept: true })
    const window = notifiedCalls([thing.declared])
    await window.tabs.openTabOfKind('thing', 'One.md')
    await window.tabs.openTabOfKind('thing', 'Two.md')

    window.restored.close()

    expect(thing.closedPaths).toEqual(['One.md', 'Two.md'])
  })

  it('lets a kind that has its own way of going take it, and asks no more', async () => {
    const going: string[] = []
    const thing = kind({
      onDestroy: (state: { targetPath: string }) => going.push(state.targetPath),
    })
    const window = notifiedCalls([thing.declared])
    await window.tabs.openTabOfKind('thing', 'One.md')

    window.restored.close()

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

    const noteTabId = await window.tabs.openTabOfKind('note', 'Shapes.md')
    expect(window.tabs.tabs.value.map((t) => t.id)).toContain(noteTabId)

    window.panes.reconcileTab('Shapes.md', 'sample')

    expect(noteKind.closedPaths).toEqual(['Shapes.md'])
    expect(window.tabs.getTab(noteTabId)).toBeNull()
  })

  it('keeps existing tabs when the kind matches', async () => {
    const noteKind = kind({
      kind: 'note',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([noteKind.declared])

    const noteTabId = await window.tabs.openTabOfKind('note', 'Note.md')
    window.panes.reconcileTab('Note.md', 'note')

    expect(noteKind.closedPaths).toEqual([])
    expect(window.tabs.getTab(noteTabId)).not.toBeNull()
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

    const plexTabId = await window.tabs.openTabOfKind('plex', 'Graph.md')
    const noteTabId = await window.tabs.openTabOfKind('note', 'Graph.md')

    window.panes.reconcileTab('Graph.md', 'sample')

    expect(noteKind.closedPaths).toEqual(['Graph.md'])
    expect(plexKind.closedPaths).toEqual([])
    expect(window.tabs.getTab(plexTabId)).not.toBeNull()
    expect(window.tabs.getTab(noteTabId)).toBeNull()
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

    let fileKind: FileKind = { isNote: true, type: 'note', tabKind: 'note' }
    const openers = fileOpeners({
      fileKinds: async (paths) => new Map(paths.map((p) => [p, fileKind])),
      reconcileTab: window.panes.reconcileTab,
    })
    openers.registerEditor('note', (path) => {
      window.tabs.openTabOfKind('note', path).catch(rethrow)
    })
    openers.registerEditor('sample', (path) => {
      window.tabs.openTabOfKind('sample', path).catch(rethrow)
    })

    await openers.openFile('Animals.md')
    expect(window.tabs.tabs.value).toHaveLength(1)
    expect(window.tabs.getTab(window.tabs.tabs.value[0]!.id)?.kind.kind).toBe('note')

    fileKind = { isNote: true, type: 'sample', tabKind: 'sample' }
    await openers.openFile('Animals.md')

    expect(window.tabs.tabs.value).toHaveLength(1)
    expect(window.tabs.getTab(window.tabs.tabs.value[0]!.id)?.kind.kind).toBe('sample')
    expect(noteKind.closedPaths).toEqual(['Animals.md'])
  })
})

describe('a window with nothing open', () => {
  it('has one pane named main, laid out horizontally and focused', () => {
    const window = notifiedCalls([])

    expect(window.panes.layout.value.axis).toBe('horizontal')
    expect(window.panes.layout.value.focus).toBe('main')
    expect(window.panes.layout.value.root).toMatchObject({ kind: 'pane', id: 'main', tabs: [] })
  })

  it('calls a tab nothing where its kind gives no title', async () => {
    const untitledKind = kind()
    delete untitledKind.one.getTitle
    const window = notifiedCalls([untitledKind.declared])

    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    expect(window.tabs.tabs.value).toEqual([{ id, title: '' }])
  })
})

describe('what a tab of a kind holds, requested by kind', () => {
  it('is nothing for an identity the window does not hold', () => {
    const window = notifiedCalls([kind().declared])

    expect(window.tabs.getTabStateIn('missing', 'thing')).toBeNull()
    expect(window.groups.handle.getTabState('thing', 'missing')).toBeNull()
  })

  it('is nothing where the tab is of another kind', async () => {
    const window = notifiedCalls([kind().declared, kind({ kind: 'other' }).declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    expect(window.tabs.getTabStateIn(id, 'other')).toBeNull()
    expect(window.tabs.getTabStateIn(id, 'thing')).toEqual({
      targetPath: 'Note.md',
      title: 'Note.md',
    })
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
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    window.groups.handle.closeTab(id)

    expect(watch.stopped).toEqual(['Note.md'])
  })

  it('is release when the tab is release', async () => {
    const watch = watching()
    const window = notifiedCalls([watch.one.declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    window.panes.releaseTab(id)

    expect(watch.stopped).toEqual(['Note.md'])
  })

  it('is released when the window closes', async () => {
    const watch = watching()
    const window = notifiedCalls([watch.one.declared])
    await window.tabs.openTabOfKind('thing', 'Note.md')

    window.restored.close()

    expect(watch.stopped).toEqual(['Note.md'])
  })

  it('outlives the scope the tab was opened from', async () => {
    const watch = watching()
    const window = notifiedCalls([watch.one.declared])
    const parent = effectScope()

    await parent.run(() => window.tabs.openTabOfKind('thing', 'Note.md'))
    parent.stop()

    expect(watch.stopped).toEqual([])
  })

  it('is no trouble to close for an identity the window does not hold', () => {
    const window = notifiedCalls([kind().declared])

    expect(() => window.groups.handle.closeTab('missing')).not.toThrow()
  })
})

describe('a tab opened alongside', () => {
  it('lands in a new pane on the right, in a horizontal layout', async () => {
    const window = notifiedCalls([kind().declared])
    const one = await window.tabs.openTabOfKind('thing', 'One.md')

    const two = await window.groups.handle.openTabAdjacent('thing', 'Two.md')

    expect(getPanes(window.panes.layout.value.root).map((pane) => pane.tabs)).toEqual([
      [one],
      [two],
    ])
    expect(window.panes.layout.value.axis).toBe('horizontal')
  })

  it('draws a plugin unavailable tab alongside for a kind the window was never notified about', async () => {
    const window = notifiedCalls([kind().declared])
    await window.tabs.openTabOfKind('thing', 'One.md')

    const id = await window.groups.handle.openTabAdjacent('nothing', 'Two.md')

    expect(id).toBe('nothing:Two.md')
    expect(onScreen(window.panes.layout.value)).toContain(id)
  })
})

describe('the tab in front, with the pane in front unknown to the layout', () => {
  it('is nothing', async () => {
    const window = notifiedCalls([kind().declared])
    await window.tabs.openTabOfKind('thing', 'One.md')
    window.panes.layout.value = { ...window.panes.layout.value, focus: 'ghost' }

    expect(window.groups.handle.front()).toBeNull()
  })
})

describe('a key struck on the window, offered in turn', () => {
  /**
   * Tabs of a kind that records which of them were offered the key and accept it
   * only where configured.
   */
  const createKeyRecorderTab = (takes: string) => {
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
    const sampleTab = createKeyRecorderTab('none')
    const window = notifiedCalls([sampleTab.one.declared])
    const first = await window.tabs.openTabOfKind('thing', 'A')
    await window.groups.handle.openTabAdjacent('thing', 'B')
    await window.groups.handle.openTabAdjacent('thing', 'C')
    window.panes.show(first)

    expect(window.panes.onKeyPress(strike)).toBe(false)

    expect(sampleTab.actions).toEqual(['A', 'C', 'B'])
  })

  it('is taken by the first tab that reads it, and by no later one', async () => {
    const recorder = createKeyRecorderTab('C')
    const window = notifiedCalls([recorder.one.declared])
    const first = await window.tabs.openTabOfKind('thing', 'A')
    await window.groups.handle.openTabAdjacent('thing', 'B')
    await window.groups.handle.openTabAdjacent('thing', 'C')
    window.panes.show(first)

    expect(window.panes.onKeyPress(strike)).toBe(true)

    expect(recorder.actions).toEqual(['A', 'C'])
  })

  it('is not offered to a tab hidden behind another in its pane', async () => {
    const recorder = createKeyRecorderTab('none')
    const window = notifiedCalls([recorder.one.declared])
    await window.tabs.openTabOfKind('thing', 'Hidden')
    await window.tabs.openTabOfKind('thing', 'Shown')

    window.panes.onKeyPress(strike)

    expect(recorder.actions).toEqual(['Shown'])
  })

  it('is no trouble for a layout drawing a tab the window does not hold', () => {
    const window = notifiedCalls([kind().declared])
    window.panes.layout.value = openTab(window.panes.layout.value, 'ghost')

    expect(window.panes.onKeyPress(strike)).toBe(false)
  })

  it('is offered to the tabs on screen while the pane in front is unknown to the layout', async () => {
    const recorder = createKeyRecorderTab('A')
    const window = notifiedCalls([recorder.one.declared])
    await window.tabs.openTabOfKind('thing', 'A')
    window.panes.layout.value = { ...window.panes.layout.value, focus: 'ghost' }

    expect(window.panes.onKeyPress(strike)).toBe(true)
    expect(recorder.actions).toEqual(['A'])
  })
})

describe('a tab release', () => {
  it('closes where its kind has nothing to finish', async () => {
    const simpleKind = kind()
    delete simpleKind.one.events?.onClose
    const window = notifiedCalls([simpleKind.declared])
    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    expect(window.panes.releaseTab(id)).toBe(true)

    expect(window.tabs.getTab(id)).toBeNull()
  })
})

describe('reconciling tabs for a path, by what a tab relies on', () => {
  it('leaves a tab with no path alone when requested for no path', async () => {
    const pathlessKind = kind({ kind: 'bare' })
    const window = notifiedCalls([pathlessKind.declared])
    const id = await window.tabs.openTabOfKind('bare', 'x')

    window.panes.reconcileTab('', 'sample')

    expect(window.tabs.getTab(id)).not.toBeNull()
    expect(pathlessKind.closedPaths).toEqual([])
  })

  it('leaves a tab of another kind alone where it relies on another path', async () => {
    const note = kind({
      kind: 'note',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([note.declared])
    const id = await window.tabs.openTabOfKind('note', 'Other.md')

    window.panes.reconcileTab('Shapes.md', 'sample')

    expect(window.tabs.getTab(id)).not.toBeNull()
  })

  it('reads the path from the target where the kind has no open tab', async () => {
    const note = kind({
      kind: 'note',
      getTarget: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([note.declared])
    const id = await window.tabs.openTabOfKind('note', 'Shapes.md')

    window.panes.reconcileTab('Shapes.md', 'sample')

    expect(window.tabs.getTab(id)).toBeNull()
  })

  it('reads the path from the target where the open tab has none', async () => {
    const note = kind({
      kind: 'note',
      getOpenTab: () => ({ path: '' }),
      getTarget: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    const window = notifiedCalls([note.declared])
    const id = await window.tabs.openTabOfKind('note', 'Shapes.md')

    window.panes.reconcileTab('Shapes.md', 'sample')

    expect(window.tabs.getTab(id)).toBeNull()
  })
})

describe('a tab opened on nothing', () => {
  it('is opened by its kind on the empty address, in front or alongside', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

    await window.tabs.openTabOfKind('thing')
    await window.groups.handle.openTabAdjacent('thing')

    expect(thing.opened).toEqual(['', ''])
  })
})

describe('a window going, with kinds that say nothing about it', () => {
  it('closes without a word to a tab whose kind has neither way of going', async () => {
    const closingKind = kind()
    delete closingKind.one.events?.onClose
    const window = notifiedCalls([closingKind.declared])
    await window.tabs.openTabOfKind('thing', 'x')

    expect(() => window.restored.close()).not.toThrow()
  })
})

describe('reconciling tabs for a path, by what a tab names', () => {
  it('leaves a tab alone whose kind says nothing of at its current position', async () => {
    const pathlessKind = kind({ kind: 'bare' })
    const window = notifiedCalls([pathlessKind.declared])
    const id = await window.tabs.openTabOfKind('bare', 'x')

    expect(() => window.panes.reconcileTab('x', 'sample')).not.toThrow()

    expect(window.tabs.getTab(id)).not.toBeNull()
  })

  it('reads the file from the target where it names no path', async () => {
    const note = kind({
      kind: 'note',
      getTarget: (state: { targetPath: string }) => ({ file: state.targetPath }),
    })
    const window = notifiedCalls([note.declared])
    const id = await window.tabs.openTabOfKind('note', 'Shapes.md')

    window.panes.reconcileTab('Shapes.md', 'sample')

    expect(window.tabs.getTab(id)).toBeNull()
  })

  it('leaves a tab alone whose target names neither path nor file', async () => {
    const note = kind({ kind: 'note', getTarget: () => ({}) })
    const window = notifiedCalls([note.declared])
    const id = await window.tabs.openTabOfKind('note', 'Shapes.md')

    window.panes.reconcileTab('Shapes.md', 'sample')

    expect(window.tabs.getTab(id)).not.toBeNull()
  })
})

describe('a tab created without being shown', () => {
  it('is opened by its kind on the empty address', async () => {
    const thing = kind()
    const window = notifiedCalls([thing.declared])

    const id = await window.tabs.createTab('thing')

    expect(thing.opened).toEqual([''])
    expect(onScreen(window.panes.layout.value)).not.toContain(id)
  })
})

describe('the registration of a kind', () => {
  it('is removed by its own disposer', async () => {
    const window = createTestTabs()
    const dispose = window.groups.registerKind(kind().one)

    dispose()

    const id = await window.tabs.createTab('thing', 'a')
    expect(window.tabs.getTab(id)?.state).toMatchObject({ kind: 'thing', targetPath: 'a' })
  })

  it('is refused for a kind another registration holds, which keeps its kind', async () => {
    const window = createTestTabs()
    const first = { ...kind().one, pluginId: 'md.numen.first' }
    const second = { ...kind().one, pluginId: 'md.numen.second' }
    window.groups.registerKind(first)

    expect(() => window.groups.registerKind(second)).toThrowError(
      'The tab kind "thing" is registered already by plugin "md.numen.first"',
    )

    const id = await window.tabs.createTab('thing', 'a')
    expect(window.tabs.getTab(id)?.kind).toBe(first)
  })

  it('is refused in a list of kinds that names a kind twice, and registers none of them', () => {
    const window = createTestTabs()

    expect(() => window.groups.registerKinds([kind().one, kind().one])).toThrowError(
      'is registered already',
    )

    expect(window.groups.hasKind('thing')).toBe(false)
  })

  it('is free again after the disposer of the registration runs', () => {
    const window = createTestTabs()
    window.groups.registerKind(kind().one)()

    expect(() => window.groups.registerKind(kind().one)).not.toThrow()
  })

  it('names the plugin of the kind of a tab', async () => {
    const window = createTestTabs()
    window.groups.registerKind({ ...kind().one, kind: 'reader:x', pluginId: 'md.numen.x' })

    const id = await window.tabs.createTab('reader:x', 'a.x')

    expect(window.tabs.getTab(id)?.kind.pluginId).toBe('md.numen.x')
  })
})

describe('a tab of a kind that is not registered', () => {
  it('opens on a restore without throwing and lists itself under the missing kind', async () => {
    const window = createTestTabs()

    const id = await window.tabs.createTab('reader:viewer', '/a.bin')

    expect(id).toBe('reader:viewer:/a.bin')
    expect(window.tabs.tabs.value).toEqual([{ id, title: 'a.bin' }])
  })

  it('does not stop the kind being registered and opened later', async () => {
    const window = createTestTabs()
    const missing = await window.tabs.createTab('thing', 'a')
    const thing = kind()
    window.groups.registerKind(thing.one)

    const openedTab = await window.tabs.createTab('thing', 'b')

    expect(openedTab).not.toBe(missing)
    expect(window.tabs.getTab(openedTab)?.state).toEqual({ targetPath: 'b', title: 'b' })
  })
})

describe('the identity a kind is given as its tab opens', () => {
  it('is the one the window files the tab under', async () => {
    const provided: string[] = []
    const thing = kind({
      open: (targetPath: string, id: string) => {
        provided.push(id)
        return { targetPath, title: targetPath }
      },
    })
    const window = createTestTabs()
    window.groups.registerKinds([thing.one])

    const id = await window.tabs.openTabOfKind('thing', 'Note.md')

    expect(provided).toEqual([id])
  })

  it('reuses a tab id from another window when one is given', async () => {
    const provided: string[] = []
    const thing = kind({
      open: (targetPath: string, id: string) => {
        provided.push(id)
        return { targetPath, title: targetPath }
      },
    })
    const window = createTestTabs()
    window.groups.registerKinds([thing.one])

    const id = await window.tabs.openTabOfKind('thing', 'Note.md', 'thing:4f1c')

    expect(id).toBe('thing:4f1c')
    expect(provided).toEqual(['thing:4f1c'])
    expect(onScreen(window.panes.layout.value)).toContain('thing:4f1c')
  })
})

describe('the kinds the window holds', () => {
  it('are the ones registered, and a kind removed is no longer listed', () => {
    const window = createTestTabs()
    const release = window.groups.registerKind(kind().one)

    expect(window.groups.hasKind('thing')).toBe(true)
    expect(window.groups.hasKind('other')).toBe(false)
    release()
    expect(window.groups.hasKind('thing')).toBe(false)
  })
  it('finds the first registered that answers, and none where none does', () => {
    const window = createTestTabs()
    window.groups.registerKind(kind().one)
    window.groups.registerKind(kind({ kind: 'chat', receiveText: vi.fn() }).one)

    expect(window.groups.findKind((one) => one.events?.receiveText !== undefined)?.kind).toBe(
      'chat',
    )
    expect(window.groups.findKind((one) => one.display?.isTool === true)).toBeUndefined()
  })
})

describe('a window opened with nothing to restore', () => {
  const placedTab = (name: string, pane: 'main' | 'aside', order: number) =>
    kind({ kind: name, placement: { pane, order } }).one

  it('opens one tab of every placed kind, in the pane and the order each declares', async () => {
    const window = createTestTabs()
    window.groups.registerKinds([
      placedTab('files', 'aside', 1),
      kind({ kind: 'unplaced' }).one,
      placedTab('agent', 'aside', 0),
      placedTab('plex', 'main', 0),
    ])

    await window.restored.openPlacedTabs(() => {
      throw new Error('no placed tab fails here')
    })

    const panes = getPanes(window.panes.layout.value.root)
    expect(panes.map((one) => [one.id, one.tabs.map((id) => id.split(':')[0])])).toStrictEqual([
      ['main', ['plex']],
      ['aside', ['agent', 'files']],
    ])
    expect(window.tabs.tabs.value.map((tab) => tab.id.split(':')[0])).toStrictEqual([
      'plex',
      'agent',
      'files',
    ])
  })

  it('keeps a tab the person opened before the placed tabs were laid out, in front', async () => {
    const window = createTestTabs()
    window.groups.registerKinds([
      {
        ...placedTab('plex', 'main', 0),
        address: { identity: () => 'main' },
      },
      placedTab('agent', 'aside', 0),
      kind({ kind: 'settings' }).one,
    ])
    // Opening a placed singleton before openPlacedTabs does not duplicate it
    await window.tabs.openTabOfKind('plex')
    const ghost = await window.tabs.openTabOfKind('settings')
    window.panes.releaseTab(ghost)
    const early = await window.tabs.openTabOfKind('settings')

    await window.restored.openPlacedTabs(() => {
      throw new Error('no placed tab fails here')
    })

    const [main, aside] = getPanes(window.panes.layout.value.root)
    expect(main?.tabs.map((id) => id.split(':')[0])).toStrictEqual(['plex', 'settings'])
    expect(main?.tabs).not.toContain(ghost)
    expect(main?.active).toBe(early)
    expect(aside?.tabs.map((id) => id.split(':')[0])).toStrictEqual(['agent'])
  })

  it('gives the whole window to the aside kinds where no kind is placed in the main pane', async () => {
    const window = createTestTabs()
    window.groups.registerKinds([placedTab('agent', 'aside', 0)])

    await window.restored.openPlacedTabs(() => {
      throw new Error('no placed tab fails here')
    })

    expect(getPanes(window.panes.layout.value.root).map((one) => one.id)).toStrictEqual(['main'])
    expect(onScreen(window.panes.layout.value).map((id) => id.split(':')[0])).toStrictEqual([
      'agent',
    ])
  })
})

describe('a placed kind whose tab fails to open', () => {
  it('is notified, left out of the layout, and releases what it started', async () => {
    const failure = new Error('the plugin is gone')
    const stoppedTabs: string[] = []
    const window = createTestTabs()
    window.groups.registerKinds([
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

    await window.restored.openPlacedTabs((name, error) => notifiedCalls.push([name, error]))

    expect(notifiedCalls).toStrictEqual([['plex', failure]])
    expect(stoppedTabs).toStrictEqual(['plex'])
    expect(onScreen(window.panes.layout.value).map((id) => id.split(':')[0])).toStrictEqual([
      'agent',
    ])
    expect(window.tabs.tabs.value.map((tab) => tab.id.split(':')[0])).toStrictEqual(['agent'])
  })
})

describe('a tab of a kind moved to a path', () => {
  /** A kind whose tabs have a path and move when requested. */
  const navigations: string[] = []
  const walker = (name = 'walker') =>
    kind({
      kind: name,
      open: (targetPath: string) => ({ targetPath, title: targetPath }),
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
      navigate: (state: { targetPath: string; title: string }, path: string, query: string) => {
        navigations.push(`${path}:${query}`)
        state.targetPath = path
      },
    })

  it('is the one the person was last in, put in front', async () => {
    navigations.length = 0
    const one = walker()
    const window = createTestTabs()
    window.groups.registerKinds([one.one, kind().one])
    const zero = await window.tabs.openTabOfKind('walker', 'Zero.md')
    const first = await window.tabs.openTabOfKind('walker', 'One.md')
    const second = await window.tabs.openTabOfKind('walker', 'Two.md')
    window.panes.onTabShown(zero)
    window.panes.onTabShown(first)
    await window.tabs.openTabOfKind('thing', 'Else.md')

    const moved = await window.tabs.navigateTab('walker', 'Wanted.md')

    expect(moved).toBe(first)
    expect(navigations).toStrictEqual(['Wanted.md:'])
    expect(window.tabs.getTab(first)?.state).toMatchObject({ targetPath: 'Wanted.md' })
    expect(window.tabs.getTab(second)?.state).toMatchObject({ targetPath: 'Two.md' })
    expect(paneById(window.panes.layout.value.root, window.panes.layout.value.focus)?.active).toBe(
      first,
    )

    // Navigating a kind without a navigate callback still shows the tab
    expect(await window.tabs.navigateTab('thing', 'Other.md')).toBeDefined()
  })

  it('is a new one opened there where the window holds none', async () => {
    const window = createTestTabs()
    window.groups.registerKinds([walker().one])

    const id = await window.tabs.navigateTab('walker', 'Wanted.md')

    expect(window.tabs.getTab(id)?.state).toMatchObject({ targetPath: 'Wanted.md' })
  })
})

describe('the tabs standing on a path that went', () => {
  it('move to another, every one of a kind that navigates, and no other', async () => {
    const window = createTestTabs()
    const receivedQueries: string[] = []
    const fixed = kind({
      kind: 'fixed',
      getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    })
    window.groups.registerKinds([
      kind({
        kind: 'walker',
        getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
        navigate: (state: { targetPath: string }, path: string, query: string) => {
          receivedQueries.push(query)
          state.targetPath = path
        },
      }).one,
      fixed.one,
    ])
    const removed = await window.tabs.openTabOfKind('walker', 'Gone.md')
    const survivor = await window.tabs.openTabOfKind('walker', 'Elsewhere.md')
    const stationary = await window.tabs.openTabOfKind('fixed', 'Gone.md')

    await window.tabs.navigateTabsAt('Gone.md', 'Root.md')

    expect(receivedQueries).toStrictEqual([''])
    expect(window.tabs.getTab(removed)?.state).toMatchObject({ targetPath: 'Root.md' })
    expect(window.tabs.getTab(survivor)?.state).toMatchObject({ targetPath: 'Elsewhere.md' })
    expect(window.tabs.getTab(stationary)?.state).toMatchObject({ targetPath: 'Gone.md' })
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
    const window = createTestTabs()
    window.groups.registerKinds([listener().one, kind().one])
    const zero = await window.tabs.openTabOfKind('listener')
    const first = await window.tabs.openTabOfKind('listener')
    const second = await window.tabs.openTabOfKind('listener')
    await window.tabs.openTabOfKind('thing')
    window.panes.onTabShown(zero)
    window.panes.onTabShown(first)
    window.panes.onTabShown(second)

    const dispatched = await window.groups.sendText('listener', 'what is this?')

    expect(dispatched).toBe(second)
    expect(window.tabs.getTab(first)?.state).toMatchObject({ received: [] })
    expect(window.tabs.getTab(second)?.state).toMatchObject({ received: ['what is this?'] })
    expect(paneById(window.panes.layout.value.root, window.panes.layout.value.focus)?.active).toBe(
      second,
    )
  })

  it('opens a tab to take it where the window holds none', async () => {
    const window = createTestTabs()
    window.groups.registerKinds([listener().one, kind().one])
    const otherId = await window.tabs.openTabOfKind('thing')

    const id = await window.groups.sendText('listener', 'what is this?')

    expect(id).not.toBe(otherId)
    expect(window.tabs.getTab(id)?.state).toMatchObject({ received: ['what is this?'] })
    expect(paneById(window.panes.layout.value.root, window.panes.layout.value.focus)?.active).toBe(
      id,
    )
  })

  it('is left unheard by a tab of a kind that takes no text', async () => {
    const window = createTestTabs()
    window.groups.registerKinds([kind().one])
    const id = await window.tabs.openTabOfKind('thing')

    await expect(window.groups.sendText('thing', 'hello')).resolves.toBe(id)
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

    expect(await window.groups.createFileOfKind('thing', 'dir', 'Name')).toStrictEqual({
      ok: true,
      value: { path: 'dir/Name.md' },
    })
    expect(createdItem).toStrictEqual([['dir', 'Name']])
  })

  it('is requested of nothing where the kind makes no file or is not registered', () => {
    const window = notifiedCalls([kind().declared])

    expect(window.groups.createFileOfKind('thing', 'dir', 'Name')).toBeUndefined()
    expect(window.groups.createFileOfKind('nothing', 'dir', 'Name')).toBeUndefined()
  })
})

const pagedKind = (pluginId = 'viewer.plugin'): AnyTabKind => ({
  kind: 'paged',
  pluginId,
  open: (targetPath: string, _id: string, saved?: unknown) => ({
    targetPath,
    page: (saved as { page?: number } | undefined)?.page ?? 1,
  }),
  pane: {},
  address: {
    getState: (state: { page: number }) => ({ page: state.page }),
    getOpenTab: (state: { targetPath: string }) => ({ path: state.targetPath }),
    identity: (targetPath: string) => targetPath,
  },
})

describe('the tabs the core kept', () => {
  it('open again under their ids with the state their kind saved, the front one in front', async () => {
    const window = createTestTabs()
    const disposedIds: string[] = []
    window.groups.registerKind({
      ...pagedKind(),
      open: (targetPath: string, id: string, saved?: unknown) => {
        onScopeDispose(() => disposedIds.push(id))
        return {
          targetPath,
          page: (saved as { page?: number } | undefined)?.page ?? 1,
        }
      },
    })
    window.groups.registerKind({
      ...pagedKind('aside.plugin'),
      kind: 'aside-kind',
      display: { placement: { pane: 'aside', order: 0 } },
    })

    const alreadyOpen = await window.tabs.openTabOfKind('paged', 'existing.bin')
    ;(window.tabs.getTabStateIn(alreadyOpen, 'paged') as { page: number }).page = 99

    const parent = effectScope()
    parent.run(() => {
      window.restored.restoreTabs(
        [
          { id: alreadyOpen, kind: 'paged', path: 'existing.bin', saved: { page: 1 } },
          { id: 'paged:a.bin', kind: 'paged', path: 'a.bin', saved: { page: 4 } },
          { id: 'paged:b.bin', kind: 'paged', path: 'b.bin' },
          { id: 'aside-kind:s.bin', kind: 'aside-kind', path: 's.bin' },
        ],
        'paged:a.bin',
      )
    })
    parent.stop()
    expect(disposedIds).toStrictEqual([])

    expect(window.tabs.getTabStateIn(alreadyOpen, 'paged')).toMatchObject({ page: 99 })
    expect(window.tabs.getTabStateIn('paged:a.bin', 'paged')).toMatchObject({ page: 4 })
    expect(window.tabs.getTabStateIn('paged:b.bin', 'paged')).toMatchObject({ page: 1 })
    expect(window.groups.handle.front()?.id).toBe('paged:a.bin')
    expect(getPanes(window.panes.layout.value.root).map((p) => [p.id, p.tabs])).toStrictEqual([
      ['main', [alreadyOpen, 'paged:a.bin', 'paged:b.bin']],
      ['aside', ['aside-kind:s.bin']],
    ])

    window.panes.releaseTab('paged:a.bin')
    expect(disposedIds).toStrictEqual(['paged:a.bin'])
  })

  it('open as placeholders that keep the saved state where no plugin registers the kind', () => {
    const window = createTestTabs()

    window.restored.restoreTabs(
      [
        {
          id: 'paged:a.bin',
          kind: 'paged',
          path: '',
          title: 'Saved Title',
          saved: { page: 4 },
        },
      ],
      'nonexistent-id',
    )

    const one = window.tabs.getTab('paged:a.bin')
    expect(one?.kind.address?.getState?.(one.state)).toStrictEqual({ page: 4 })
    expect(window.tabs.tabs.value[0]?.title).toBe('Saved Title')
    expect(getPanes(window.panes.layout.value.root).map((p) => p.tabs)).toStrictEqual([
      ['paged:a.bin'],
    ])
  })
})

describe('the tabs of a plugin that stops', () => {
  it('turn into placeholders that keep the state, and back once the kind is registered again', () => {
    const window = createTestTabs()
    const disposed: string[] = []
    const destroyed: string[] = []
    const closed: string[] = []
    const makeKind = (withDestroy: boolean): AnyTabKind => ({
      ...pagedKind(),
      open: (targetPath: string, id: string, saved?: unknown) => {
        onScopeDispose(() => disposed.push(`open:${id}`))
        return {
          targetPath,
          page: (saved as { page?: number } | undefined)?.page ?? 1,
        }
      },
      events: withDestroy
        ? { onDestroy: (_s: unknown, id: string) => destroyed.push(id) }
        : { onClose: (_s: unknown, id: string) => (closed.push(id), true) },
    })

    const unregister = window.groups.registerKind(makeKind(true))
    window.restored.restoreTabs(
      [
        { id: 'paged:a.bin', kind: 'paged', path: 'a.bin', saved: { page: 4 } },
        { id: 'other:b.bin', kind: 'other', path: 'b.bin', saved: { page: 2 } },
      ],
      '',
    )

    const parent = effectScope()
    parent.run(() => {
      window.groups.closeTabsOfPlugin('viewer.plugin')
    })
    parent.stop()

    expect(destroyed).toStrictEqual(['paged:a.bin'])
    expect(disposed).toStrictEqual(['open:paged:a.bin'])
    unregister()
    const placeholder = window.tabs.getTab('paged:a.bin')
    expect(placeholder?.kind.pluginId).toBeUndefined()
    expect(placeholder?.kind.address?.getState?.(placeholder.state)).toStrictEqual({ page: 4 })

    // Re-registering `paged` reopens `paged:a.bin`, leaves `other:b.bin` as placeholder, and leaves active tabs alone
    const parentReopen = effectScope()
    parentReopen.run(() => {
      window.groups.registerKind(makeKind(false))
    })
    parentReopen.stop()
    expect(disposed).toStrictEqual(['open:paged:a.bin'])
    expect(window.tabs.getTabStateIn('paged:a.bin', 'paged')).toMatchObject({ page: 4 })
    expect(window.tabs.getTab('other:b.bin')?.kind.pluginId).toBeUndefined()

    // Registering `paged` again while it is registered is refused and leaves `paged:a.bin` alone
    expect(() => window.groups.registerKind(makeKind(false))).toThrowError('is registered already')
    expect(disposed).toStrictEqual(['open:paged:a.bin'])

    // Closing plugin when kind defines only onClose calls onClose
    window.groups.closeTabsOfPlugin('viewer.plugin')
    expect(closed).toStrictEqual(['paged:a.bin'])

    // Releasing the placeholder stops its scope
    window.panes.releaseTab('paged:a.bin')
  })

  it('leaves the tabs of other plugins as they are', () => {
    const window = createTestTabs()
    window.groups.registerKind(pagedKind('other.plugin'))
    window.restored.restoreTabs([{ id: 'paged:a.bin', kind: 'paged', path: 'a.bin' }], '')

    window.groups.closeTabsOfPlugin('viewer.plugin')

    expect(window.tabs.getTab('paged:a.bin')?.kind.pluginId).toBe('other.plugin')
  })
})

describe('a saved tab whose kind fails to open it', () => {
  it('opens as a placeholder that keeps the saved state, and the other tabs open as usual', () => {
    const window = createTestTabs()
    const disposedOnFail: string[] = []
    window.groups.registerKind({
      ...pagedKind(),
      open: (targetPath: string, id: string) => {
        onScopeDispose(() => disposedOnFail.push(id))
        if (targetPath === 'dir/bad.bin') throw new Error('cannot read bad.bin')
        if (targetPath === '') throw 'string failure'
        return { targetPath, page: 1 }
      },
    })

    const parent = effectScope()
    parent.run(() => {
      window.restored.restoreTabs(
        [
          { id: 'paged:a.bin', kind: 'paged', path: 'a.bin' },
          { id: 'paged:bad.bin', kind: 'paged', path: 'dir/bad.bin', saved: { page: 7 } },
          {
            id: 'paged:titled.bin',
            kind: 'paged',
            path: 'dir/bad.bin',
            title: 'Custom Title',
          },
          { id: 'paged:empty', kind: 'paged', path: '' },
          { id: 'paged:c.bin', kind: 'paged', path: 'c.bin' },
        ],
        'paged:c.bin',
      )
    })
    parent.stop()

    expect(disposedOnFail).toStrictEqual(['paged:bad.bin', 'paged:titled.bin', 'paged:empty'])
    expect(window.tabs.tabs.value.map((one) => [one.id, one.title])).toStrictEqual([
      ['paged:a.bin', ''],
      ['paged:bad.bin', 'bad.bin'],
      ['paged:titled.bin', 'Custom Title'],
      ['paged:empty', 'paged'],
      ['paged:c.bin', ''],
    ])
    expect(window.tabs.getTabStateIn('paged:c.bin', 'paged')).toMatchObject({ page: 1 })
    const failed = window.tabs.getTab('paged:bad.bin')
    expect(failed?.state).toMatchObject({ message: 'cannot read bad.bin' })
    expect(failed?.kind.address?.getState?.(failed.state)).toStrictEqual({ page: 7 })
    expect(failed?.kind.address?.getOpenTab?.(failed.state)).toStrictEqual({ path: 'dir/bad.bin' })
    expect(failed?.kind.address?.identity?.('dir/bad.bin')).toBe('dir/bad.bin')
    expect(window.tabs.getTab('paged:empty')?.state).toMatchObject({ message: 'string failure' })
    expect(window.groups.handle.front()?.id).toBe('paged:c.bin')
  })
})

describe('openSavedTab', () => {
  it('opens the tab with its id and saved state as the active tab', async () => {
    const window = createTestTabs()
    window.groups.registerKind(pagedKind())
    await window.tabs.openTabOfKind('paged', 'existing.bin')

    window.restored.openSavedTab({
      id: 'paged:a.bin',
      kind: 'paged',
      path: 'a.bin',
      saved: { page: 4 },
    })

    expect(window.tabs.getTabStateIn('paged:a.bin', 'paged')).toMatchObject({ page: 4 })
    expect(window.groups.handle.front()?.id).toBe('paged:a.bin')
  })

  it('activates an open tab with the same id without opening it again', async () => {
    const window = createTestTabs()
    window.groups.registerKind(pagedKind())
    const id = await window.tabs.openTabOfKind('paged', 'a.bin')
    ;(window.tabs.getTabStateIn(id, 'paged') as { page: number }).page = 7
    await window.tabs.openTabOfKind('paged', 'b.bin')

    window.restored.openSavedTab({ id, kind: 'paged', path: 'a.bin', saved: { page: 4 } })

    expect(window.tabs.getTabStateIn(id, 'paged')).toMatchObject({ page: 7 })
    expect(window.groups.handle.front()?.id).toBe(id)
  })
})

describe('a tab of a kind whose plugin has not started', () => {
  it('starts the plugin of the kind, and becomes a tab of the kind once the plugin registers it', async () => {
    const started: string[] = []
    const window = createTestTabs({
      ...NO_TAB_KIND_PLUGINS,
      startKindPlugin: (one) => started.push(one),
    })
    const id = await window.tabs.openTabOfKind('thing', 'One.md')

    expect(started).toStrictEqual(['thing'])
    const thing = kind()
    window.groups.registerKind(thing.one)

    expect(window.tabs.getTab(id)?.kind).toBe(thing.one)
    expect(thing.opened).toStrictEqual(['One.md'])
  })

  it('starts nothing for a kind the window holds', async () => {
    const started: string[] = []
    const window = createTestTabs({
      ...NO_TAB_KIND_PLUGINS,
      startKindPlugin: (one) => started.push(one),
    })
    window.groups.registerKind(kind().one)

    await window.tabs.openTabOfKind('thing', 'One.md')

    expect(started).toStrictEqual([])
  })
})
