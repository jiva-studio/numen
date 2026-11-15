import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginParts } from '../context'
import { createTestContextHost } from '../../testing/contextHost'
import type { DropRegistration } from '@numen/plugin-sdk'
import { createTestPluginHost, type TestHostDeps } from '../../testing/hostDeps'

const URI_LIST = 'text/uri-list'

function createRegisteringHost() {
  const registered = vi.fn((_id: string, _reg: DropRegistration) => ({ dispose: vi.fn() }))
  const host = createTestContextHost({ views: { registerDropHandler: registered } })
  return { host, registered }
}

const takeNoPaths = async (): Promise<readonly string[]> => []

describe('a drop handler registration', () => {
  it('fails for a type the manifest does not declare, naming the plugin and the type', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:read'],
      dropTypes: ['text/plain'],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)

    expect(() =>
      context.plugin.register.dropHandler?.({ types: [URI_LIST], drop: takeNoPaths }),
    ).toThrowError(
      'Plugin "md.numen.sample" registers the drop type "text/uri-list", which its manifest does not declare in contributes.dropHandlers',
    )
    expect(() =>
      context.plugin.register.dropHandler?.({ types: ['text/plain', URI_LIST], drop: takeNoPaths }),
    ).toThrowError('registers the drop type "text/uri-list"')
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails for a plugin whose manifest declares no drop types', () => {
    const { host, registered } = createRegisteringHost()
    const context = createPluginParts(
      'md.numen.sample',
      host,
      createMockManifest('md.numen.sample', { permissions: ['vault:read'] }),
    )

    expect(() =>
      context.plugin.register.dropHandler?.({ types: [URI_LIST], drop: takeNoPaths }),
    ).toThrowError('does not declare')
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails without vault:read even for a declared type', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:write'],
      dropTypes: [URI_LIST],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)

    expect(() =>
      context.plugin.register.dropHandler?.({ types: [URI_LIST], drop: takeNoPaths }),
    ).toThrowError('lacks required permission "vault:read" for dropHandler')
    expect(registered).not.toHaveBeenCalled()
  })

  it('registers the types the registration had when it was checked, not the types it has later', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:read'],
      dropTypes: [URI_LIST],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)
    const types = [URI_LIST]

    context.plugin.register.dropHandler?.({ types, drop: takeNoPaths })
    types.push('text/html')

    expect(registered.mock.calls[0]?.[1].types).toEqual([URI_LIST])
  })

  it('does not let a getter answer the check and the registry differently', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:read'],
      dropTypes: [URI_LIST],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)
    let reads = 0
    const registration: DropRegistration = {
      get types() {
        reads += 1
        return reads === 1 ? [URI_LIST] : ['text/html']
      },
      drop: takeNoPaths,
    }

    context.plugin.register.dropHandler?.(registration)

    expect(registered.mock.calls[0]?.[1].types).toEqual([URI_LIST])
  })
})

describe('a drop in a host with several plugins', () => {
  const createHost = () =>
    createTestPluginHost({
      target: 'desktop.editor',
      registerKind: () => vi.fn(),
      registerExtensionReader: () => vi.fn(),
      openTab: () => Promise.resolve('tab'),
    } as unknown as TestHostDeps)
  const sampleManifest = (id: string, dropTypes: readonly string[]) =>
    createMockManifest(id, { permissions: ['vault:read'], dropTypes })

  it('reaches only the plugin that declared the type', async () => {
    const host = createHost()
    const intruder = host.lifecycle.createContext(
      'md.numen.intruder',
      sampleManifest('md.numen.intruder', []),
    )
    const taker = host.lifecycle.createContext(
      'md.numen.links',
      sampleManifest('md.numen.links', [URI_LIST]),
    )
    const delivered: string[] = []

    expect(() =>
      intruder.plugin.register.dropHandler?.({
        types: [URI_LIST],
        drop: async () => {
          delivered.push('intruder')
          return []
        },
      }),
    ).toThrowError('does not declare')
    taker.plugin.register.dropHandler?.({
      types: [URI_LIST],
      drop: async () => {
        delivered.push('links')
        return ['clip.md']
      },
    })
    const drops = host.lifecycle.createContext(
      'md.numen.views.file.tree',
      createMockManifest('md.numen.views.file.tree', { permissions: ['vault:write'] }),
    ).host.ui.drops

    expect(await drops?.drop('', { [URI_LIST]: 'https://video.example/clip' })).toEqual(['clip.md'])
    expect(delivered).toEqual(['links'])
  })

  it('refuses a second plugin that declares a type the first already takes, naming both', async () => {
    const host = createHost()
    const first = host.lifecycle.createContext(
      'md.numen.first',
      sampleManifest('md.numen.first', [URI_LIST]),
    )
    const second = host.lifecycle.createContext(
      'md.numen.second',
      sampleManifest('md.numen.second', [URI_LIST]),
    )

    first.plugin.register.dropHandler?.({ types: [URI_LIST], drop: async () => ['first.md'] })

    expect(() =>
      second.plugin.register.dropHandler?.({ types: [URI_LIST], drop: async () => ['second.md'] }),
    ).toThrowError(
      'Plugin "md.numen.second" registers a drop handler for "text/uri-list", which plugin "md.numen.first" already takes',
    )
    const drops = host.lifecycle.createContext(
      'md.numen.views.file.tree',
      createMockManifest('md.numen.views.file.tree', { permissions: ['vault:write'] }),
    ).host.ui.drops
    expect(await drops?.drop('', { [URI_LIST]: 'x' })).toEqual(['first.md'])
  })
})
