/** The window handle a kind receives, activating tabs by id. */
import { describe, expect, it } from 'vitest'
import { getPanes } from '@numen/workbench'
import type { AnyTabKind } from '../lib/kinds'
import { createTestTabs } from '@/testing/tabs'

const thing: AnyTabKind = {
  kind: 'thing',
  open: (targetPath: string) => ({ targetPath, title: targetPath }),
  pane: {},
  getTitle: (state: { title: string }) => state.title,
}

describe('the show of the window handle', () => {
  it('activates an open tab', async () => {
    const window = createTestTabs()
    window.groups.registerKinds([thing])
    const first = await window.tabs.openTabOfKind('thing', 'Heron.md')
    await window.tabs.openTabOfKind('thing', 'Reed.md')

    window.groups.handle.show(first)

    expect(window.groups.handle.front()?.id).toBe(first)
  })

  it('changes nothing for an id of no open tab', async () => {
    const window = createTestTabs()
    window.groups.registerKinds([thing])
    await window.tabs.openTabOfKind('thing', 'Heron.md')
    const before = window.panes.layout.value

    window.groups.handle.show('tab-9')

    expect(window.panes.layout.value).toBe(before)
    expect(getPanes(window.panes.layout.value.root).flatMap((pane) => pane.tabs)).not.toContain(
      'tab-9',
    )
  })
})
