/**
 * Tests the mapping of tab links requests and stream to the workspace service.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { tabLinksCore } from './tabLinks'

const request = vi.hoisted(() => ({
  watchTabLinks: vi.fn(),
  cancelWatchTabLinks: vi.fn(),
  writeTabLinks: vi.fn(),
  syncTabPath: vi.fn(),
}))

vi.mock('@/shared/clients', () => ({ workspace: request }))

beforeEach(() => {
  vi.clearAllMocks()
})

async function* createResponses(responses: readonly unknown[]) {
  yield* responses
}

describe('watchTabLinks', () => {
  it('maps a snapshot, a sync and a keepalive', async () => {
    request.watchTabLinks.mockReturnValue(
      createResponses([
        {
          event: {
            case: 'links',
            value: {
              groups: [{ id: 'group-1', colorIndex: 2, tabIds: ['plex:1', 'files:2'] }],
              tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits', windowId: 'window-a' }],
            },
          },
        },
        {
          event: {
            case: 'sync',
            value: {
              groupId: 'group-1',
              sourceTabId: 'plex:1',
              path: 'Orbits.md',
              title: 'Orbits',
            },
          },
        },
        { event: { case: undefined } },
      ]),
    )
    const signal = new AbortController().signal

    const events = []
    for await (const event of tabLinksCore.watchTabLinks('window-a', signal)) events.push(event)

    expect(request.watchTabLinks).toHaveBeenCalledWith({ windowId: 'window-a' }, { signal })
    expect(events).toStrictEqual([
      {
        links: {
          groups: [{ id: 'group-1', colorIndex: 2, tabs: ['plex:1', 'files:2'] }],
          tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits', windowId: 'window-a' }],
        },
      },
      { sync: { groupId: 'group-1', sourceTabId: 'plex:1', path: 'Orbits.md', title: 'Orbits' } },
      {},
    ])
  })

  it('cancels the watch by window id', async () => {
    request.cancelWatchTabLinks.mockResolvedValue({})

    await tabLinksCore.cancelTabLinksWatch('window-a')

    expect(request.cancelWatchTabLinks).toHaveBeenCalledWith({ windowId: 'window-a' })
  })
})

describe('writeTabLinks', () => {
  it('writes a link, an unlink or the open tabs', async () => {
    request.writeTabLinks.mockResolvedValue({})

    await tabLinksCore.writeTabLinks('window-a', { link: { source: 'plex:1', target: 'files:2' } })
    await tabLinksCore.writeTabLinks('window-a', { unlink: 'plex:1' })
    await tabLinksCore.writeTabLinks('window-a', {
      tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }],
    })

    expect(request.writeTabLinks.mock.calls).toStrictEqual([
      [
        {
          windowId: 'window-a',
          change: { case: 'link', value: { sourceTabId: 'plex:1', targetTabId: 'files:2' } },
        },
      ],
      [{ windowId: 'window-a', change: { case: 'unlinkTabId', value: 'plex:1' } }],
      [
        {
          windowId: 'window-a',
          change: {
            case: 'tabs',
            value: { tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }] },
          },
        },
      ],
    ])
  })

  it('sends a sync of a path', async () => {
    request.syncTabPath.mockResolvedValue({})
    const sync = { sourceTabId: 'plex:1', path: 'Orbits.md', title: 'Orbits' }

    await tabLinksCore.syncTabPath(sync)

    expect(request.syncTabPath).toHaveBeenCalledWith(sync)
  })
})
