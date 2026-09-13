/**
 * The plex tabs of a window: where each one stands, which the person is looking
 * at, and what the window answers about one.
 *
 * A note is asked for somewhere the window cannot see — the palette, an agent
 * working the vault beside the person — and one of these has to take it.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { paneById, panesOf } from '@numen/ui'
import { plexKind } from './kind'
import type { PlexTabState } from './model/usePlexTab'
import { createVault, settle, viewOn } from './fixtures'
import { useWindowTabs, type AnyTabKind } from '@/entities/tab'
import { PLEX } from '@/entities/tab'

/** A kind that is not a plex, for the person to be in a tab of. */
const other: AnyTabKind = {
  kind: 'other',
  open: () => ({}),
  getTitle: () => 'Other',
  pane: {},
}

/** The plexes of a window, each standing where it was told to. */
const window = (opening = 'Opening.md') => {
  const views: ReturnType<typeof viewOn>[] = []
  /** Every time the vault was asked where it opens, and what it answered then. */
  const asked: string[] = []
  /** Whether a node hangs the parts of its note, which a test turns. */
  const isHanging = ref(true)
  /** Every question the vault was asked about what the notes hold. */
  const headingsAsked: (readonly string[])[] = []
  const openingPath = ref(opening)

  const createView = () => {
    const view = viewOn('')
    views.push(view)
    return view.view
  }
  const held = useWindowTabs()
  const plexes = plexKind(held.handle, createView, {
    editor: createVault().editor,
    ready: ref(true),
    isHanging,
    parts: ref(6),
    openNote: () => {},
    readHeadings: async (paths) => {
      headingsAsked.push(paths)
      return new Map()
    },
    askAgent: () => {},
    runCommand: () => {},
    openingPath,
    readOpeningPath: async () => {
      asked.push(openingPath.value)
      return openingPath.value
    },
    dragged: ref([]),
    showMessage: () => {},
    createUntitledNote: async () => '',
    creatable: ['parent', 'child', 'jump'],
  })
  held.registerKinds([plexes.kind, other])

  /** A plex tab of this window, opened on what it was given. */
  const openTab = async (at = '') => {
    const id = await held.openTabOfKind(PLEX, at)
    return { id, state: held.getTabStateIn<PlexTabState>(id, PLEX)! }
  }
  /** The person is in this tab now. */
  const showTab = (id: string) => held.onTabShown(id)
  /** The tab closes, and the window lets go of what it held. */
  const closeTab = (id: string) => held.releaseTab(id)
  /** The vault gained a note, which is what it opens with from now on. */
  const setOpeningNote = (path: string) => {
    openingPath.value = path
  }
  const onScreen = () => panesOf(held.layout.value.root).flatMap((pane) => pane.tabs)
  /** The tab the person is in, which is the active tab of the pane they are in. */
  const active = () => paneById(held.layout.value.root, held.layout.value.focus)?.active ?? ''
  /** A tab holding no plex, opened in front of the person. */
  const elsewhere = () => held.openTabOfKind(other.kind)
  return {
    ...plexes,
    openTab,
    showTab,
    closeTab,
    setOpeningNote,
    onScreen,
    active,
    elsewhere,
    views,
    asked,
    isHanging,
    headingsAsked,
  }
}

describe('a plex tab as it opens', () => {
  it('stands on the note the vault opens with', async () => {
    const one = window('Opening.md')

    const { state } = await one.openTab()

    expect(state.view.here.value).toBe('Opening.md')
  })

  it('stands where it was told to, whatever the vault opens with', async () => {
    const one = window('Opening.md')

    const { state } = await one.openTab('Told.md')

    expect(state.view.here.value).toBe('Told.md')
  })

  it('stands where the person is looking, when another one is open', async () => {
    const one = window('Opening.md')
    const first = await one.openTab()
    await first.state.view.go('Here.md')

    const second = await one.openTab()

    expect(second.state.view.here.value).toBe('Here.md')
  })

  it('stands nowhere while the vault opens with nothing', async () => {
    const one = window('')

    expect((await one.openTab()).state.view.here.value).toBe('')
  })
})

describe('the plex the person is looking at', () => {
  it('is the one they were last in', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')

    one.showTab(first.id)
    expect(one.looking()).toBe('One.md')

    one.showTab(second.id)
    expect(one.looking()).toBe('Two.md')
  })

  it('does not carry the error of a tab that closed to the one before it', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    second.state.view.error.value = 'Two.md is not in the vault'

    one.closeTab(second.id)

    expect(one.looking()).toBe('One.md')
    expect(first.state.view.error.value).toBe('')
  })

  it('is the one before it when the tab in front closes', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    const third = await one.openTab('Three.md')
    one.showTab(second.id)
    one.showTab(third.id)

    one.closeTab(third.id)

    expect(one.looking()).toBe('Two.md')
    expect(first.state.view.here.value).toBe('One.md')
  })

  it('is nothing at all in a window holding no plex', () => {
    const one = window()

    expect(one.looking()).toBe('')
  })
})

describe('a note put in front of the person', () => {
  it('is where the plex they are looking at travels', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    one.showTab(second.id)

    await one.travel('Wanted.md')

    expect(second.state.view.here.value).toBe('Wanted.md')
    expect(first.state.view.here.value).toBe('One.md')
  })

  it('opens a plex of its own in a window holding none', async () => {
    const one = window()

    await one.travel('Wanted.md')

    expect(one.onScreen()).toHaveLength(1)
    expect(one.looking()).toBe('Wanted.md')
  })
})

describe('a note that is no longer in the vault', () => {
  it('leaves every plex standing on it somewhere else', async () => {
    const one = window()
    const first = await one.openTab('Gone.md')
    const second = await one.openTab('Gone.md')
    const third = await one.openTab('Elsewhere.md')

    await one.leavePath('Gone.md', 'Root.md')

    expect(first.state.view.here.value).toBe('Root.md')
    expect(second.state.view.here.value).toBe('Root.md')
    expect(third.state.view.here.value).toBe('Elsewhere.md')
  })

  it('leaves a window holding no plex at all alone', async () => {
    const one = window()

    await expect(one.leavePath('Gone.md', 'Root.md')).resolves.toBeUndefined()
  })

  it('brings the plex in front of the person, who was in another tab', async () => {
    const one = window()
    const plex = await one.openTab('One.md')
    await one.elsewhere()

    await one.travel('Wanted.md')

    expect(one.active()).toBe(plex.id)
    expect(plex.state.view.here.value).toBe('Wanted.md')
  })
})

describe('every plex asked for its picture again', () => {
  it('asks for the note it is standing on, each of its own', async () => {
    const one = window()
    await one.openTab('One.md')
    const second = await one.openTab('Two.md')

    await one.refresh()

    expect(one.views[0]?.went).toContain('One.md')
    expect(one.views[1]?.went).toContain('Two.md')
    expect(second.state.view.here.value).toBe('Two.md')
  })

  it('stands a plex on where the note under it went', async () => {
    const one = window()
    const plex = await one.openTab('One.md')

    await one.refresh([{ from: 'One.md', to: 'Renamed.md' }])

    expect(plex.state.view.here.value).toBe('Renamed.md')
    expect(one.views[0]?.went.at(-1)).toBe('Renamed.md')
  })

  it('leaves the note under it called what the picture called it', async () => {
    const one = window()
    const plex = await one.openTab('One.md')
    const before = plex.state.picture.value?.nodes.map((node) => node.id)

    await one.refresh([{ from: 'One.md', to: 'Renamed.md' }])

    expect(plex.state.picture.value?.nodes.map((node) => node.id)).toStrictEqual(before)
  })

  it('leaves a plex standing on a note nothing moved', async () => {
    const one = window()
    const plex = await one.openTab('One.md')

    await one.refresh([{ from: 'Other.md', to: 'Renamed.md' }])

    expect(plex.state.view.here.value).toBe('One.md')
  })

  it('gives one standing nowhere the note an empty vault has just gained', async () => {
    const one = window('')
    const { state } = await one.openTab()
    one.setOpeningNote('First.md')

    await one.refresh()

    expect(state.view.here.value).toBe('First.md')
  })

  it('asks the vault where it opens once, however many stand nowhere', async () => {
    const one = window('')
    await one.openTab()
    await one.openTab()
    await one.openTab()

    await one.refresh()

    expect(one.asked).toHaveLength(1)
  })

  it('asks it not at all while every one of them is standing somewhere', async () => {
    const one = window()
    await one.openTab('One.md')
    await one.openTab('Two.md')

    await one.refresh()

    expect(one.asked).toStrictEqual([])
  })

  it('asks nothing for a plex whose tab has closed', async () => {
    const one = window()
    const first = await one.openTab('One.md')
    const second = await one.openTab('Two.md')
    one.closeTab(second.id)

    await one.refresh()

    expect(one.views[0]?.went).toContain('One.md')
    expect(one.views[1]?.went.filter((where) => where === 'Two.md')).toHaveLength(1)
    expect(first.state.view.here.value).toBe('One.md')
  })
})

describe('a plex tab the person has closed', () => {
  it('asks the vault nothing when a setting it once answered turns', async () => {
    const one = window()
    const plex = await one.openTab('One.md')
    one.isHanging.value = false
    await settle()

    one.closeTab(plex.id)
    one.headingsAsked.length = 0
    one.isHanging.value = true
    await settle()

    expect(one.headingsAsked).toStrictEqual([])
  })
})

describe('what a command asked over a plex tab is over', () => {
  it('is the note the plex is standing on, under the name the picture gives it', async () => {
    const one = window()
    const { state } = await one.openTab('physics/Ontology.md')

    expect(one.kind.getTarget!(state)).toStrictEqual({
      path: 'physics/Ontology.md',
      title: 'physics/Ontology',
    })
  })

  it('is no note at all while the plex stands nowhere', async () => {
    const one = window('')
    const { state } = await one.openTab()

    expect(one.kind.getTarget!(state)).toStrictEqual({ path: '', title: '' })
  })
})

describe('what a plex tab holds, as whoever answers for the person is told it', () => {
  it('is the note it is standing on', async () => {
    const one = window()
    const { state } = await one.openTab('Root.md')

    expect(one.kind.getOpenTab!(state)).toStrictEqual({ path: 'Root.md' })
  })
})

describe('what a plex tab is called', () => {
  it('is the note it stands on', async () => {
    const one = window()
    const { state } = await one.openTab('Root.md')

    expect(one.kind.getTitle?.(state)).toBe('Root')
  })

  it('is the word for a plex while it stands nowhere', async () => {
    const one = window('')
    const { state } = await one.openTab()

    expect(one.kind.getTitle?.(state)).toBe('Plex')
  })
})
