import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginContext } from '../context'
import { createPluginHost } from '../host'
import type { PluginHostDeps } from '../types'
import { createTestContextHost } from '../../testing/contextHost'
import type { DropRegistration } from '@numen/plugin-sdk'

const URI_LIST = 'text/uri-list'

function createRegistering() {
  const registered = vi.fn((_id: string, _reg: DropRegistration) => ({ dispose: vi.fn() }))
  const host = createTestContextHost({ views: { registerDropHandler: registered } })
  return { host, registered }
}

const takeNothing = async (): Promise<readonly string[]> => []

describe('a drop handler registration', () => {
  it('fails for a type the manifest does not declare, naming the plugin and the type', () => {
    const { host, registered } = createRegistering()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:read'],
      dropTypes: ['text/plain'],
    })
    const context = createPluginContext('md.numen.sample', host, manifest)

    expect(() =>
      context.register.dropHandler?.({ types: [URI_LIST], drop: takeNothing }),
    ).toThrowError(
      'Plugin "md.numen.sample" registers the drop type "text/uri-list", which its manifest does not declare in contributes.dropHandlers',
    )
    expect(() =>
      context.register.dropHandler?.({ types: ['text/plain', URI_LIST], drop: takeNothing }),
    ).toThrowError('registers the drop type "text/uri-list"')
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails for a plugin whose manifest declares no drop types', () => {
    const { host, registered } = createRegistering()
    const context = createPluginContext(
      'md.numen.sample',
      host,
      createMockManifest('md.numen.sample', { permissions: ['vault:read'] }),
    )

    expect(() =>
      context.register.dropHandler?.({ types: [URI_LIST], drop: takeNothing }),
    ).toThrowError('does not declare')
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails without vault:read even for a declared type', () => {
    const { host, registered } = createRegistering()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:write'],
      dropTypes: [URI_LIST],
    })
    const context = createPluginContext('md.numen.sample', host, manifest)

    expect(() =>
      context.register.dropHandler?.({ types: [URI_LIST], drop: takeNothing }),
    ).toThrowError('lacks required permission "vault:read" for dropHandler')
    expect(registered).not.toHaveBeenCalled()
  })

  it('registers the types the registration had when it was checked, not the types it has later', () => {
    const { host, registered } = createRegistering()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:read'],
      dropTypes: [URI_LIST],
    })
    const context = createPluginContext('md.numen.sample', host, manifest)
    const types = [URI_LIST]

    context.register.dropHandler?.({ types, drop: takeNothing })
    types.push('text/html')

    expect(registered.mock.calls[0]?.[1].types).toEqual([URI_LIST])
  })

  it('does not let a getter answer the check and the registry differently', () => {
    const { host, registered } = createRegistering()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:read'],
      dropTypes: [URI_LIST],
    })
    const context = createPluginContext('md.numen.sample', host, manifest)
    let reads = 0
    const registration: DropRegistration = {
      get types() {
        reads += 1
        return reads === 1 ? [URI_LIST] : ['text/html']
      },
      drop: takeNothing,
    }

    context.register.dropHandler?.(registration)

    expect(registered.mock.calls[0]?.[1].types).toEqual([URI_LIST])
  })
})

describe('a drop in a host with several plugins', () => {
  const createHost = () =>
    createPluginHost({
      registerKind: () => vi.fn(),
      registerExtensionReader: () => vi.fn(),
      openTab: () => Promise.resolve('tab'),
    } as unknown as PluginHostDeps)
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
    const seen: string[] = []

    expect(() =>
      intruder.register.dropHandler?.({
        types: [URI_LIST],
        drop: async () => {
          seen.push('intruder')
          return []
        },
      }),
    ).toThrowError('does not declare')
    taker.register.dropHandler?.({
      types: [URI_LIST],
      drop: async () => {
        seen.push('links')
        return ['clip.md']
      },
    })
    const drops = host.lifecycle.createContext(
      'md.numen.views.file.tree',
      createMockManifest('md.numen.views.file.tree', { permissions: ['vault:write'] }),
    ).ui.drops

    expect(await drops?.drop('', { [URI_LIST]: 'https://video.example/clip' })).toEqual(['clip.md'])
    expect(seen).toEqual(['links'])
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

    first.register.dropHandler?.({ types: [URI_LIST], drop: async () => ['first.md'] })

    expect(() =>
      second.register.dropHandler?.({ types: [URI_LIST], drop: async () => ['second.md'] }),
    ).toThrowError(
      'Plugin "md.numen.second" registers a drop handler for "text/uri-list", which plugin "md.numen.first" already takes',
    )
    const drops = host.lifecycle.createContext(
      'md.numen.views.file.tree',
      createMockManifest('md.numen.views.file.tree', { permissions: ['vault:write'] }),
    ).ui.drops
    expect(await drops?.drop('', { [URI_LIST]: 'x' })).toEqual(['first.md'])
  })
})
