import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import type { CommandNote } from '@numen/plugin-sdk'
import { activate } from './index'
import type { LinkFacts, Links } from './types'

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
  const facts: LinkFacts = { url: 'https://videos.example/clip', isCopyable }
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
    const { links, copied } = createLinks(true)
    activate(ctx, undefined, links)
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
    activate(ctx, undefined, createLinks(false).links)

    expect(await getHandler(ctx)?.listItems?.(CLIP)).toStrictEqual([])
  })
})
