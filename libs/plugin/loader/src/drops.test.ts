import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import type { Manifest } from '@numen/core-client'
import { createDropRegistry } from './drops'

const URI_LIST = 'text/uri-list'

describe('createDropRegistry', () => {
  it('calls the drop handler registered for the dropped type', async () => {
    const drops = createDropRegistry()
    const calls: string[] = []
    drops.registerDropHandler('md.numen.links', {
      types: [URI_LIST, 'text/html'],
      drop: async (folder, data) => {
        calls.push(`${folder} ${data[URI_LIST]}`)
        return ['talks/clip.md']
      },
    })

    expect(drops.canDrop([URI_LIST, 'text/plain'])).toBe(true)
    expect(drops.canDrop(['Files'])).toBe(false)
    expect(await drops.drop('talks', { [URI_LIST]: 'https://video.example/clip' })).toEqual([
      'talks/clip.md',
    ])
    expect(calls).toEqual(['talks https://video.example/clip'])
  })

  it('refuses a type another plugin already takes, and takes it again after that plugin is disposed', async () => {
    const drops = createDropRegistry()
    const first = drops.registerDropHandler('md.numen.links', {
      types: [URI_LIST],
      drop: async () => ['links.md'],
    })

    expect(() =>
      drops.registerDropHandler('md.numen.other', {
        types: [URI_LIST, 'text/plain'],
        drop: async () => [],
      }),
    ).toThrowError(
      'Plugin "md.numen.other" registers a drop handler for "text/uri-list", which plugin "md.numen.links" already takes',
    )
    expect(drops.canDrop(['text/plain'])).toBe(false)
    first.dispose()
    drops.registerDropHandler('md.numen.other', {
      types: [URI_LIST],
      drop: async () => ['other.md'],
    })
    expect(await drops.drop('', { [URI_LIST]: 'x' })).toEqual(['other.md'])
  })

  it('returns no paths after dispose or for an unregistered or empty type', async () => {
    const drops = createDropRegistry()
    const registered = drops.registerDropHandler('md.numen.links', {
      types: [URI_LIST],
      drop: async () => ['talks/clip.md'],
    })
    drops.registerDropHandler('md.numen.plain', {
      types: ['text/plain'],
      drop: async () => ['talks/plain.md'],
    })

    expect(await drops.drop('', { [URI_LIST]: '', 'text/html': '<b>x</b>' })).toEqual([])
    registered.dispose()
    registered.dispose()
    expect(drops.canDrop([URI_LIST])).toBe(false)
    expect(drops.canDrop(['text/plain'])).toBe(true)
    expect(await drops.drop('', { [URI_LIST]: 'https://video.example/clip' })).toEqual([])
  })

  it('allows the same plugin to register another handler for a type it already owns', () => {
    const drops = createDropRegistry()
    drops.registerDropHandler('md.numen.links', {
      types: [URI_LIST],
      drop: async () => ['first.md'],
    })

    expect(() =>
      drops.registerDropHandler('md.numen.links', {
        types: [URI_LIST],
        drop: async () => ['second.md'],
      }),
    ).not.toThrow()
  })

  it('queries and starts declaring plugins only when no handler is registered yet', async () => {
    const bareManifest: Manifest = { ...createMockManifest('bare'), contributes: undefined }
    const multiManifest = createMockManifest('md.numen.links', { dropTypes: ['image/png'] })
    const secondDrop = createMockManifest('md.numen.links', {
      dropTypes: ['text/html', URI_LIST],
    }).contributes?.dropHandlers
    if (multiManifest.contributes && secondDrop) {
      multiManifest.contributes.dropHandlers.push(...secondDrop)
    }
    const manifests = [bareManifest, createMockManifest('md.numen.empty'), multiManifest]
    const findPluginsOf = vi.fn((match: (m: Manifest) => boolean) =>
      manifests.filter(match).map((m) => m.id),
    )
    const startPluginsOf = vi.fn(async (match: (m: Manifest) => boolean) => {
      if (manifests.some(match)) {
        drops.registerDropHandler('md.numen.links', {
          types: [URI_LIST],
          drop: async (folder, data) => [`${folder}/${data[URI_LIST]}`],
        })
      }
    })
    const drops = createDropRegistry({ findPluginsOf, startPluginsOf })

    expect(drops.canDrop(['application/pdf'])).toBe(false)
    expect(drops.canDrop([URI_LIST])).toBe(true)
    expect(findPluginsOf).toHaveBeenCalledTimes(2)

    expect(await drops.drop('inbox', { [URI_LIST]: 'clip.md' })).toEqual(['inbox/clip.md'])
    expect(startPluginsOf).toHaveBeenCalledTimes(1)

    findPluginsOf.mockClear()
    startPluginsOf.mockClear()
    expect(drops.canDrop([URI_LIST])).toBe(true)
    expect(await drops.drop('inbox', { [URI_LIST]: 'next.md' })).toEqual(['inbox/next.md'])
    expect(findPluginsOf).not.toHaveBeenCalled()
    expect(startPluginsOf).not.toHaveBeenCalled()
  })
})
