import { describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '../lib/kinds'
import { openPlacedTabs, startPlacedTabs } from './openPlacedTabs'

const withPlacement = (name: string, open: AnyTabKind['open']): AnyTabKind => ({
  kind: name,
  open,
  pane: {},
  getTitle: (state: { title: string }) => state.title,
  display: { placement: { pane: 'main', order: 0 } },
})

describe('the tabs the kinds place', () => {
  it('open, and a kind that fails is written as a message and left out', async () => {
    const writeMessage = vi.fn()
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([
      withPlacement('welcome', () => ({ title: 'Welcome' })),
      withPlacement('broken', () => {
        throw createLostConnection('no pane')
      }),
    ])

    await openPlacedTabs(windowTabs, writeMessage)

    expect(windowTabs.tabs.tabs.value.map((one) => one.title)).toStrictEqual(['Welcome'])
    expect(writeMessage).toHaveBeenCalledExactlyOnceWith(
      'The broken tab did not open: Numen is not responding; retrying',
      'error',
    )
  })

  it('open without waiting when started in the background', async () => {
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([withPlacement('welcome', () => ({ title: 'Welcome' }))])

    startPlacedTabs(windowTabs, vi.fn())

    await vi.waitFor(() => expect(windowTabs.tabs.tabs.value).toHaveLength(1))
  })
})
