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
  built: {} as Record<'files' | 'vaultFiles', MockSeen & { making?: unknown }>,
}))

vi.mock('../plugins/agentPort', () => ({
  agentPort: { ask: vi.fn(), finish: vi.fn(() => Promise.resolve()) },
}))
vi.mock('../plugins/fileServices', () => ({
  createFileMakers: (args: unknown) => {
    hoisted.built.files = args as MockSeen
    return 'made'
  },
  createVaultFiles: (args: unknown) => {
    hoisted.built.vaultFiles = args as MockSeen & { making?: unknown }
    return { name: 'files' }
  },
  createIconHost: () => ({}),
}))

/** A tool tab that follows a note, and the note its kind is attached to. */
interface Graph {
  readonly target?: { path: string; title: string }
}

/** A tab kind that accepts text. */
const CHAT = { kind: 'chat', isTool: true, receiveText: vi.fn() }

const create = ({ graph, hasChat = true }: { graph?: Graph; hasChat?: boolean } = {}) => {
  const writer = vi.fn()
  const graphKind = {
    kind: 'graph',
    isTool: true,
    navigate: vi.fn(),
    getTarget: () => graph?.target,
  }
  const open = new Map<string, { kind: object; state: string }>([
    ['graph:1', { kind: graphKind, state: 'graph state' }],
    ['chat:1', { kind: CHAT, state: 'chat state' }],
  ])
  const deps = {
    log: { getWriter: vi.fn(() => writer) },
    tabOpeners: {
      openFileAt: vi.fn(),
      openFile: vi.fn(() => Promise.resolve()),
      findFileIcon: vi.fn((path: string) => (path.endsWith('.pdf') ? 'pdf-icon' : undefined)),
    },
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
    editing: { making: {}, fileFlush: { addHandler: vi.fn(), raise: vi.fn() } },
    settings: { kinds: ['settings'] },
    window: { failure: ref(''), opening: ref('Root.md'), readInitialNote: vi.fn() },
    core: { relations: { roles: ['parent', 'child', 'jump'] } },
    getTarget: vi.fn(() => ({})),
    runCommand: vi.fn(),
  } as unknown as Parameters<typeof useWindowKinds>[0]
  return { deps, writer, window: useWindowKinds(deps) }
}

beforeEach(() => {
  hoisted.built = {} as Record<'files' | 'vaultFiles', MockSeen>
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
    const { deps, window } = create({
      graph: { target: { path: 'Ideas/Plan.md', title: 'The plan' } },
    })
    const earlierGraph = {
      kind: 'graph',
      isTool: true,
      navigate: vi.fn(),
      getTarget: () => ({ path: 'Old.md', title: 'Old' }),
    }
    const nonToolWithNav = {
      kind: 'note',
      isTool: false,
      navigate: vi.fn(),
      getTarget: () => ({ path: 'Note.md', title: 'Note' }),
    }
    const toolWithoutGetTarget = {
      kind: 'tool-no-target',
      isTool: true,
      navigate: vi.fn(),
    }
    ;(deps.windowTabs.tabs as { value: unknown }).value = [
      { id: 'graph:0' },
      { id: 'graph:1' },
      { id: 'note:1' },
      { id: 'tool:no-target' },
      { id: 'missing:1' },
      { id: 'chat:1' },
    ]
    const origImpl = vi.mocked(deps.windowTabs.getTab).getMockImplementation()!
    vi.mocked(deps.windowTabs.getTab).mockImplementation((id: string) => {
      if (id === 'graph:0') return { kind: earlierGraph, state: 's0' } as never
      if (id === 'note:1') return { kind: nonToolWithNav, state: 's1' } as never
      if (id === 'tool:no-target') return { kind: toolWithoutGetTarget, state: 's2' } as never
      if (id === 'missing:1') return null as never
      return origImpl(id)
    })

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
  it('is opened in its editor', async () => {
    const { deps, window } = create()

    await window.travel('a.md')

    expect(deps.tabOpeners.openFile).toHaveBeenCalledWith('a.md', '', 'current')
  })

  it('moves every tab standing on a note that left the vault', async () => {
    const { deps, window } = create()

    await window.leavePath('Gone.md', 'Root.md')

    expect(deps.windowTabs.navigateTabsAt).toHaveBeenCalledWith(
      'Gone.md',
      'Root.md',
      expect.any(Function),
    )
  })

  it('records the new path of each moved tab in the tab sync', async () => {
    const { deps, window } = create()
    const setShownPath = vi.fn()
    Object.assign(deps, { tabSync: { setShownPath } })
    vi.mocked(deps.windowTabs.navigateTabsAt).mockImplementationOnce((_from, _to, onMove) => {
      onMove?.('graph:1')
      return Promise.resolve()
    })

    await window.leavePath('Gone.md', 'Root.md')

    expect(setShownPath).toHaveBeenCalledExactlyOnceWith('graph:1', 'Root.md')
  })
})

describe('the destinations', () => {
  it('give the icon of the reader that opens a file', () => {
    const { deps, window } = create()

    expect(window.destinations.findFileIcon?.('doc.pdf')).toBe('pdf-icon')
    expect(deps.tabOpeners.findFileIcon).toHaveBeenCalledWith('doc.pdf')
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
    const { workspace } = window.services

    workspace?.ask('what is this?')
    workspace?.setDraggedPaths(['a.md'])
    workspace?.showError('the link was not written')
    workspace?.getMessageWriter('form')('named a form')

    expect(deps.windowTabs.sendText).toHaveBeenCalledWith('chat', 'what is this?')
    expect(window.dragged.value).toStrictEqual(['a.md'])
    expect(writer).toHaveBeenCalledWith('the link was not written', 'error')
    expect(deps.log.getWriter).toHaveBeenCalledWith('form')
    expect(workspace?.getToday()).toBe('2026-09-04')
    expect(workspace?.openingPath.value).toBe('Root.md')
    expect(window.services.relations?.roles).toStrictEqual(['parent', 'child', 'jump'])
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
    const { deps, window } = create()

    expect(window.services.agent).toBeDefined()
    await expect(window.services.agent?.finish('conversation:1')).resolves.toBeUndefined()

    const flushFn = vi.fn(() => Promise.resolve())
    window.services.buffers?.addFlushHandler(flushFn)
    expect(deps.editing.fileFlush.addHandler).toHaveBeenCalledWith(flushFn)
    expect(hoisted.built.vaultFiles.making).toBe(deps.editing.making)
    expect(window.services.relations?.roles).toStrictEqual(['parent', 'child', 'jump'])
    ;(deps.core as unknown as { headings: ReturnType<typeof vi.fn> }).headings = vi.fn(
      async () => new Map([['a.md', []]]),
    )
    await expect(window.services.notes?.getHeadings(['a.md'])).resolves.toEqual(
      new Map([['a.md', []]]),
    )
  })
})
