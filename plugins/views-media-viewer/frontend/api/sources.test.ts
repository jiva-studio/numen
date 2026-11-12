import { describe, expect, it } from 'vitest'
import type { AddressSource, JsonValue } from '@numen/plugin-sdk'
import { createSources } from './sources'

const resolve = async (address: string): Promise<AddressSource> => ({
  url: `http://127.0.0.1:1/token/${address}`,
  mediaType: address.endsWith('.mp4')
    ? 'video/mp4'
    : address.endsWith('.html')
      ? 'text/html'
      : 'audio/mpeg',
})

const notesOf = (properties: Record<string, Record<string, JsonValue>>) => ({
  readProperties: async (path: string) => properties[path] ?? {},
})

describe('where a tab plays from', () => {
  it('plays a file of the vault from its own address', async () => {
    const { getSource } = createSources(() => notesOf({}), resolve)

    expect(await getSource('talks/Ants.mp3')).toStrictEqual({
      mediaUrl: 'http://127.0.0.1:1/token/talks/Ants.mp3',
      mediaType: 'audio/mpeg',
      pageUrl: '',
    })
  })

  it('plays the media a note names', async () => {
    const notes = notesOf({
      'links/tide.md': {
        media: 'plugin://org.example.links/tide.mp4',
        page: 'plugin://org.example.links/tide.html',
      },
    })
    const { getSource } = createSources(() => notes, resolve)

    expect(await getSource('links/tide.md')).toStrictEqual({
      mediaUrl: 'http://127.0.0.1:1/token/plugin://org.example.links/tide.mp4',
      mediaType: 'video/mp4',
      pageUrl: '',
    })
  })

  it('frames the page a note names while it names no media', async () => {
    const notes = notesOf({ 'links/tide.md': { page: 'plugin://org.example.links/tide.html' } })
    const { getSource } = createSources(() => notes, resolve)

    expect(await getSource('links/tide.md')).toStrictEqual({
      mediaUrl: '',
      mediaType: '',
      pageUrl: 'http://127.0.0.1:1/token/plugin://org.example.links/tide.html',
    })
  })

  it('plays nothing for a note that names neither, or a host without notes', async () => {
    const notes = notesOf({ 'links/tide.md': { media: '', url: 'https://tides.example/mill' } })

    expect(await createSources(() => notes, resolve).getSource('links/tide.md')).toStrictEqual({
      mediaUrl: '',
      mediaType: '',
      pageUrl: '',
    })
    expect(await createSources(() => undefined, resolve).getSource('links/tide.md')).toStrictEqual({
      mediaUrl: '',
      mediaType: '',
      pageUrl: '',
    })
  })
})
