import { describe, expect, it } from 'vitest'
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
      drops.registerDropHandler('md.numen.other', { types: [URI_LIST, 'text/plain'], drop: async () => [] }),
    ).toThrowError('Plugin "md.numen.other" registers a drop handler for "text/uri-list", which plugin "md.numen.links" already takes')
    expect(drops.canDrop(['text/plain'])).toBe(false)
    first.dispose()
    drops.registerDropHandler('md.numen.other', { types: [URI_LIST], drop: async () => ['other.md'] })
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
})
