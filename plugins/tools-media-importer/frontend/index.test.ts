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
  })

  it('watches the work of its own tasks through the task host', () => {
    const ctx = createMockPluginContext({ pluginId: 'md.numen.tools.media.importer' })
    const tasks = {
      watch: vi.fn((onTasks: (list: readonly { about: string }[]) => void) => {
        onTasks([{ about: 'links/a.md' }])
        return { dispose: () => {} }
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
