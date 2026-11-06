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

/** The links of a vault whose one link publishes a video where isCopyable is set, and its copies. */
function createLinks(isCopyable: boolean) {
  const copied: string[] = []
  const facts: LinkFacts = {
    url: 'https://videos.example/clip',
    isCopyable,
    isTextDeleted: false,
    media: '',
    page: '',
  }
  const links: Links = {
    getLink: async () => ({ ok: true, value: facts }),
    readArticle: async () => '',
    getSource: async () => ({ mediaUrl: '', mediaType: '' }),
    fetchText: async () => ({ ok: true, value: null }),
    copyVideo: async (path) => {
      copied.push(path)
      return { ok: true, value: null }
    },
    deleteCopy: async () => ({ ok: true, value: null }),
    watchWork: () => () => undefined,
  }
  return { links, copied }
}

const getHandler = (ctx: ReturnType<typeof createMockPluginContext>) =>
  ctx.getRegisteredCommands().find((one) => one.id === 'app:link.downloadMedia')

describe('the handler of app:link.downloadMedia', () => {
  it('offers the video of a link whose address publishes one, and downloads it', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    const { links, copied } = createLinks(true)
    activate(ctx, undefined, links)
    const handler = getHandler(ctx)

    expect(await handler?.listItems?.(CLIP)).toStrictEqual([{ id: 'video', title: 'Video' }])
    await handler?.handler(CLIP, 'video')
    expect(copied).toStrictEqual(['links/clip.md'])
  })

  it('offers nothing for a link whose address publishes no video', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    activate(ctx, undefined, createLinks(false).links)

    expect(await getHandler(ctx)?.listItems?.(CLIP)).toStrictEqual([])
  })
})
