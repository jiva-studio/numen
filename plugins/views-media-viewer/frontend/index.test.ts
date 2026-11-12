import { describe, expect, it, vi } from 'vitest'
import { AudioLines } from '@lucide/vue'
import { createMockPluginContext } from '@numen/plugin-sdk'
import plugin, { activate, MEDIA, MediaRecordingTab, RecordingTab, RECORDER } from './index'
import { recordings } from './api/wire'

describe('media-viewer plugin', () => {
  it('registers audio reader and recording tab kind upon activation', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.media.viewer' })
    const registerReaderSpy = vi.spyOn(context.register, 'reader')
    const registerTabKindSpy = vi.spyOn(context.register, 'tabKind')

    activate(context)

    expect(registerReaderSpy).toHaveBeenCalledWith({
      id: 'media-viewer',
      component: RecordingTab,
      title: 'Audio Player',
      icon: AudioLines,
      capabilities: ['recording', 'media', 'audio'],
    })

    expect(registerTabKindSpy).toHaveBeenCalledWith({
      kind: 'recording',
      component: MediaRecordingTab,
      title: 'Record Audio',
    })
  })

  it('registers the tab kind a note with media opens in, titled by the note', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.media.viewer' })

    activate(context)

    const kind = context.getRegisteredTabKinds().find((one) => one.kind === MEDIA)
    expect(MEDIA).toBe('media')
    expect(kind?.component).toBe(RecordingTab)
    const state = kind?.open?.('links/Tidal mills.md', 'tab')
    expect(kind?.getTitle?.(state as never)).toBe('Tidal mills')
    expect(kind?.getTarget?.(state as never)).toStrictEqual({
      path: 'links/Tidal mills.md',
      file: 'links/Tidal mills.md',
      title: 'Tidal mills',
    })
  })

  it('opens a tab to record in from its command', async () => {
    const kinds: (string | undefined)[] = []
    const context = createMockPluginContext({
      pluginId: 'md.numen.views.media.viewer',
      openTab: (kind) => {
        kinds.push(kind)
        return Promise.resolve('')
      },
    })
    activate(context)

    expect(context.getRegisteredCommands().map((one) => one.id)).toStrictEqual(['recording.create'])
    await context.triggerCommand('recording.create')

    expect(kinds).toStrictEqual([RECORDER])
  })

  it('declares activate hook on default export', () => {
    expect(plugin).toBeDefined()
    expect(typeof plugin.activate).toBe('function')
  })

  it('watches the work of every plugin on a recording through the task host', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.media.viewer' })
    const tasks = {
      watch: vi.fn(),
      watchAll: vi.fn((onTasks: (list: readonly { about: string }[]) => void) => {
        onTasks([{ about: 'talks/a.mp3' }])
        return { dispose: () => undefined }
      }),
    }
    Object.assign(context.work, { tasks })
    activate(context)
    const changes: boolean[] = []

    const stop = recordings.watchWork('talks/a.mp3', (isWorking) => changes.push(isWorking))
    stop()

    expect(tasks.watchAll).toHaveBeenCalledTimes(1)
    expect(tasks.watch).not.toHaveBeenCalled()
    expect(changes).toEqual([true])
  })
})
