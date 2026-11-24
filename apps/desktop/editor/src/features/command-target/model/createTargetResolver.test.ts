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
  windowTabs.kinds.registerKinds([note])
  const vault = { id: 'vault-1', name: 'Harbor' }
  const resolve = createTargetResolver(
    windowTabs,
    { activeVault: ref(vault) } as unknown as Parameters<typeof createTargetResolver>[1],
    { failure: ref(failure) },
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

  it('resolves empty path, file, and title when the front tab is missing or has no target reader', async () => {
    const { windowTabs, resolve } = create()
    windowTabs.kinds.registerKinds([
      {
        kind: 'plain',
        open: () => ({ title: 'Plain' }),
        pane: {},
        getTitle: (state: { title: string }) => state.title,
      } as AnyTabKind,
      {
        kind: 'address-only',
        open: () => ({ title: 'AddressOnly' }),
        pane: {},
        getTitle: (state: { title: string }) => state.title,
        address: {},
      } as AnyTabKind,
    ])

    const plainId = await windowTabs.tabs.openTabOfKind('plain', '')
    expect(resolve()).toMatchObject({ tab: plainId, kind: 'plain', path: '', title: '', file: '' })

    const addrId = await windowTabs.tabs.openTabOfKind('address-only', '')
    expect(resolve()).toMatchObject({
      tab: addrId,
      kind: 'address-only',
      path: '',
      title: '',
      file: '',
    })

    const orphanResolver = createTargetResolver(
      {
        handle: {
          ...windowTabs.handle,
          getFrontTab: () => ({ id: 'ghost', kind: 'note', state: { path: 'a.md', title: 'A' } }),
        },
        tabs: { getTab: () => null } as never,
      },
      { activeVault: ref({ id: 'vault-1', name: 'Harbor' }) } as never,
      { failure: ref('') },
    )
    expect(orphanResolver()).toMatchObject({
      tab: 'ghost',
      kind: 'note',
      path: '',
      title: '',
      file: '',
    })
  })
})
