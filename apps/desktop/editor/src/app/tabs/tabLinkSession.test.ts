import { describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import type { TabLinksEvent } from '../ports/vault'
import { createTabLinkSession, getLinkableTabs, type TabLinkSessionDeps } from './tabLinkSession'

const SNAPSHOT = {
  groups: [{ id: 'group-1', colorIndex: 1, tabs: ['plex:1', 'plex:2'] }],
  tabs: [
    { id: 'plex:1', kind: 'plex', title: 'Orbits', windowId: 'window-a' },
    { id: 'plex:2', kind: 'plex', title: 'Tides', windowId: 'window-b' },
  ],
}
const SYNC = { groupId: 'group-1', sourceTabId: 'plex:2', path: 'Tides.md', title: 'Tides' }
const OPEN_TABS = [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }]

/** Returns a stream that yields the events, then waits until the signal aborts. */
async function* createStream(events: readonly TabLinksEvent[], signal: AbortSignal) {
  yield* events
  if (!signal.aborted) await new Promise((resolve) => signal.addEventListener('abort', resolve))
}

const setup = (events: readonly TabLinksEvent[], getOpenTabs = () => OPEN_TABS) => {
  const signals: AbortSignal[] = []
  const core = {
    watchTabLinks: vi.fn((_windowId: string, signal: AbortSignal) => {
      signals.push(signal)
      return createStream(events, signal)
    }),
    cancelTabLinksWatch: vi.fn((_windowId: string) => Promise.resolve()),
    writeTabLinks: vi.fn((_windowId: string, _change: unknown) => Promise.resolve()),
  }
  const deps = {
    core,
    windowId: 'window-a',
    tabLinks: { applySnapshot: vi.fn() },
    applySync: vi.fn(),
    getOpenTabs,
    wait: () => Promise.resolve(),
    onError: vi.fn(),
  } satisfies TabLinkSessionDeps
  return { deps, core, signals, session: createTabLinkSession(deps) }
}

describe('createTabLinkSession', () => {
  it('applies snapshots and syncs, and reports the open tabs when the stream opens', async () => {
    const { deps, core, session } = setup([{ links: SNAPSHOT }, {}, { sync: SYNC }])

    const stream = session.start()

    await vi.waitFor(() => expect(deps.applySync).toHaveBeenCalledWith(SYNC))
    expect(core.watchTabLinks).toHaveBeenCalledWith('window-a', expect.any(AbortSignal))
    expect(deps.tabLinks.applySnapshot).toHaveBeenCalledExactlyOnceWith(SNAPSHOT)
    expect(core.writeTabLinks).toHaveBeenCalledExactlyOnceWith('window-a', { tabs: OPEN_TABS })
    session.close()
    await stream
  })

  it('reports the open tabs again when the stream reopens', async () => {
    const { core, session } = setup([{ links: SNAPSHOT }])
    core.watchTabLinks.mockImplementationOnce(async function* () {
      yield { links: SNAPSHOT }
    })

    const stream = session.start()

    await vi.waitFor(() => expect(core.writeTabLinks).toHaveBeenCalledTimes(2))
    expect(core.watchTabLinks).toHaveBeenCalledTimes(2)
    session.close()
    await stream
  })

  it('aborts the stream and cancels the watch once on close', async () => {
    const { core, session, signals } = setup([])
    const stream = session.start()
    await vi.waitFor(() => expect(signals).toHaveLength(1))

    session.close()
    session.close()
    await stream

    expect(signals[0]?.aborted).toBe(true)
    expect(core.cancelTabLinksWatch).toHaveBeenCalledExactlyOnceWith('window-a')
  })

  it('reports an error when the core rejects the tab report', async () => {
    const { deps, core, session } = setup([])
    const failure = new Error('the core is gone')
    core.writeTabLinks.mockRejectedValueOnce(failure)

    session.reportTabs()

    await vi.waitFor(() => expect(deps.onError).toHaveBeenCalledWith(failure))
  })
})

describe('getLinkableTabs', () => {
  it('returns the tabs whose kind supports navigate, with kind and title', () => {
    const kinds: Record<string, { kind: string; navigate?: () => void }> = {
      'plex:1': { kind: 'plex', navigate: () => undefined },
      'agent:1': { kind: 'agent' },
    }
    const windowTabs = {
      tabs: computed(() => [
        { id: 'agent:1', title: 'Agent' },
        { id: 'plex:1', title: 'Orbits' },
        { id: 'gone:1', title: 'Gone' },
      ]),
      getTab: (id: string) => (kinds[id] ? { kind: kinds[id], state: null } : null),
    } as unknown as Parameters<typeof getLinkableTabs>[0]

    expect(getLinkableTabs(windowTabs)).toStrictEqual([
      { id: 'plex:1', kind: 'plex', title: 'Orbits' },
    ])
  })
})

describe('reportTabs', () => {
  it('writes reports in order, each with the tabs open at call time', async () => {
    const tabs = [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }]
    const { core, session } = setup([], () => [...tabs])
    let finishFirst: () => void = () => undefined
    core.writeTabLinks.mockImplementationOnce(
      () => new Promise<void>((resolve) => (finishFirst = resolve)),
    )

    session.reportTabs()
    tabs.push({ id: 'plex:2', kind: 'plex', title: 'Tides' })
    session.reportTabs()
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(core.writeTabLinks).toHaveBeenCalledOnce()
    finishFirst()

    await vi.waitFor(() => expect(core.writeTabLinks).toHaveBeenCalledTimes(2))
    expect(core.writeTabLinks.mock.calls.map(([, change]) => change)).toStrictEqual([
      { tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }] },
      {
        tabs: [
          { id: 'plex:1', kind: 'plex', title: 'Orbits' },
          { id: 'plex:2', kind: 'plex', title: 'Tides' },
        ],
      },
    ])
    session.close()
  })

  it('sends later reports after a failed one', async () => {
    const { deps, core, session } = setup([])
    const failure = new Error('the core is gone')
    core.writeTabLinks.mockRejectedValueOnce(failure)

    session.reportTabs()
    session.reportTabs()

    await vi.waitFor(() => expect(core.writeTabLinks).toHaveBeenCalledTimes(2))
    expect(deps.onError).toHaveBeenCalledExactlyOnceWith(failure)
  })
})
