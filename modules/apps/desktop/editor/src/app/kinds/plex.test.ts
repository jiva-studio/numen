/**
 * What a plex tab is handed by the window: a note it opens or stands on goes
 * to the tabs linked to the plex tab the gesture came from, and the rest is
 * the vault and the window answering.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import type { PlexTabDeps, PlexTabState } from '@plugins/view-plex-graph/frontend'
import { createPlexKind, type PlexKindDeps } from './plex'

const built = vi.hoisted(() => ({ deps: null as unknown }))

vi.mock('./plexKind', () => ({
  createPlexKind: (_handle: unknown, _createView: unknown, deps: unknown) => {
    built.deps = deps
    return {}
  },
}))

vi.mock('@plugins/view-plex-graph/frontend', () => ({
  usePlexView: () => ({}),
}))

const searched = vi.hoisted(() => ({ asked: [] as [string, number][] }))

vi.mock('../vault/search', () => ({
  searchCore: {
    names: async (query: string, limit: number) => {
      searched.asked.push([query, limit])
      return [{ path: 'Compost.md', title: 'Compost', heading: 'Leaves', line: 3, at: [] }]
    },
  },
}))

/** A plex tab as the window holds one. */
interface HeldTab {
  readonly id: string
  readonly kind: string
  readonly state: PlexTabState
}

const createPlexState = (name: string) => ({ name }) as unknown as PlexTabState

const first = createPlexState('first')
const second = createPlexState('second')
const third = createPlexState('third')

/** Three plex tabs, the last of them in front. */
const THREE: readonly HeldTab[] = [
  { id: 'plex:1', kind: 'plex', state: first },
  { id: 'plex:2', kind: 'plex', state: second },
  { id: 'plex:3', kind: 'plex', state: third },
]

const plexKind = (
  more: {
    tabs?: readonly HeldTab[]
    front?: HeldTab | null
    isLinked?: boolean
    hasTabSync?: boolean
    created?: { path: string } | null
  } = {},
) => {
  const tabs = more.tabs ?? THREE
  const synced: [string, string, string][] = []
  const opened: [string, string, string, number | undefined][] = []
  const written: [string, string][] = []
  const headingsAsked: (readonly string[])[] = []
  const untitled: [string, readonly string[]][] = []
  const headings = new Map([['Compost.md', [{ text: 'Leaves', level: 1, line: 3 }]]])

  createPlexKind({
    core: {
      headings: async (paths: readonly string[]) => {
        headingsAsked.push(paths)
        return headings
      },
    },
    tabOpeners: {
      openFile: async (
        path: string,
        title: string,
        options?: string | { where?: string; line?: number },
      ) => {
        const showing = typeof options === 'string' ? options : (options?.where ?? 'here')
        const line = typeof options === 'object' ? options?.line : undefined
        opened.push([path, title, showing, line])
      },
    },
    held: {
      handle: {
        each: (kind: string) => tabs.filter((tab) => tab.kind === kind),
        front: () => (more.front === undefined ? (tabs.at(-1) ?? null) : more.front),
        last: (kind: string) => tabs.filter((tab) => tab.kind === kind).at(-1),
      },
    },
    editing: {
      making: {
        createUntitled: async (path: string, parts: readonly string[]) => {
          untitled.push([path, parts])
          return more.created === undefined ? { path: 'Untitled.md' } : more.created
        },
      },
    },
    settings: { nesting: { isNesting: ref(true), headings: ref(6) } },
    window: {
      failure: ref(''),
      opening: ref(''),
      readInitialNote: async () => 'Opening.md',
    },
    getTarget: () => ({}),
    runCommand: () => {},
    dragged: shallowRef([]),
    writeMessage: (text: string, tone: string) => written.push([text, tone]),
    askAgent: () => {},
    tabLinks: {},
    tabSync:
      more.hasTabSync === false
        ? undefined
        : {
            syncPathFromTab: (id: string, path: string, title: string) => {
              synced.push([id, path, title])
              return more.isLinked ?? true
            },
          },
  } as unknown as PlexKindDeps)

  return {
    deps: built.deps as PlexTabDeps,
    synced,
    opened,
    written,
    headingsAsked,
    headings,
    untitled,
  }
}

describe('a note a plex tab opens', () => {
  it('goes to the tabs linked to the plex tab whose state asked', () => {
    const one = plexKind()

    one.deps.openNote('Compost.md', { title: 'Compost', showing: 'here', state: second })

    expect(one.synced).toStrictEqual([['plex:2', 'Compost.md', 'Compost']])
    expect(one.opened).toStrictEqual([])
  })

  it('goes to the tabs linked to the plex tab in front where no tab holds that state', () => {
    const one = plexKind()

    one.deps.openNote('Compost.md', {
      title: 'Compost',
      showing: 'here',
      state: createPlexState('gone'),
    })

    expect(one.synced).toStrictEqual([['plex:3', 'Compost.md', 'Compost']])
  })

  it('opens in a tab of its own where no tab is linked', () => {
    const one = plexKind({ isLinked: false })

    one.deps.openNote('Compost.md', {
      title: 'Compost',
      showing: 'beside',
      line: 7,
      state: second,
    })

    expect(one.opened).toStrictEqual([['Compost.md', 'Compost', 'beside', 7]])
  })

  it('opens in a tab of its own where the window links no tabs', () => {
    const one = plexKind({ hasTabSync: false })

    one.deps.openNote('Compost.md', { title: 'Compost', showing: 'here', state: second })

    expect(one.opened).toStrictEqual([['Compost.md', 'Compost', 'here', undefined]])
  })

  it('opens a note when called without options', () => {
    const one = plexKind({ hasTabSync: false })

    one.deps.openNote('Compost.md')

    expect(one.opened).toStrictEqual([['Compost.md', '', 'here', undefined]])
  })

  it('goes to the tabs linked to the plex tab in front where none was named', () => {
    const one = plexKind({ front: THREE[0]! })

    one.deps.openNote('Compost.md', { title: 'Compost', showing: 'here' })

    expect(one.synced).toStrictEqual([['plex:1', 'Compost.md', 'Compost']])
  })

  it('goes to the tabs linked to the last plex tab where another kind of tab is in front', () => {
    const one = plexKind({ front: { id: 'note:1', kind: 'note', state: first } })

    one.deps.openNote('Compost.md', { title: 'Compost', showing: 'here' })

    expect(one.synced).toStrictEqual([['plex:3', 'Compost.md', 'Compost']])
  })
})

describe('a note a plex tab opens where a tab holds no state', () => {
  it('goes to the plex tab in front, whatever tab holds no state', () => {
    const one = plexKind({
      tabs: [{ id: 'plex:0', kind: 'plex', state: undefined as unknown as PlexTabState }, ...THREE],
    })

    one.deps.openNote('Compost.md', { title: 'Compost', showing: 'here' })

    expect(one.synced).toStrictEqual([['plex:3', 'Compost.md', 'Compost']])
  })
})

describe('a note a plex tab stands on', () => {
  it('goes to the tabs linked to the plex tab whose state asked, under its title', () => {
    const one = plexKind()

    one.deps.onSelectPath?.('Compost.md', 'Compost', second)

    expect(one.synced).toStrictEqual([['plex:2', 'Compost.md', 'Compost']])
  })

  it('goes untitled where it has no title', () => {
    const one = plexKind()

    one.deps.onSelectPath?.('Compost.md', undefined, second)

    expect(one.synced).toStrictEqual([['plex:2', 'Compost.md', '']])
  })

  it('goes nowhere where the window holds no plex tab', () => {
    const one = plexKind({ tabs: [] })

    one.deps.onSelectPath?.('Compost.md', 'Compost', second)

    expect(one.synced).toStrictEqual([])
  })

  it('goes nowhere where the window links no tabs', () => {
    const one = plexKind({ hasTabSync: false })

    expect(() => one.deps.onSelectPath?.('Compost.md', 'Compost', second)).not.toThrow()
  })
})

describe('what the window answers a plex tab', () => {
  it('reads the headings of the notes the plex asks about from the vault', async () => {
    const one = plexKind()

    expect(await one.deps.readHeadings(['Compost.md'])).toBe(one.headings)
    expect(one.headingsAsked).toStrictEqual([['Compost.md']])
  })

  it('reads the note the window opened with', async () => {
    expect(await plexKind().deps.readOpeningPath()).toBe('Opening.md')
  })

  it('writes a message as an error', () => {
    const one = plexKind()

    one.deps.showMessage('No link was written')

    expect(one.written).toStrictEqual([['No link was written', 'error']])
  })

  it('makes an untitled note at the top of the vault and answers its path', async () => {
    const one = plexKind()

    expect(await one.deps.createUntitledNote()).toBe('Untitled.md')
    expect(one.untitled).toStrictEqual([['', []]])
  })

  it('answers no path where the vault made no note', async () => {
    expect(await plexKind({ created: null }).deps.createUntitledNote()).toBe('')
  })

  it('finds notes by name, as a path and a title each', async () => {
    searched.asked.length = 0

    expect(await plexKind().deps.searchNames?.('comp')).toStrictEqual([
      { path: 'Compost.md', title: 'Compost' },
    ])
    expect(searched.asked).toStrictEqual([['comp', 20]])
  })
})
