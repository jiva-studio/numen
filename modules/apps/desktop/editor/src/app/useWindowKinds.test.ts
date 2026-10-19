/**
 * The kinds a window registers, and what each is handed of the others.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useWindowKinds } from './useWindowKinds'

type MockSeen = {
  writeMessage: unknown
  dragged: unknown
  destinations: unknown
  about: () => unknown
}

const hoisted = vi.hoisted(() => ({
  seen: {} as Record<'agent' | 'files', MockSeen>,
  agents: {} as Record<string, unknown>,
}))

vi.mock('./kinds/agent', () => ({
  createAgentKind: (args: unknown) => {
    hoisted.seen.agent = args as MockSeen
    return hoisted.agents
  },
}))
vi.mock('./fileServices', () => ({
  createFileMakers: (args: unknown) => {
    hoisted.seen.files = args as MockSeen
    return 'made'
  },
  createVaultFiles: () => ({ name: 'files' }),
  createPathCommandHost: () => ({}),
}))

/** The plex tab the window holds, and what its kind says it is over. */
interface Plex {
  readonly target?: { path: string; title: string }
}

const create = ({ plex, hasPlexKind = true }: { plex?: Plex; hasPlexKind?: boolean } = {}) => {
  const writer = vi.fn()
  const deps = {
    log: { getWriter: vi.fn(() => writer) },
    tabOpeners: { openFileAt: vi.fn(), openFile: vi.fn(() => Promise.resolve()) },
    held: {
      registerKinds: vi.fn(),
      hasKind: vi.fn(() => hasPlexKind),
      navigateTab: vi.fn(() => Promise.resolve('plex:1')),
      navigateTabsAt: vi.fn(() => Promise.resolve()),
      handle: { last: vi.fn(() => (plex ? { id: 'plex:1', state: 'plex state' } : null)) },
      getTab: vi.fn(() => ({ kind: { getTarget: () => plex?.target } })),
    },
    editing: { kinds: ['note', 'deck'], making: {} },
    settings: { kinds: ['settings'], nesting: { isNesting: ref(true), headings: ref(2) } },
    window: { failure: ref(''), opening: ref('Root.md'), readInitialNote: vi.fn() },
    core: {},
    getTarget: vi.fn(() => ({})),
    runCommand: vi.fn(),
  } as unknown as Parameters<typeof useWindowKinds>[0]
  return { deps, writer, window: useWindowKinds(deps) }
}

beforeEach(() => {
  hoisted.seen = {} as Record<'agent' | 'files', MockSeen>
  hoisted.agents = { kind: 'agent', askQuestion: vi.fn(() => Promise.resolve()) }
})

describe('the window kinds', () => {
  it('writes command messages through the command writer of the log', () => {
    const { deps, writer } = create()

    expect(deps.log.getWriter).toHaveBeenCalledWith('command')
    expect(hoisted.seen.files.writeMessage).toBe(writer)
  })

  it('starts with nothing dragged', () => {
    const { window } = create()

    expect(window.dragged.value).toEqual([])
  })

  it('registers the kinds of the editors, the settings and each window kind, in that order', () => {
    const { deps } = create()

    expect(deps.held.registerKinds).toHaveBeenCalledWith(['note', 'deck', 'settings', 'agent'])
  })

  it('makes files through the makers it hands the commands, and reveals a path in the files tab', async () => {
    const { deps, window } = create()

    expect(window.made).toBe('made')
    expect(window.services.files).toStrictEqual({ name: 'files' })
    await window.files.revealPath('notes/a.md')
    expect(deps.held.navigateTab).toHaveBeenCalledWith('files', 'notes/a.md')
  })

  it('reveals nothing where the window has no files kind', async () => {
    const { deps, window } = create({ hasPlexKind: false })

    await window.files.revealPath('notes/a.md')

    expect(deps.held.navigateTab).not.toHaveBeenCalled()
  })
})

describe('what the agent is about', () => {
  it('is what the plex the person was last in is over', () => {
    create({ plex: { target: { path: 'Ideas/Plan.md', title: 'The plan' } } })

    expect(hoisted.seen.agent.about()).toEqual({ path: 'Ideas/Plan.md', title: 'The plan' })
  })

  it('is no note at all in a window holding no plex, or one over nothing', () => {
    create()
    expect(hoisted.seen.agent.about()).toEqual({ path: '', title: '' })

    create({ plex: {} })
    expect(hoisted.seen.agent.about()).toEqual({ path: '', title: '' })
  })
})

describe('a note travelled to', () => {
  it('is put in the plex the person was last in', async () => {
    const { deps, window } = create()

    await window.travel('a.md')

    expect(deps.held.navigateTab).toHaveBeenCalledWith('plex', 'a.md')
    expect(deps.tabOpeners.openFile).not.toHaveBeenCalled()
  })

  it('is opened in its editor where the window has no plex kind', async () => {
    const { deps, window } = create({ hasPlexKind: false })

    await window.destinations.travel('a.md')

    expect(deps.held.navigateTab).not.toHaveBeenCalled()
    expect(deps.tabOpeners.openFile).toHaveBeenCalledWith('a.md', '', 'here')
  })

  it('moves every tab standing on a note that left the vault', async () => {
    const { deps, window } = create()

    await window.leavePath('Gone.md', 'Root.md')

    expect(deps.held.navigateTabsAt).toHaveBeenCalledWith('Gone.md', 'Root.md')
  })
})

describe('the destinations', () => {
  it('open the file at the path with the one run', () => {
    const { deps, window } = create()
    const run = { from: 1, to: 4 }

    window.destinations.openFileAt('a.md', run)

    expect(deps.tabOpeners.openFileAt).toHaveBeenCalledWith('a.md', [run])
  })

  it('open the file in the tab in front, at the line', () => {
    const { deps, window } = create()

    window.destinations.openFile('a.md', 'A', 7)

    expect(deps.tabOpeners.openFile).toHaveBeenCalledWith('a.md', 'A', { where: 'here', line: 7 })
  })
})

describe('the services the window gives a plugin', () => {
  it('ask the agent, share what is dragged and stand on the note the vault opens with', () => {
    const { window, writer } = create()
    const { workspace, notes } = window.services

    workspace?.ask('what is this?')
    workspace?.setDraggedPaths(['a.md'])
    workspace?.showError('the link was not written')

    expect(hoisted.agents.askQuestion).toHaveBeenCalledWith('what is this?')
    expect(window.dragged.value).toStrictEqual(['a.md'])
    expect(writer).toHaveBeenCalledWith('the link was not written', 'error')
    expect(workspace?.openingPath.value).toBe('Root.md')
    expect(workspace?.headingDisplay.count.value).toBe(2)
    expect(notes?.creatableRelations).toStrictEqual(['parent', 'child', 'jump'])
  })
})
