import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate } from './index'

describe('the media importer frontend', () => {
  it('registers no tab kind and no reader', () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    activate(ctx)

    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredReaders()).toHaveLength(0)
  })
})

describe('addresses dropped from outside the window', () => {
  it('become link notes in the folder they were dropped into', async () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    const made: string[] = []
    activate(ctx, async (folder, urls) => {
      made.push(`${folder}: ${urls.join(' ')}`)
      return { ok: true, value: urls.map((_, i) => `${folder}/link ${i}.md`) }
    })

    const [drop] = ctx.getRegisteredDropHandlers()
    expect(drop?.types).toEqual(['text/uri-list'])
    const paths = await drop?.drop('talks', {
      'text/uri-list': '# from a browser\nhttps://video.test/clip\nhttps://tidal-mills.test/',
    })

    expect(made).toEqual(['talks: https://video.test/clip https://tidal-mills.test/'])
    expect(paths).toEqual(['talks/link 0.md', 'talks/link 1.md'])
  })
})
