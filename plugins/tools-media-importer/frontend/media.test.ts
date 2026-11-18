import { describe, expect, it } from 'vitest'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import type { CommandNote } from '@numen/plugin-sdk'
import { activate } from './index'
import type { Links } from './types'

const CLIP: CommandNote = {
  path: 'links/clip.md',
  title: 'clip',
  isFolder: false,
  others: [],
  tab: '',
  type: 'link',
}

function createLinks(isCopyable: boolean) {
  const copied: string[] = []
  const facts = { url: 'https://videos.example/clip', isCopyable }
  const links: Links = {
    getLink: async () => ({ ok: true, value: facts }),
    copyMedia: async (path, kind) => {
      copied.push(`${path} ${kind}`)
      return { ok: true, value: null }
    },
  }
  return { links, copied }
}

const getHandler = (ctx: ReturnType<typeof createMockPluginContext>) =>
  ctx.getRegisteredCommands().find((one) => one.id === 'link.downloadMedia')

describe('the handler of link.downloadMedia', () => {
  it('offers the video and the sound of a link whose address publishes a video, and downloads the one picked', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    const host = createMockHostServices()
    const { links, copied } = createLinks(true)
    activate(ctx, host, undefined, links)
    const handler = getHandler(ctx)

    expect(await handler?.listItems?.(CLIP)).toStrictEqual([
      { id: 'video', title: 'Video' },
      { id: 'audio', title: 'Audio' },
    ])
    await handler?.handler(CLIP, 'audio')
    await handler?.handler(CLIP, 'video')
    expect(copied).toStrictEqual(['links/clip.md audio', 'links/clip.md video'])
  })

  it('offers nothing for a link whose address publishes no video', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    const host = createMockHostServices()
    activate(ctx, host, undefined, createLinks(false).links)

    expect(await getHandler(ctx)?.listItems?.(CLIP)).toStrictEqual([])
  })

  it('offers nothing and does not call the backend over a note of another type', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    const host = createMockHostServices()
    let calls = 0
    const links: Links = {
      getLink: async () => {
        calls += 1
        return { ok: true, value: { url: 'https://videos.example/clip', isCopyable: true } }
      },
      copyMedia: async () => ({ ok: true, value: null }),
    }
    activate(ctx, host, undefined, links)

    expect(await getHandler(ctx)?.listItems?.({ ...CLIP, noteType: 'note' })).toStrictEqual([])
    expect(calls).toBe(0)
  })
})
