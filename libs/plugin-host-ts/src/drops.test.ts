import { describe, expect, it } from 'vitest'
import { createDropRegistry } from './drops'

const URI_LIST = 'text/uri-list'

describe('createDropRegistry', () => {
  it('calls the drop handler registered for the dropped type', async () => {
    const drops = createDropRegistry()
    const calls: string[] = []
    drops.registerDropHandler('md.numen.links', {
      types: [URI_LIST],
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

  it('returns no paths after dispose or for an unregistered type', async () => {
    const drops = createDropRegistry()
    const registered = drops.registerDropHandler('md.numen.links', {
      types: [URI_LIST],
      drop: async () => ['talks/clip.md'],
    })

    expect(await drops.drop('', { 'text/plain': 'https://video.example/clip' })).toEqual([])
    registered.dispose()
    expect(drops.canDrop([URI_LIST])).toBe(false)
    expect(await drops.drop('', { [URI_LIST]: 'https://video.example/clip' })).toEqual([])
  })
})
