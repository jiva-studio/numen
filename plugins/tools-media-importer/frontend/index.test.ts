import { describe, expect, it, vi } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, LINK, LinkTab } from './index'
import { links as servedLinks } from './api/wire'

describe('the media importer frontend', () => {
  it('opens a note of type link in its tab', () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    activate(ctx)

    expect(ctx.getRegisteredReaders()).toHaveLength(0)
    const kind = ctx.getRegisteredTabKinds().find((one) => one.kind === LINK)
    expect(LINK).toBe('link')
    expect(kind?.component).toBe(LinkTab)
    expect(kind?.getTitle?.(kind.open?.('links/Tidal mills.md', 'tab') as never)).toBe(
      'Tidal mills',
    )
    expect(kind?.getTarget?.({ path: 'links/Tidal mills.md' })).toEqual({
      path: 'links/Tidal mills.md',
      file: 'links/Tidal mills.md',
      title: 'Tidal mills',
    })
  })

  it('watches the work of its own tasks through the task host', () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    const tasks = {
      watch: vi.fn((onTasks: (list: readonly { about: string }[]) => void) => {
        onTasks([{ about: 'links/a.md' }])
        return { dispose: () => undefined }
      }),
      watchAll: vi.fn(),
    }
    activate(Object.assign(ctx, { tasks }))
    const changes: boolean[] = []

    const stop = servedLinks.watchWork('links/a.md', (isWorking) => changes.push(isWorking))
    stop()

    expect(tasks.watch).toHaveBeenCalledTimes(1)
    expect(tasks.watchAll).not.toHaveBeenCalled()
    expect(changes).toEqual([true])
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
