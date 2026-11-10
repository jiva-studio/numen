/**
 * The tabs a window keeps open and the detach of a tab.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreHandlers } from '@numen/core-client/testing'
import { tabSessionCore } from './tabSession'

const request = {
  writeOpenTabs: vi.fn(),
  detachTab: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  vi.clearAllMocks()
  restore = installCoreHandlers(request)
})

afterEach(() => {
  restore()
})

describe('the tabs a window holds open', () => {
  it('are written as they stand, with the one in front', async () => {
    const tab = { id: 't1', kind: 'note', path: 'Llama.md' }

    await tabSessionCore.writeOpenTabs({ tabs: [tab], activeId: 't1' } as never)

    expect(request.writeOpenTabs.mock.lastCall?.[0]).toMatchObject({ tabs: [tab], activeId: 't1' })
  })
})

describe('detaching a tab', () => {
  it('sends DetachTab with the tab, its saved state and the rounded position and size', async () => {
    request.detachTab.mockResolvedValueOnce({ error: 0 })
    request.detachTab.mockResolvedValueOnce({ error: 2 })
    const tab = {
      id: 'plex:3b9d',
      kind: 'plex',
      path: 'Physics.md',
      title: 'Physics',
      state: '{"focus":"Kepler"}',
    }
    expect(
      await tabSessionCore.detachTab(tab, { x: 100.6, y: 200.4, width: 640.5, height: 479.6 }),
    ).toStrictEqual({
      ok: true,
      value: undefined,
    })
    expect(await tabSessionCore.detachTab(tab, { x: 0, y: 0, width: 0, height: 0 })).toStrictEqual({
      ok: false,
      error: 'timeout',
    })

    expect(request.detachTab.mock.calls[0]?.[0]).toMatchObject({
      tab,
      x: 101,
      y: 200,
      width: 641,
      height: 480,
    })
  })
})
