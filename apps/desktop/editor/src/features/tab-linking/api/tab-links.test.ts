/**
 * Tests the mapping of tab links requests and stream to the workspace service.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createAnsweredClients } from '@/testing/coreAnswers'
import { createTabLinksApi } from './tab-links'

const request = {
  watchTabLinks: vi.fn(),
  cancelWatchTabLinks: vi.fn(),
  writeTabLinks: vi.fn(),
  syncTabPath: vi.fn(),
}

const tabLinksCore = createTabLinksApi(createAnsweredClients({ workspace: request }))

const ORBITS = {
  id: 'plex:1',
  kind: 'plex',
  path: 'notes/Orbits.md',
  title: 'Orbits',
  state: '{"focus":"Kepler"}',
}
const MAIN_WINDOW = {
  isMain: true,
  activeTitle: 'Orbits',
  activeKind: 'plex',
  tabCount: 2,
}

beforeEach(() => {
  vi.clearAllMocks()
})

async function* createResponses(responses: readonly unknown[]) {
  yield* responses
}

describe('watchTabLinks', () => {
  it('maps a snapshot, a sync, a moved tab, a highlight and a keepalive', async () => {
    request.watchTabLinks.mockReturnValue(
      createResponses([
        {
          event: {
            case: 'links',
            value: {
              groups: [
                {
                  id: 'group-1',
                  colorIndex: 2,
                  tabIds: ['plex:1', 'files:2'],
                  path: 'Orbits.md',
                  title: 'Orbits',
                },
              ],
              tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits', windowId: 'window-a' }],
              windows: [{ id: 'window-a', ...MAIN_WINDOW }],
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
        {
          event: {
            case: 'movedTab',
            value: {
              tab: { ...ORBITS, $typeName: 'numen.v1.Tab' },
              moveId: 'move-1',
              $typeName: 'numen.v1.MovedTab',
            },
          },
        },
        { event: { case: 'highlight', value: { isOn: true } } },
        { event: { case: undefined } },
      ]),
    )
    const signal = new AbortController().signal

    const events = []
    for await (const event of tabLinksCore.watchTabLinks('window-a', signal)) events.push(event)

    expect(request.watchTabLinks.mock.lastCall?.[0]).toMatchObject({ windowId: 'window-a' })
    expect(events).toStrictEqual([
      {
        links: {
          groups: [
            {
              id: 'group-1',
              colorIndex: 2,
              tabs: ['plex:1', 'files:2'],
              path: 'Orbits.md',
              title: 'Orbits',
            },
          ],
          tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits', windowId: 'window-a' }],
          windows: [{ id: 'window-a', ...MAIN_WINDOW }],
        },
      },
      { sync: { groupId: 'group-1', sourceTabId: 'plex:1', path: 'Orbits.md', title: 'Orbits' } },
      { movedTab: { tab: ORBITS, moveId: 'move-1' } },
      { isHighlighted: true },
      {},
    ])
  })

  it('cancels the watch by window id', async () => {
    request.cancelWatchTabLinks.mockResolvedValue({})

    await tabLinksCore.cancelTabLinksWatch('window-a')

    expect(request.cancelWatchTabLinks.mock.lastCall?.[0]).toMatchObject({ windowId: 'window-a' })
  })
})

describe('writeTabLinks', () => {
  it('writes a link, an unlink or the open tabs', async () => {
    request.writeTabLinks.mockResolvedValue({})

    await tabLinksCore.writeTabLinks('window-a', { link: { source: 'plex:1', target: 'files:2' } })
    await tabLinksCore.writeTabLinks('window-a', { unlink: 'plex:1' })
    await tabLinksCore.writeTabLinks('window-a', {
      tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }],
      window: MAIN_WINDOW,
    })

    expect(request.writeTabLinks.mock.calls.map(([sent]) => sent)).toMatchObject([
      {
        windowId: 'window-a',
        change: { case: 'link', value: { sourceTabId: 'plex:1', targetTabId: 'files:2' } },
      },
      { windowId: 'window-a', change: { case: 'unlinkTabId', value: 'plex:1' } },
      {
        windowId: 'window-a',
        change: {
          case: 'tabs',
          value: { tabs: [{ id: 'plex:1', kind: 'plex', title: 'Orbits' }], window: MAIN_WINDOW },
        },
      },
    ])
  })

  it('sends a sync of a path', async () => {
    request.syncTabPath.mockResolvedValue({})
    const sync = { sourceTabId: 'plex:1', path: 'Orbits.md', title: 'Orbits' }

    await tabLinksCore.syncTabPath(sync)

    expect(request.syncTabPath.mock.lastCall?.[0]).toMatchObject(sync)
  })
})
