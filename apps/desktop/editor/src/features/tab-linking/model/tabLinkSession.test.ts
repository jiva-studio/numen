import { describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import { createLostConnection } from '@numen/core-client/testing'
import { createAnsweredApi } from '@/testing/coreAnswers'
import type { TabLinksEvent } from '@/entities/tab'
import { createTabLinkSession, getWindowReport, type TabLinkSessionDeps } from './tabLinkSession'
import { createTabReports } from './tabReports'

const SNAPSHOT = {
  groups: [
    {
      id: 'group-1',
      colorIndex: 1,
      tabs: ['plex:1', 'plex:2'],
      path: 'Orbits.md',
      title: 'Orbits',
    },
  ],
  tabs: [
    { id: 'plex:1', kind: 'plex', title: 'Orbits', windowId: 'window-a' },
    { id: 'plex:2', kind: 'plex', title: 'Tides', windowId: 'window-b' },
  ],
  windows: [],
}
const SYNC = { groupId: 'group-1', sourceTabId: 'plex:2', path: 'Tides.md', title: 'Tides' }
const OPEN_TABS = [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }]
const WINDOW = { isMain: true, activeTitle: 'Orbits', activeKind: 'plex', tabCount: 1 }
const MOVED_TAB = {
  tab: {
    id: 'plex:2',
    kind: 'plex',
    path: 'notes/Tides.md',
    title: 'Tides',
    state: '{"focus":"Moon"}',
  },
  moveId: 'move-1',
}

const FAILURE = createLostConnection('the core is gone')
const isFailure = expect.objectContaining({
  message: expect.stringContaining('the core is gone'),
})

const toWire = (event: TabLinksEvent) => {
  if (event.links) {
    const { groups, tabs, windows } = event.links
    return {
      event: {
        case: 'links' as const,
        value: {
          groups: groups.map(({ tabs: tabIds, ...rest }) => ({ ...rest, tabIds: [...tabIds] })),
          tabs: [...tabs],
          windows: [...windows],
        },
      },
    }
  }
  if (event.sync) return { event: { case: 'sync' as const, value: event.sync } }
  if (event.movedTab) return { event: { case: 'movedTab' as const, value: event.movedTab } }
  if (event.isHighlighted !== undefined) {
    return { event: { case: 'highlight' as const, value: { isOn: event.isHighlighted } } }
  }
  return { event: { case: undefined } }
}

async function* createStream(events: readonly TabLinksEvent[], signal: AbortSignal) {
  yield* events.map(toWire)
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
    writeTabLinks: vi.fn((_report: { windowId: string; tabs?: unknown; window?: unknown }) =>
      Promise.resolve(),
    ),
  }
  const api = createAnsweredApi({
    workspace: {
      watchTabLinks: (request, context) => core.watchTabLinks(request.windowId, context.signal),
      cancelWatchTabLinks: async (request) => {
        await core.cancelTabLinksWatch(request.windowId)
        return {}
      },
      writeTabLinks: async ({ windowId, change }) => {
        const report = change.case === 'tabs' ? change.value : undefined
        await core.writeTabLinks({
          windowId,
          tabs: report?.tabs.map(({ id, kind, title }) => ({ id, kind, title })),
          window: report?.window && {
            isMain: report.window.isMain,
            activeTitle: report.window.activeTitle,
            activeKind: report.window.activeKind,
            tabCount: report.window.tabCount,
          },
        })
        return {}
      },
    },
  })
  const wait = vi.fn((_ms: number) => Promise.resolve())
  const onError = vi.fn()
  const deps = {
    api: api.tabLinks,
    reports: createTabReports({
      writeTabLinks: api.tabLinks.writeTabLinks,
      windowId: 'window-a',
      getReport: () => ({ tabs: getOpenTabs(), window: WINDOW }),
      wait,
      onError,
    }),
    windowId: 'window-a',
    tabLinks: { applySnapshot: vi.fn() },
    applySync: vi.fn(),
    applyGroupPaths: vi.fn(),
    openMovedTab: vi.fn(),
    setHighlighted: vi.fn(),
    wait,
    onError,
  } satisfies TabLinkSessionDeps
  return { deps, core, signals, session: createTabLinkSession(deps) }
}

const REPORT = { windowId: 'window-a', tabs: OPEN_TABS, window: WINDOW }

describe('createTabLinkSession', () => {
  it('applies snapshots and syncs, and reports the open tabs when the stream opens', async () => {
    const { deps, core, session } = setup([{ links: SNAPSHOT }, {}, { sync: SYNC }])

    const stream = session.start()

    await vi.waitFor(() => expect(deps.applySync).toHaveBeenCalledWith(SYNC))
    expect(core.watchTabLinks).toHaveBeenCalledWith('window-a', expect.any(AbortSignal))
    expect(deps.tabLinks.applySnapshot).toHaveBeenCalledExactlyOnceWith(SNAPSHOT)
    expect(core.writeTabLinks).toHaveBeenCalledExactlyOnceWith(REPORT)
    session.close()
    await stream
  })

  it('reports the open tabs again when the stream reopens', async () => {
    const { core, session } = setup([{ links: SNAPSHOT }])
    core.watchTabLinks.mockImplementationOnce(async function* () {
      yield toWire({ links: SNAPSHOT })
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
    core.writeTabLinks.mockRejectedValueOnce(FAILURE)

    session.reportTabs()

    await vi.waitFor(() => expect(deps.onError).toHaveBeenCalledWith(isFailure))
  })

  it('keeps the stream open when applySync throws', async () => {
    const { deps, core, session } = setup([{ sync: SYNC }, { links: SNAPSHOT }])
    const failure = new Error('navigate failed')
    deps.applySync.mockImplementationOnce(() => {
      throw failure
    })

    const stream = session.start()

    await vi.waitFor(() => expect(deps.tabLinks.applySnapshot).toHaveBeenCalledWith(SNAPSHOT))
    expect(deps.onError).toHaveBeenCalledWith(failure)
    expect(core.watchTabLinks).toHaveBeenCalledOnce()
    session.close()
    await stream
  })
})

describe('getWindowReport', () => {
  it('returns the tabs whose kind supports navigate, with kind and title', () => {
    const kinds: Record<string, { kind: string; address?: { navigate: () => void } }> = {
      'plex:1': { kind: 'plex', address: { navigate: () => undefined } },
      'agent:1': { kind: 'agent' },
    }
    const windowTabs = {
      tabs: {
        tabs: computed(() => [
          { id: 'agent:1', title: 'Agent' },
          { id: 'plex:1', title: 'Orbits' },
          { id: 'gone:1', title: 'Gone' },
        ]),
        getTab: (id: string) => (kinds[id] ? { kind: kinds[id], state: null } : null),
      },
      groups: { handle: { front: () => null } },
    } as unknown as Parameters<typeof getWindowReport>[0]

    expect(getWindowReport(windowTabs, true).tabs).toStrictEqual([
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
    expect(core.writeTabLinks.mock.calls.map(([report]) => report)).toStrictEqual([
      {
        windowId: 'window-a',
        tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }],
        window: WINDOW,
      },
      {
        windowId: 'window-a',
        tabs: [
          { id: 'plex:1', kind: 'plex', title: 'Orbits' },
          { id: 'plex:2', kind: 'plex', title: 'Tides' },
        ],
        window: WINDOW,
      },
    ])
    session.close()
  })

  it('sends later reports after a failed one', async () => {
    const { deps, core, session } = setup([])
    core.writeTabLinks.mockRejectedValueOnce(FAILURE)

    session.reportTabs()
    session.reportTabs()

    await vi.waitFor(() => expect(core.writeTabLinks).toHaveBeenCalledTimes(2))
    expect(deps.onError).toHaveBeenCalledExactlyOnceWith(isFailure)
  })
})

describe('a snapshot', () => {
  it('moves the tabs of this window to the path of each group', async () => {
    const { deps, session } = setup([{ links: SNAPSHOT }])

    const stream = session.start()

    await vi.waitFor(() => expect(deps.applyGroupPaths).toHaveBeenCalledWith(SNAPSHOT.groups))
    session.close()
    await stream
  })
})

describe('a failed tab report', () => {
  it('is sent again after the retry delay', async () => {
    const { deps, core, session } = setup([])
    core.writeTabLinks.mockRejectedValueOnce(FAILURE)

    session.reportTabs()

    await vi.waitFor(() => expect(core.writeTabLinks).toHaveBeenCalledTimes(2))
    expect(deps.wait).toHaveBeenCalledWith(2000)
    expect(core.writeTabLinks.mock.calls[1]).toStrictEqual([REPORT])
  })

  it('is replaced by a newer report made during the retry delay', async () => {
    const tabs = [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }]
    const { deps, core, session } = setup([], () => [...tabs])
    core.writeTabLinks.mockRejectedValueOnce(FAILURE)
    let endDelay = (): void => undefined
    deps.wait.mockImplementationOnce(() => new Promise<void>((resolve) => (endDelay = resolve)))

    session.reportTabs()
    await vi.waitFor(() => expect(deps.wait).toHaveBeenCalledOnce())
    tabs.push({ id: 'plex:2', kind: 'plex', title: 'Tides' })
    session.reportTabs()
    endDelay()

    await vi.waitFor(() => expect(core.writeTabLinks).toHaveBeenCalledTimes(2))
    expect(core.writeTabLinks.mock.calls[1]?.[0]).toStrictEqual({
      windowId: 'window-a',
      tabs: [
        { id: 'plex:1', kind: 'plex', title: 'Orbits' },
        { id: 'plex:2', kind: 'plex', title: 'Tides' },
      ],
      window: WINDOW,
    })
  })

  it('is not sent again after close', async () => {
    const { deps, core, session } = setup([])
    core.writeTabLinks.mockRejectedValueOnce(FAILURE)
    let endDelay = (): void => undefined
    deps.wait.mockImplementationOnce(() => new Promise<void>((resolve) => (endDelay = resolve)))

    session.reportTabs()
    await vi.waitFor(() => expect(deps.wait).toHaveBeenCalledOnce())
    session.close()
    endDelay()
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(core.writeTabLinks).toHaveBeenCalledOnce()
  })
})

describe('reopen', () => {
  it('starts a new stream after close and reports the tabs again', async () => {
    const { core, session, signals } = setup([{ links: SNAPSHOT }])
    const first = session.start()
    await vi.waitFor(() => expect(core.writeTabLinks).toHaveBeenCalledOnce())
    session.close()
    await first

    const second = session.reopen()

    await vi.waitFor(() => expect(core.writeTabLinks).toHaveBeenCalledTimes(2))
    expect(signals).toHaveLength(2)
    expect(signals[1]?.aborted).toBe(false)
    session.close()
    await second
  })

  it('does nothing while the session is open', async () => {
    const { core, session } = setup([])

    await session.reopen()

    expect(core.watchTabLinks).not.toHaveBeenCalled()
  })
})

describe('stream events for this window', () => {
  it('opens a moved tab and sets the highlight from the stream', async () => {
    const { deps, session } = setup([
      { movedTab: MOVED_TAB },
      { isHighlighted: true },
      { isHighlighted: false },
    ])

    const stream = session.start()

    await vi.waitFor(() => expect(deps.setHighlighted).toHaveBeenCalledTimes(2))
    expect(deps.openMovedTab).toHaveBeenCalledExactlyOnceWith(MOVED_TAB)
    expect(deps.setHighlighted.mock.calls).toStrictEqual([[true], [false]])
    session.close()
    await stream
  })

  it('settles waitForReports after the pending tab report is written', async () => {
    const { core, session } = setup([])
    let finishReport: () => void = () => undefined
    core.writeTabLinks.mockImplementationOnce(
      () => new Promise<void>((resolve) => (finishReport = resolve)),
    )
    let isSettled = false

    session.reportTabs()
    const reports = session.waitForReports().then(() => (isSettled = true))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(isSettled).toBe(false)
    finishReport()
    await reports

    expect(isSettled).toBe(true)
  })
})

describe('getWindowReport', () => {
  it('returns the active tab title and kind and the tab count', () => {
    const kinds: Record<string, { kind: string; address?: { navigate: () => void } }> = {
      'plex:1': { kind: 'plex', address: { navigate: () => undefined } },
      'agent:1': { kind: 'agent' },
    }
    const windowTabs = {
      tabs: {
        tabs: computed(() => [
          { id: 'agent:1', title: 'Agent' },
          { id: 'plex:1', title: 'Orbits' },
        ]),
        getTab: (id: string) => (kinds[id] ? { kind: kinds[id], state: null } : null),
      },
      groups: { handle: { front: () => ({ id: 'agent:1', kind: 'agent', state: null }) } },
    } as unknown as Parameters<typeof getWindowReport>[0]

    expect(getWindowReport(windowTabs, false)).toStrictEqual({
      tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }],
      window: { isMain: false, activeTitle: 'Agent', activeKind: 'agent', tabCount: 2 },
    })
  })

  it('returns an empty active tab when no tab is active', () => {
    const windowTabs = {
      tabs: { tabs: computed(() => []), getTab: () => null },
      groups: { handle: { front: () => null } },
    } as unknown as Parameters<typeof getWindowReport>[0]

    expect(getWindowReport(windowTabs, true).window).toStrictEqual({
      isMain: true,
      activeTitle: '',
      activeKind: '',
      tabCount: 0,
    })
  })
})
