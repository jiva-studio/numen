/**
 * The kinds a window registers, and what each is handed of the others.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useWindowKinds } from './useWindowKinds'

const h = vi.hoisted(() => ({
  seen: {} as Record<string, any>,
  plexes: {} as Record<string, any>,
  agents: {} as Record<string, any>,
}))

vi.mock('./kinds/plex', () => ({
  createPlexKind: (args: unknown) => {
    h.seen.plex = args
    return h.plexes
  },
}))
vi.mock('./kinds/agent', () => ({
  createAgentKind: (args: unknown) => {
    h.seen.agent = args
    return h.agents
  },
}))
vi.mock('./kinds/media', () => ({
  createMediaKinds: () => ({}),
}))
vi.mock('./kinds/files', () => ({
  createFilesKind: (args: unknown) => {
    h.seen.files = args
    return { files: { kind: 'files' }, made: 'made' }
  },
}))

const create = () => {
  const writer = vi.fn()
  const deps: any = {
    log: { getWriter: vi.fn(() => writer) },
    tabOpeners: { openFileAt: vi.fn(), openFile: vi.fn() },
    held: { registerKinds: vi.fn() },
    editing: { kinds: ['note', 'deck'] },
    settings: { kinds: ['settings'] },
  }
  return { deps, writer, window: useWindowKinds(deps) }
}

beforeEach(() => {
  h.seen = {}
  h.plexes = {
    kind: 'plex',
    looking: vi.fn(() => 'Ideas/Plan.md'),
    getName: vi.fn(() => 'The plan'),
    travel: vi.fn(),
  }
  h.agents = { kind: 'agent', askQuestion: vi.fn() }
})

describe('the window kinds', () => {
  it('writes command messages through the command writer of the log', () => {
    const { deps, writer } = create()

    expect(deps.log.getWriter).toHaveBeenCalledWith('command')
    expect(h.seen.plex.writeMessage).toBe(writer)
    expect(h.seen.files.writeMessage).toBe(writer)
  })

  it('starts with nothing dragged, shared by the plex and files kinds', () => {
    const { window } = create()

    expect(window.dragged.value).toEqual([])
    expect(h.seen.plex.dragged).toBe(window.dragged)
    expect(h.seen.files.dragged).toBe(window.dragged)
  })

  it('registers the kinds of the editors, the settings and each window kind, in that order', () => {
    const { deps } = create()

    expect(deps.held.registerKinds).toHaveBeenCalledWith([
      'note',
      'deck',
      'settings',
      'plex',
      'agent',
      'files',
    ])
  })

  it('hands the files kind the destinations it returns', () => {
    const { window } = create()

    expect(h.seen.files.destinations).toBe(window.destinations)
    expect(window.made).toBe('made')
  })
})

describe('asking the agent from the plex', () => {
  it('puts the text to the agent as a question', () => {
    create()

    h.seen.plex.askAgent('what is this?')

    expect(h.agents.askQuestion).toHaveBeenCalledWith('what is this?')
  })
})

describe('what the agent is about', () => {
  it('is the plex path under the name the plex gives it', () => {
    create()

    expect(h.seen.agent.about()).toEqual({ path: 'Ideas/Plan.md', title: 'The plan' })
    expect(h.plexes.getName).toHaveBeenCalledWith('Ideas/Plan.md')
  })

  it('is titled by the path when the plex gives it no name', () => {
    create()
    h.plexes.getName = vi.fn(() => '')

    expect(h.seen.agent.about()).toEqual({ path: 'Ideas/Plan.md', title: 'Ideas/Plan.md' })
  })
})

describe('the destinations', () => {
  it('travel moves the plex to the path', () => {
    const { window } = create()

    window.destinations.travel('a.md')

    expect(h.plexes.travel).toHaveBeenCalledWith('a.md')
  })

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
