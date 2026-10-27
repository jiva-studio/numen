/**
 * The kinds a window registers, and what each is handed of the others.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useWindowKinds } from './useWindowKinds'

type MockSeen = {
  writeMessage: unknown
  dragged: unknown
  destinations: unknown
}

const hoisted = vi.hoisted(() => ({
  built: {} as Record<'files', MockSeen>,
}))

vi.mock('../plugins/agentPort', () => ({
  agentPort: { ask: vi.fn(), finish: vi.fn(() => Promise.resolve()) },
}))
vi.mock('../plugins/fileServices', () => ({
  createFileMakers: (args: unknown) => {
    hoisted.built.files = args as MockSeen
    return 'made'
  },
  createVaultFiles: () => ({ name: 'files' }),
  createPathCommandHost: () => ({}),
}))

/** A tool tab that follows a note, and what its kind says it is over. */
interface Graph {
  readonly target?: { path: string; title: string }
}

/** A kind of tab that takes text put to it. */
const CHAT = { kind: 'chat', isTool: true, receiveText: () => {} }

const create = ({
  graph,
  hasChat = true,
  showsNotes = true,
}: { graph?: Graph; hasChat?: boolean; showsNotes?: boolean } = {}) => {
  const writer = vi.fn()
  const graphKind = {
    kind: 'graph',
    isTool: true,
    navigate: () => {},
    getTarget: () => graph?.target,
  }
  const open = new Map<string, { kind: object; state: string }>([
    ['graph:1', { kind: graphKind, state: 'graph state' }],
    ['chat:1', { kind: CHAT, state: 'chat state' }],
  ])
  const deps = {
    log: { getWriter: vi.fn(() => writer) },
    tabOpeners: { openFileAt: vi.fn(), openFile: vi.fn(() => Promise.resolve()) },
    windowTabs: {
      registerKinds: vi.fn(),
      sendText: vi.fn(() => Promise.resolve('chat:1')),
      findKind: vi.fn((isWanted: (kind: object) => boolean) =>
        [graphKind, ...(hasChat ? [CHAT] : [])].find(isWanted),
      ),
      navigateTabsAt: vi.fn(() => Promise.resolve()),
      tabs: ref(graph ? [{ id: 'graph:1' }, { id: 'chat:1' }] : [{ id: 'chat:1' }]),
      getTab: vi.fn((id: string) => open.get(id)),
    },
    showNote: vi.fn(() => Promise.resolve(showsNotes)),
    getNoteCommands: () => [],
    editing: { making: {}, fileFlush: { addHandler: vi.fn(), raise: vi.fn() } },
    settings: { kinds: ['settings'] },
    window: { failure: ref(''), opening: ref('Root.md'), readInitialNote: vi.fn() },
    core: {},
    getTarget: vi.fn(() => ({})),
    runCommand: vi.fn(),
  } as unknown as Parameters<typeof useWindowKinds>[0]
  return { deps, writer, window: useWindowKinds(deps) }
}

beforeEach(() => {
  hoisted.built = {} as Record<'files', MockSeen>
})

describe('the window kinds', () => {
  it('writes command messages through the command writer of the log', () => {
    const { deps, writer } = create()

    expect(deps.log.getWriter).toHaveBeenCalledWith('command')
    expect(hoisted.built.files.writeMessage).toBe(writer)
  })

  it('starts with nothing dragged', () => {
    const { window } = create()

    expect(window.dragged.value).toEqual([])
  })

  it('registers the kinds of the settings', () => {
    const { deps } = create()

    expect(deps.windowTabs.registerKinds).toHaveBeenCalledWith(['settings'])
  })

  it('makes files through the makers it passes the commands', () => {
    const { window } = create()

    expect(window.made).toBe('made')
    expect(window.services.files).toStrictEqual({ name: 'files' })
  })
})

describe('the note the window relies on', () => {
  it('is what the tool that follows a note the person was last in is over', () => {
    const { window } = create({ graph: { target: { path: 'Ideas/Plan.md', title: 'The plan' } } })

    expect(window.services.workspace?.getNoteInFocus()).toEqual({
      path: 'Ideas/Plan.md',
      title: 'The plan',
    })
  })

  it('is no note at all in a window holding no such tool, or one over nothing', () => {
    expect(create().window.services.workspace?.getNoteInFocus()).toEqual({ path: '', title: '' })
    expect(create({ graph: {} }).window.services.workspace?.getNoteInFocus()).toEqual({
      path: '',
      title: '',
    })
  })
})

describe('a note travelled to', () => {
  it('is put in front by the command a plugin declares for it', async () => {
    const { deps, window } = create()

    await window.travel('a.md')

    expect(deps.showNote).toHaveBeenCalledWith('a.md')
    expect(deps.tabOpeners.openFile).not.toHaveBeenCalled()
  })

  it('is opened in its editor where no plugin declares one', async () => {
    const { deps, window } = create({ showsNotes: false })

    await window.destinations.travel('a.md')

    expect(deps.tabOpeners.openFile).toHaveBeenCalledWith('a.md', '', 'here')
  })

  it('moves every tab standing on a note that left the vault', async () => {
    const { deps, window } = create()

    await window.leavePath('Gone.md', 'Root.md')

    expect(deps.windowTabs.navigateTabsAt).toHaveBeenCalledWith('Gone.md', 'Root.md')
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
  afterEach(() => {
    vi.useRealTimers()
  })

  it('ask the agent, share what is dragged and stand on the note the vault opens with', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 8, 4, 3, 30))
    const { deps, window, writer } = create()
    const { workspace, notes } = window.services

    workspace?.ask('what is this?')
    workspace?.setDraggedPaths(['a.md'])
    workspace?.showError('the link was not written')
    workspace?.getMessageWriter('stencil')('named a stencil')

    expect(deps.windowTabs.sendText).toHaveBeenCalledWith('chat', 'what is this?')
    expect(window.dragged.value).toStrictEqual(['a.md'])
    expect(writer).toHaveBeenCalledWith('the link was not written', 'error')
    expect(deps.log.getWriter).toHaveBeenCalledWith('stencil')
    expect(workspace?.getToday()).toBe('2026-09-04')
    expect(workspace?.openingPath.value).toBe('Root.md')
    expect(notes?.creatableRelations).toStrictEqual(['parent', 'child', 'jump'])
  })

  it('tell the person why a question could not be put to the agent', async () => {
    const { deps, window, writer } = create()
    vi.mocked(deps.windowTabs.sendText).mockRejectedValueOnce(new Error('no agent tab'))

    window.services.workspace?.ask('what is this?')
    await Promise.resolve()
    await Promise.resolve()

    expect(writer).toHaveBeenCalledWith(expect.any(String), 'error')
  })

  it('tell the person a question cannot be put where no kind of tab takes text', async () => {
    const { deps, window, writer } = create({ hasChat: false })

    window.services.workspace?.ask('what is this?')
    await Promise.resolve()

    expect(deps.windowTabs.sendText).not.toHaveBeenCalled()
    expect(writer).toHaveBeenCalledWith(expect.stringContaining('not available'), 'error')
  })

  it('offer the agent the window asks, over the port it talks on', async () => {
    const { window } = create()

    expect(window.services.agent).toBeDefined()
    await expect(window.services.agent?.finish('conversation:1')).resolves.toBeUndefined()
  })
})
