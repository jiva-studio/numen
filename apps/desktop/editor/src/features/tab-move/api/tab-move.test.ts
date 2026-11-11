/**
 * Tab moves: the requests that send a tab to another window and the answers they get.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreAnswers } from '@numen/core-client/testing'
import * as tabMoveCore from './tab-move'

const request = {
  readDetachedTab: vi.fn(),
  moveTab: vi.fn(),
  acceptTabMove: vi.fn(),
  detachTab: vi.fn(),
  highlightWindow: vi.fn(),
}

const ORBITS = {
  id: 'plex:1',
  kind: 'plex',
  path: 'notes/Orbits.md',
  title: 'Orbits',
  state: '{"focus":"Kepler"}',
}

let restore: () => void

beforeEach(() => {
  vi.clearAllMocks()
  restore = installCoreAnswers({ workspace: request })
})

afterEach(() => {
  restore()
})

describe('tab move requests', () => {
  it('reads a detached tab by window and id, and returns null when none is pending', async () => {
    request.readDetachedTab.mockResolvedValueOnce({
      tab: { ...ORBITS, $typeName: 'numen.v1.Tab' },
      moveId: 'move-1',
    })
    request.readDetachedTab.mockResolvedValueOnce({})

    expect(await tabMoveCore.readDetachedTab('window-b', 'plex:1')).toStrictEqual({
      tab: ORBITS,
      moveId: 'move-1',
    })
    expect(await tabMoveCore.readDetachedTab('window-b', 'plex:9')).toBeNull()
    expect(request.readDetachedTab.mock.calls[0]?.[0]).toMatchObject({
      windowId: 'window-b',
      tabId: 'plex:1',
    })
  })

  it('sends MoveTab and converts the error code of the response', async () => {
    request.moveTab.mockResolvedValueOnce({ error: 0 })
    request.moveTab.mockResolvedValueOnce({ error: 1 })
    request.moveTab.mockResolvedValueOnce({ error: 2 })
    request.moveTab.mockResolvedValueOnce({ error: 3 })
    request.moveTab.mockResolvedValueOnce({ error: 4 })

    expect(await tabMoveCore.moveTab('window-a', 'window-b', ORBITS)).toStrictEqual({
      ok: true,
      value: undefined,
    })
    expect(await tabMoveCore.moveTab('window-a', 'window-b', ORBITS)).toStrictEqual({
      ok: false,
      error: 'window-closed',
    })
    expect(await tabMoveCore.moveTab('window-a', 'window-b', ORBITS)).toStrictEqual({
      ok: false,
      error: 'timeout',
    })
    expect(await tabMoveCore.moveTab('window-a', 'window-b', ORBITS)).toStrictEqual({
      ok: false,
      error: 'not-owner',
    })
    expect(await tabMoveCore.moveTab('window-a', 'window-b', ORBITS)).toStrictEqual({
      ok: false,
      error: 'cancelled',
    })
    expect(request.moveTab.mock.lastCall?.[0]).toMatchObject({
      windowId: 'window-a',
      targetWindowId: 'window-b',
      tab: ORBITS,
    })
  })

  it('sends AcceptTabMove and converts the error code of the response', async () => {
    request.acceptTabMove.mockResolvedValueOnce({ error: 2 })

    expect(await tabMoveCore.acceptTabMove('window-b', 'plex:1', 'move-1')).toStrictEqual({
      ok: false,
      error: 'timeout',
    })
    expect(request.acceptTabMove.mock.lastCall?.[0]).toMatchObject({
      windowId: 'window-b',
      tabId: 'plex:1',
      moveId: 'move-1',
    })
  })

  it('sends DetachTab with the tab, its saved state and the rounded position and size', async () => {
    request.detachTab.mockResolvedValueOnce({ error: 0 })
    request.detachTab.mockResolvedValueOnce({ error: 2 })

    expect(
      await tabMoveCore.detachTab(ORBITS, { x: 100.6, y: 200.4, width: 640.5, height: 479.6 }),
    ).toStrictEqual({ ok: true, value: undefined })
    expect(await tabMoveCore.detachTab(ORBITS, { x: 0, y: 0, width: 0, height: 0 })).toStrictEqual({
      ok: false,
      error: 'timeout',
    })

    expect(request.detachTab.mock.calls[0]?.[0]).toMatchObject({
      tab: ORBITS,
      x: 101,
      y: 200,
      width: 641,
      height: 480,
    })
  })
})

describe('highlighting a window', () => {
  it('sends HighlightWindow for the target window', async () => {
    request.highlightWindow.mockResolvedValue({})

    await tabMoveCore.highlightWindow('window-b', true)

    expect(request.highlightWindow.mock.lastCall?.[0]).toMatchObject({
      windowId: 'window-b',
      isOn: true,
    })
  })
})
