import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '@numen/plugin-loader'
import { createTargetResolver } from './createTargetResolver'

const note: AnyTabKind = {
  kind: 'note',
  open: (path: string) => ({ title: 'Tide table', path }),
  pane: {},
  getTitle: (state: { title: string }) => state.title,
  address: {
    getTarget: (state: { path: string; title: string }) => ({
      file: state.path,
      path: state.path.replace(/\.md$/, ''),
      title: state.title,
    }),
  },
}

const create = (failure = '') => {
  const windowTabs = createTestTabs()
  windowTabs.groups.registerKinds([note])
  const vault = { id: 'vault-1', name: 'Harbor' }
  const resolve = createTargetResolver(
    windowTabs,
    { activeVault: ref(vault) } as unknown as Parameters<typeof createTargetResolver>[1],
    { health: { failure: ref(failure) } } as unknown as Parameters<typeof createTargetResolver>[2],
  )
  return { windowTabs, resolve, vault }
}

describe('the target of a command that names none', () => {
  it('is nothing where no tab is in front', () => {
    const { resolve, vault } = create()

    expect(resolve()).toEqual({
      tab: '',
      kind: null,
      path: '',
      title: '',
      file: '',
      vault,
      isReady: true,
    })
  })

  it('takes its file, path and title from the kind of the tab in front', async () => {
    const { windowTabs, resolve } = create()
    const id = await windowTabs.tabs.openTabOfKind('note', 'Tides.md')

    expect(resolve()).toMatchObject({
      tab: id,
      file: 'Tides.md',
      path: 'Tides',
      title: 'Tide table',
    })
    expect(resolve().kind).not.toBeNull()
  })

  it('is not ready while the window has failed', () => {
    const { resolve } = create('the vault did not load')

    expect(resolve().isReady).toBe(false)
  })
})
