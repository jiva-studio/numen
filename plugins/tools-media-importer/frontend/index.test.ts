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
      return {
        ok: true,
        value: { paths: urls.map((_, i) => `${folder}/link ${i}.md`), failed: [] },
      }
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

describe('addresses a drop could not make notes for', () => {
  const drop = async (failed: readonly string[]) => {
    const errors: string[] = []
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.tools.media.importer',
      workspace: { messages: { showError: (text: string) => errors.push(text) } } as never,
    })
    activate(ctx, async () => ({ ok: true, value: { paths: ['links/a.md'], failed } }))
    const paths = await ctx.getRegisteredDropHandlers()[0]?.drop('links', {
      'text/uri-list': 'https://a.test/',
    })
    return { errors, paths }
  }

  it('are reported once, with the paths of the other notes still returned', async () => {
    const { errors, paths } = await drop(['https://x.test/', 'https://y.test/'])

    expect(errors).toEqual(['Could not create a note for: https://x.test/, https://y.test/'])
    expect(paths).toEqual(['links/a.md'])
  })

  it('report nothing when there are none', async () => {
    expect((await drop([])).errors).toEqual([])
  })
})
