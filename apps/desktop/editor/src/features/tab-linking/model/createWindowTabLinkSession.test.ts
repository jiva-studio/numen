import { describe, expect, it, vi } from 'vitest'
import { ReportedError } from '@/shared/api/errors'
import { nextTick } from 'vue'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '@/entities/tab'
import { createTabLinking } from './createTabLinking'
import { createWindowTabLinkSession } from './createWindowTabLinkSession'

const linkable: AnyTabKind = {
  kind: 'plex',
  open: (path: string) => ({ title: path }),
  pane: {},
  getTitle: (state: { title: string }) => state.title,
  address: { navigate: () => undefined },
}

const setup = (events: readonly object[] = []) => {
  const reports: { windowId: string; tabs: readonly { id: string; title: string }[] }[] = []
  const cancelled: string[] = []
  const watched = vi.fn()
  const api = createAnsweredApi({
    workspace: {
      watchTabLinks: async function* (_request, context) {
        watched()
        yield* events as never[]
        await new Promise((resolve) => context.signal.addEventListener('abort', resolve))
      },
      cancelWatchTabLinks: ({ windowId }) => {
        cancelled.push(windowId)
        return {}
      },
      writeTabLinks: ({ windowId, change }) => {
        if (change.case === 'tabs') reports.push({ windowId, tabs: change.value.tabs })
        return {}
      },
    },
  })
  const windowTabs = createTestTabs()
  windowTabs.kinds.registerKinds([linkable])
  const writeMessage = vi.fn()
  const { tabLinks, reports: tabReports } = createTabLinking({
    writeTabLinks: api.tabs.tabLinks.writeTabLinks,
    windowId: 'window-a',
    windowTabs,
    isMain: true,
    writeMessage,
  })
  const session = createWindowTabLinkSession({
    api: api.tabs.tabLinks,
    windowId: 'window-a',
    windowTabs,
    isMain: true,
    tabLinks,
    reports: tabReports,
    tabSync: { applySync: vi.fn(), applyGroupPaths: vi.fn() },
    openMovedTab: vi.fn(),
    writeMessage,
  })
  return { session, windowTabs, reports, cancelled, watched, writeMessage }
}

describe('the tab link session of a window', () => {
  it('reports the linkable tabs again whenever they change', async () => {
    const { windowTabs, reports } = setup()

    const id = await windowTabs.tabs.openTabOfKind('plex', 'Tides.md')
    await nextTick()

    await vi.waitFor(() =>
      expect(reports.at(-1)).toStrictEqual({
        windowId: 'window-a',
        tabs: [expect.objectContaining({ id, title: 'Tides.md' })],
      }),
    )
  })

  it('is outlined while another window offers it a tab, and not after', async () => {
    let releaseSecond!: () => void
    const { windowTabs, writeMessage } = setup()
    const api = createAnsweredApi({
      workspace: {
        watchTabLinks: async function* (_request, context) {
          yield { event: { case: 'highlight' as const, value: { isOn: true } } }
          await new Promise<void>((resolve) => {
            releaseSecond = resolve
          })
          yield { event: { case: 'highlight' as const, value: { isOn: false } } }
          await new Promise((resolve) => context.signal.addEventListener('abort', resolve))
        },
      },
    })
    const { tabLinks, reports } = createTabLinking({
      writeTabLinks: api.tabs.tabLinks.writeTabLinks,
      windowId: 'window-a',
      windowTabs,
      isMain: true,
      writeMessage,
    })
    const session = createWindowTabLinkSession({
      api: api.tabs.tabLinks,
      windowId: 'window-a',
      windowTabs,
      isMain: true,
      tabLinks,
      reports,
      tabSync: { applySync: vi.fn(), applyGroupPaths: vi.fn() },
      openMovedTab: vi.fn(),
      writeMessage,
    })

    expect(session.isHighlighted.value).toBe(false)
    const stream = session.start()
    await vi.waitFor(() => expect(session.isHighlighted.value).toBe(true))
    releaseSecond()
    await vi.waitFor(() => expect(session.isHighlighted.value).toBe(false))
    session.close()
    await stream
  })

  it('cancels its watch in the core when the page is hidden', async () => {
    const { cancelled } = setup()

    globalThis.dispatchEvent(new Event('pagehide'))

    await vi.waitFor(() => expect(cancelled).toContain('window-a'))
  })

  it('watches again when a page restored from the cache is shown', async () => {
    const { watched, cancelled } = setup()
    globalThis.dispatchEvent(new Event('pagehide'))
    await vi.waitFor(() => expect(cancelled.length).toBeGreaterThan(0))

    globalThis.dispatchEvent(Object.assign(new Event('pageshow'), { persisted: true }))

    await vi.waitFor(() => expect(watched).toHaveBeenCalled())
  })

  it('does not watch again when a page that was not cached is shown', async () => {
    const { watched, cancelled } = setup()
    globalThis.dispatchEvent(new Event('pagehide'))
    await vi.waitFor(() => expect(cancelled.length).toBeGreaterThan(0))

    globalThis.dispatchEvent(Object.assign(new Event('pageshow'), { persisted: false }))
    await new Promise((resolve) => setTimeout(resolve, 20))

    expect(watched).not.toHaveBeenCalled()
  })

  it('writes link changes after reports and reports errors from linking, reports and cancelling', async () => {
    const windowTabs = createTestTabs()
    windowTabs.kinds.registerKinds([linkable])
    const writeMessage = vi.fn()
    const writeTabLinks = vi
      .fn<Parameters<typeof createTabLinking>[0]['writeTabLinks']>()
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(new ReportedError('link failed'))
      .mockRejectedValueOnce(new ReportedError('report failed'))

    const { tabLinks, reports } = createTabLinking({
      writeTabLinks,
      windowId: 'window-a',
      windowTabs,
      isMain: true,
      writeMessage,
    })

    tabLinks.linkTabs('a', 'b')
    await vi.waitFor(() =>
      expect(writeTabLinks).toHaveBeenCalledWith('window-a', {
        link: { source: 'a', target: 'b' },
      }),
    )

    tabLinks.linkTabs('a', 'c')
    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith('The tabs were not linked: link failed', 'error'),
    )

    reports.reportTabs()
    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith(
        'The tab links were not written: report failed',
        'error',
      ),
    )
    reports.stop()

    const session = createWindowTabLinkSession({
      api: {
        watchTabLinks: () => {
          throw new ReportedError('stream failed')
        },
        cancelTabLinksWatch: () => Promise.reject(new ReportedError('cancel failed')),
      },
      windowId: 'window-a',
      windowTabs,
      isMain: true,
      tabLinks,
      reports,
      tabSync: { applySync: vi.fn(), applyGroupPaths: vi.fn() },
      openMovedTab: vi.fn(),
      writeMessage,
    })

    session.close()
    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith(
        'The tab links were not written: cancel failed',
        'error',
      ),
    )
  })
})
