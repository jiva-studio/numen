import { describe, expect, it, vi } from 'vitest'
import { ReportedError } from '@numen/result'
import { WIRE_ERROR_CODE } from '@numen/core-client'
import type { CoreAnswers } from '@numen/core-client/testing'
import type { AnyTabKind } from '@/entities/tab'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createTestTabs } from '@/testing/tabs'
import { createMovedTabStarter } from './createMovedTabStarter'
import { createWindowTabMover } from './createWindowTabMover'

type HighlightWindow = NonNullable<NonNullable<CoreAnswers['workspace']>['highlightWindow']>

const setup = (asking = false, highlightWindow?: (requests: string[]) => HighlightWindow) => {
  const requests: string[] = []
  const gates: (() => void)[] = []
  const api = createAnsweredApi({
    workspace: {
      highlightWindow:
        highlightWindow?.(requests) ??
        (({ windowId, isOn }) => {
          requests.push(`${windowId}:${isOn ? 'on' : 'off'}`)
          return new Promise<Record<string, never>>((resolve) => gates.push(() => resolve({})))
        }),
    },
  })
  const mover = createWindowTabMover({
    api: api.tabs.tabMove,
    windowId: 'window-a',
    windowTabs: createTestTabs(),
    notes: { isAsking: () => asking, flushPending: () => Promise.resolve() },
    closeTab: () => true,
    writeMessage: vi.fn(),
  })
  return { mover, requests, gates }
}

describe('the outline a window offers over another', () => {
  it('sends the requests one at a time, so an off cannot pass the on it ends', async () => {
    const { mover, requests, gates } = setup()

    mover.highlightWindow('window-b', true)
    mover.highlightWindow('window-b', false)
    await vi.waitFor(() => expect(requests).toStrictEqual(['window-b:on']))
    gates[0]?.()

    await vi.waitFor(() => expect(requests).toStrictEqual(['window-b:on', 'window-b:off']))
  })

  it('goes on after a request that failed', async () => {
    const { mover, requests } = setup(false, (asked) => ({ isOn }) => {
      asked.push(isOn ? 'on' : 'off')
      if (isOn) throw new Error('window gone')
      return {}
    })

    mover.highlightWindow('window-b', true)
    mover.highlightWindow('window-b', false)

    await vi.waitFor(() => expect(requests).toStrictEqual(['on', 'off']))
  })
})

describe('the tabs of a window', () => {
  it('say whether a tab waits for an answer, as the notes do', () => {
    expect(setup(true).mover.isTabAsking('plex:1')).toBe(true)
    expect(setup(false).mover.isTabAsking('plex:1')).toBe(false)
  })

  it('detaches and moves tabs in the background and reports starter errors', async () => {
    const calls: string[] = []
    const api = createAnsweredApi({
      workspace: {
        detachTab: ({ tab, x, y }) => {
          calls.push(`detach:${tab?.id}:${x}:${y}`)
          return { error: WIRE_ERROR_CODE.UNSPECIFIED }
        },
        moveTab: ({ tab, targetWindowId }) => {
          calls.push(`move:${tab?.id}:${targetWindowId}`)
          return { error: WIRE_ERROR_CODE.UNSPECIFIED }
        },
      },
    })
    const windowTabs = createTestTabs()
    windowTabs.kinds.registerKinds([
      {
        kind: 'note',
        open: () => ({ title: 'Tides' }),
        pane: {},
        getTitle: (state: { title: string }) => state.title,
      } as AnyTabKind,
    ])
    const id1 = await windowTabs.tabs.openTabOfKind('note', 'A.md')
    const id2 = await windowTabs.tabs.openTabOfKind('note', 'B.md')
    const id3 = await windowTabs.tabs.openTabOfKind('note', 'C.md')

    const mover = createWindowTabMover({
      api: api.tabs.tabMove,
      windowId: 'window-a',
      windowTabs,
      notes: { isAsking: () => false, flushPending: () => Promise.resolve() },
      closeTab: () => true,
      writeMessage: vi.fn(),
    })

    mover.detachTab(id1, { x: 10, y: 20 })
    mover.detachTabInPlace(id2)
    mover.moveTab(id3, 'window-b')

    await vi.waitFor(() =>
      expect(calls).toStrictEqual([
        `detach:${id1}:10:20`,
        `detach:${id2}:0:0`,
        `move:${id3}:window-b`,
      ]),
    )

    const writeMessage = vi.fn()
    const startMoved = createMovedTabStarter({
      windowTabs: {
        tabs: windowTabs.tabs,
        restored: {
          openSavedTab: () => {
            throw new ReportedError('bad tab')
          },
        },
      } as never,
      closeTab: () => true,
      acceptMove: async () => ({ ok: true, value: undefined }),
      focusWindow: async () => undefined,
      writeMessage,
    })
    startMoved({
      moveId: 'move-1',
      tab: { id: 'note:9', kind: 'note', path: 'A.md', title: 'A', state: '' },
    })
    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith('The moved tab did not open: bad tab', 'error'),
    )
  })
})
