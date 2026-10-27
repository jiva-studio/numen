import { describe, expect, it, vi } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, createActivate, LINK, LinkTab } from './index'
import { links as servedLinks } from './api/wire'
import type { Links } from './types'

const links = (made: string[]): Links => ({
  getLink: async () => ({ ok: true, value: { url: '', isCopyable: false, copySize: 0 } }),
  readArticle: async () => '',
  getSource: async () => ({ mediaUrl: '', mediaType: '' }),
  createLink: async (folder, url) => {
    made.push(`${folder}|${url}`)
    return { ok: true, value: { path: 'links/made.url' } }
  },
  watchWork: () => () => {},
})

describe('the media importer frontend', () => {
  it('opens a link file in its tab', () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    createActivate(links([]))(ctx)

    const readers = ctx.getRegisteredReaders()
    expect(readers).toHaveLength(1)
    expect(readers[0]?.fileExtensions).toEqual(['.url'])
    expect(readers[0]?.component).toBe(LinkTab)
  })

  it('makes the link of an address as the file of its kind', async () => {
    const made: string[] = []
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    createActivate(links(made))(ctx)

    const kind = ctx.getRegisteredTabKinds().find((one) => one.kind === LINK)
    expect(kind?.component).toBe(LinkTab)
    const answer = await kind?.createFile?.('links', 'http://tidal-mills.numen.test/history/rance')
    expect(answer).toEqual({ ok: true, value: { path: 'links/made.url' } })
    expect(made).toEqual(['links|http://tidal-mills.numen.test/history/rance'])
    expect(kind?.getTitle?.(kind.open?.('links/Tidal mills.url', 'tab') as never)).toBe(
      'Tidal mills',
    )
  })

  it('watches the work of its own tasks through the task host', () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    const tasks = {
      watch: vi.fn((onTasks: (list: readonly { about: string }[]) => void) => {
        onTasks([{ about: 'links/a.url' }])
        return { dispose: () => {} }
      }),
      watchAll: vi.fn(),
    }
    activate(Object.assign(ctx, { tasks }))
    const changes: boolean[] = []

    const stop = servedLinks.watchWork('links/a.url', (isWorking) => changes.push(isWorking))
    stop()

    expect(tasks.watch).toHaveBeenCalledTimes(1)
    expect(tasks.watchAll).not.toHaveBeenCalled()
    expect(changes).toEqual([true])
  })
})
