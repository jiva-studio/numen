/** The window handle a kind receives, activating tabs by id. */
import { describe, expect, it } from 'vitest'
import { getPanes } from '@numen/shell'
import type { AnyTabKind } from './kinds'
import { useWindowTabs } from './windowTabs'

const thing: AnyTabKind = {
  kind: 'thing',
  open: (targetPath: string) => ({ targetPath, title: targetPath }),
  getTitle: (state: { title: string }) => state.title,
  pane: {},
}

describe('the show of the window handle', () => {
  it('activates an open tab', async () => {
    const window = useWindowTabs()
    window.registerKinds([thing])
    const first = await window.openTabOfKind('thing', 'Heron.md')
    await window.openTabOfKind('thing', 'Reed.md')

    window.handle.show(first)

    expect(window.handle.front()?.id).toBe(first)
  })

  it('changes nothing for an id of no open tab', async () => {
    const window = useWindowTabs()
    window.registerKinds([thing])
    await window.openTabOfKind('thing', 'Heron.md')
    const before = window.layout.value

    window.handle.show('tab-9')

    expect(window.layout.value).toBe(before)
    expect(getPanes(window.layout.value.root).flatMap((pane) => pane.tabs)).not.toContain('tab-9')
  })
})
