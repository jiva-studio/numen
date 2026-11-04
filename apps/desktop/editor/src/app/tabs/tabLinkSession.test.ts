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

/** A stream that yields the events, then waits until the signal aborts. */
async function* createStream(events: readonly TabLinksEvent[], signal: AbortSignal) {
  yield* events
  await new Promise((resolve) => signal.addEventListener('abort', resolve))
}

const setup = (events: readonly TabLinksEvent[]) => {
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
    getOpenTabs: () => OPEN_TABS,
    wait: () => Promise.resolve(),
    onError: vi.fn(),
  } satisfies TabLinkSessionDeps
  return { deps, core, signals, session: createTabLinkSession(deps) }
}

describe('the tab links session', () => {
  it('mirrors each snapshot, applies each sync, and reports the open tabs once the stream opens', async () => {
    const { deps, core, session } = setup([{ links: SNAPSHOT }, {}, { sync: SYNC }])

    const reading = session.start()

    await vi.waitFor(() => expect(deps.applySync).toHaveBeenCalledWith(SYNC))
    expect(core.watchTabLinks).toHaveBeenCalledWith('window-a', expect.any(AbortSignal))
    expect(deps.tabLinks.applySnapshot).toHaveBeenCalledExactlyOnceWith(SNAPSHOT)
    expect(core.writeTabLinks).toHaveBeenCalledExactlyOnceWith('window-a', { tabs: OPEN_TABS })
    session.close()
    await reading
  })

  it('reports the open tabs again on a stream opened after the first ended', async () => {
    const { core, session } = setup([{ links: SNAPSHOT }])
    core.watchTabLinks.mockImplementationOnce(async function* () {
      yield { links: SNAPSHOT }
    })

    const reading = session.start()

    await vi.waitFor(() => expect(core.writeTabLinks).toHaveBeenCalledTimes(2))
    expect(core.watchTabLinks).toHaveBeenCalledTimes(2)
    session.close()
    await reading
  })

  it('aborts the stream and cancels the watch of the window when it closes, once', async () => {
    const { core, session, signals } = setup([])
    const reading = session.start()
    await vi.waitFor(() => expect(signals).toHaveLength(1))

    session.close()
    session.close()
    await reading

    expect(signals[0]?.aborted).toBe(true)
    expect(core.cancelTabLinksWatch).toHaveBeenCalledExactlyOnceWith('window-a')
  })

  it('reports a report the core did not take', async () => {
    const { deps, core, session } = setup([])
    const refused = new Error('the core is gone')
    core.writeTabLinks.mockRejectedValueOnce(refused)

    session.reportTabs()

    await vi.waitFor(() => expect(deps.onError).toHaveBeenCalledWith(refused))
  })
})

describe('the linkable tabs of the window', () => {
  it('are the tabs whose kind can move to a path, with their kind and title', () => {
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
