/**
 * The plex tabs of a window: where each one stands, which one a new plex opens
 * beside, and what the window is told about one.
 */
import { describe, expect, it } from 'vitest'
import { effectScope, type EffectScope } from 'vue'
import { createPlexKind, PLEX } from './kind'
import { createHost, type HostOptions } from './fixtures/host'
import { settle, viewOn } from './fixtures'
import type { PlexTabState } from './types'

/** The plex tabs of a window, opened, shown and closed the way the window does it. */
const window = (options: HostOptions = {}) => {
  const fixture = createHost(options)
  const views: ReturnType<typeof viewOn>[] = []
  /** How many times a tab handed its view back. */
  const closed: number[] = []
  const createView = () => {
    const view = viewOn('')
    views.push(view)
    ;(view.view as { close: () => void }).close = () => {
      closed.push(views.length)
    }
    return view.view
  }
  const plexes = createPlexKind(fixture.host, createView)
  const { kind } = plexes
  const open = new Map<string, { state: PlexTabState; scope: EffectScope }>()
  let count = 0

  /** A plex tab of this window, opened on what it was given. */
  const openTab = async (at = '') => {
    count += 1
    const id = `${PLEX}:${count}`
    const scope = effectScope(true)
    const state = scope.run(() => kind.open!(at, id))!
    open.set(id, { state, scope })
    await settle()
    return { id, state }
  }
  /** The person is in this tab now. */
  const showTab = (id: string) => kind.onShow!(open.get(id)!.state, id)
  /** The tab closes, and the window drops what it held. */
  const closeTab = (id: string) => {
    const one = open.get(id)!
    kind.onClose!(one.state, id)
    one.scope.stop()
    open.delete(id)
  }
  return { ...fixture, ...plexes, openTab, showTab, closeTab, views, closed }
}

describe('a plex tab as it opens', () => {
  it('relies on the note the vault opens with', async () => {
    const { state } = await window({ opening: 'Opening.md' }).openTab()

    expect(state.view.here.value).toBe('Opening.md')
  })

  it('remains where it was told to, whatever the vault opens with', async () => {
    const { state } = await window({ opening: 'Opening.md' }).openTab('Told.md')

    expect(state.view.here.value).toBe('Told.md')
  })

  it('remains where the plex the person was last in remains, when another one is open', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    one.showTab(first.id)

    const third = await one.openTab()

    expect(third.state.view.here.value).toBe('One.md')
    expect(second.state.view.here.value).toBe('Two.md')
  })

  it('remains where the one before remains when the tab the person was in has closed', async () => {
    const one = window()
    await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    one.closeTab(second.id)

    const third = await one.openTab()

    expect(third.state.view.here.value).toBe('One.md')
  })

  it('remains nowhere, and is sent nowhere, while the vault opens with nothing', async () => {
    const one = window({ opening: '' })

    const { state } = await one.openTab()

    expect(state.view.here.value).toBe('')
    expect(one.views[0]?.went).toStrictEqual([])
  })
})

describe('a change to the vault', () => {
  it('asks every plex for the note it is standing on, each of its own', async () => {
    const one = window()
    await one.openTab('One.md')
    const second = await one.openTab('Two.md')

    await one.refresh({})

    expect(one.views[0]?.went).toContain('One.md')
    expect(one.views[1]?.went).toContain('Two.md')
    expect(second.state.view.here.value).toBe('Two.md')
  })

  it('remains a plex on where the note under it went, under the same node', async () => {
    const one = window()
    const plex = await one.openTab('One.md')
    const before = plex.state.picture.value?.nodes.map((node) => node.id)

    await one.refresh({ renamed: [{ from: 'One.md', to: 'Renamed.md' }] })

    expect(plex.state.view.here.value).toBe('Renamed.md')
    expect(one.views[0]?.went.at(-1)).toBe('Renamed.md')
    expect(plex.state.picture.value?.nodes.map((node) => node.id)).toStrictEqual(before)
  })

  it('leaves a plex standing on a note nothing moved', async () => {
    const one = window()
    const plex = await one.openTab('One.md')

    await one.refresh({ renamed: [{ from: 'Other.md', to: 'Renamed.md' }] })

    expect(plex.state.view.here.value).toBe('One.md')
  })

  it('gives one standing nowhere the note an empty vault has just gained', async () => {
    const one = window({ opening: '' })
    const { state } = await one.openTab()
    one.openingPath.value = 'First.md'

    await one.refresh({})

    expect(state.view.here.value).toBe('First.md')
  })

  it('asks the vault where it opens once, however many stand nowhere', async () => {
    const one = window({ opening: '' })
    await one.openTab()
    await one.openTab()
    await one.openTab('One.md')

    await one.refresh({})

    expect(one.asked.opening).toHaveLength(1)
  })

  it('asks it not at all while every one of them is standing somewhere', async () => {
    const one = window()
    await one.openTab('One.md')
    await one.openTab('Two.md')

    await one.refresh({})

    expect(one.asked.opening).toStrictEqual([])
  })

  it('asks nothing for a plex whose tab has closed', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    one.closeTab(second.id)

    await one.refresh({})

    expect(one.views[1]?.went.filter((where) => where === 'Two.md')).toHaveLength(1)
    expect(first.state.view.here.value).toBe('One.md')
  })

  it('sends no plex anywhere while the vault opens with nothing', async () => {
    const one = window({ opening: '' })
    await one.openTab()

    await one.refresh({})

    expect(one.views[0]?.went).toStrictEqual([])
  })
})

describe('a plex tab the person has closed', () => {
  it('passes its view back', async () => {
    const one = window()
    await one.openTab('One.md')
    const second = await one.openTab('Two.md')

    one.closeTab(second.id)

    expect(one.closed).toStrictEqual([2])
  })

  it('asks the vault nothing when a setting it once answered turns', async () => {
    const one = window()
    const plex = await one.openTab('One.md')
    one.isNesting.value = false
    await settle()

    one.closeTab(plex.id)
    one.asked.headings.length = 0
    one.isNesting.value = true
    await settle()

    expect(one.asked.headings).toStrictEqual([])
  })
})

describe('what the window is told about a plex tab', () => {
  it('is the note it relies on as what a command is over, under the name the picture gives it', async () => {
    const one = window()
    const { state } = await one.openTab('physics/Ontology.md')

    expect(one.kind.getTarget!(state)).toStrictEqual({
      path: 'physics/Ontology.md',
      title: 'physics/Ontology',
    })
    expect(one.kind.getOpenTab!(state)).toStrictEqual({ path: 'physics/Ontology.md' })
  })

  it('is no note at all while the plex remains nowhere', async () => {
    const one = window({ opening: '' })
    const { state } = await one.openTab()

    expect(one.kind.getTarget!(state)).toStrictEqual({ path: '', title: '' })
  })

  it('calls it the note it relies on, and a plex while active nowhere', async () => {
    const one = window({ opening: '' })
    const somewhere = await one.openTab('Root.md')
    const nowhere = await window({ opening: '' }).openTab()

    expect(one.kind.getTitle!(somewhere.state)).toBe('Root')
    expect(one.kind.getTitle!(nowhere.state)).toBe('Plex')
  })

  it('calls it a plex while its view holds no neighbourhood', async () => {
    const one = window({ opening: '' })
    const { state } = await one.openTab()
    ;(state.view.neighbourhood as { value: unknown }).value = null

    expect(one.kind.getTitle!(state)).toBe('Plex')
  })

  it('is a tool the main pane opens with, under the plex icon', () => {
    const { kind } = window()

    expect(kind.kind).toBe('plex')
    expect(kind.isTool).toBe(true)
    expect(kind.placement).toStrictEqual({ pane: 'main', order: 0 })
    expect(kind.icon).toBeDefined()
  })
})

describe('a plex tab as it moves', () => {
  it('notifies the tabs linked to it the note it came to stand on, under its name', async () => {
    const one = window({ opening: '' })
    const { id, state } = await one.openTab()

    await state.view.go('physics/Ontology.md')
    await settle()

    expect(one.recorded.synced).toStrictEqual([[id, 'physics/Ontology.md', 'physics/Ontology']])
  })

  it('notifies them nothing when it evaluates to stand nowhere', async () => {
    const one = window()
    const { state } = await one.openTab('One.md')
    one.recorded.synced.length = 0

    await state.view.go('')
    await settle()

    expect(one.recorded.synced).toStrictEqual([])
  })

  it('notifies them the note of a node the person chose', async () => {
    const one = window()
    const { id, state } = await one.openTab('One.md')
    one.recorded.synced.length = 0

    state.activate(state.picture.value!.nodes[0]!.id)

    expect(one.recorded.synced[0]).toStrictEqual([id, 'One.md', 'One'])
  })
})

describe('a plex tab a linked tab moves', () => {
  it('goes to the note it is moved to', async () => {
    const one = window()
    const { state } = await one.openTab('Start.md')

    await one.kind.navigate!(state, 'Next.md', 'Next')

    expect(state.view.here.value).toBe('Next.md')
  })

  it('stays at its current position when moved to a file that is not a note, or to where it is', async () => {
    const one = window()
    const { state } = await one.openTab('Start.md')
    const went = one.views[0]!.went.length

    await one.kind.navigate!(state, 'Books/Guide.pdf', '')
    await one.kind.navigate!(state, 'Start.md', '')

    expect(one.views[0]!.went).toHaveLength(went)
    expect(state.view.here.value).toBe('Start.md')
  })
})
