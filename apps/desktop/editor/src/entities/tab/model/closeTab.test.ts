import { describe, expect, it, vi } from 'vitest'
import { getPanes } from '@numen/workbench'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '../lib/kinds'

const onScreen = (windowTabs: ReturnType<typeof createTestTabs>) =>
  getPanes(windowTabs.panes.layout.value.root).flatMap((pane) => pane.tabs)

const kind = (onClose: () => boolean): AnyTabKind => ({
  kind: 'form',
  open: (path: string) => ({ title: path }),
  pane: {},
  getTitle: (state: { title: string }) => state.title,
  events: { onClose },
})

describe('closing a tab', () => {
  it('removes a tab its kind releases', async () => {
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([kind(() => true)])
    const id = await windowTabs.tabs.openTabOfKind('form', 'Tides')

    expect(windowTabs.closeTab(id)).toBe(true)

    expect(onScreen(windowTabs)).toStrictEqual([])
  })

  it('keeps a tab its kind holds, and runs the hold instead', async () => {
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([kind(() => false)])
    const id = await windowTabs.tabs.openTabOfKind('form', 'Tides')
    const hold = vi.fn()

    expect(windowTabs.closeTab(id, hold)).toBe(false)

    expect(hold).toHaveBeenCalledOnce()
    expect(onScreen(windowTabs)).toStrictEqual([id])
  })

  it('leaves the removal to the caller where one passes a hold and the kind releases it', async () => {
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([kind(() => true)])
    const id = await windowTabs.tabs.openTabOfKind('form', 'Tides')
    const hold = vi.fn()

    expect(windowTabs.closeTab(id, hold)).toBe(true)

    expect(hold).not.toHaveBeenCalled()
    expect(onScreen(windowTabs)).toStrictEqual([id])
  })
})
