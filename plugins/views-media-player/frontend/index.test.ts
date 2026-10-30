import { describe, expect, it, vi } from 'vitest'
import { AudioLines } from '@lucide/vue'
import { createMockPluginContext } from '@numen/plugin-sdk'
import plugin, { activate, MediaRecordingTab, RecordingTab, RECORDER } from './index'
import { recordings } from './api/wire'

describe('media-player plugin', () => {
  it('registers audio reader and recording tab kind upon activation', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.media.player' })
    const registerReaderSpy = vi.spyOn(context, 'registerReader')
    const registerTabKindSpy = vi.spyOn(context, 'registerTabKind')

    activate(context)

    expect(registerReaderSpy).toHaveBeenCalledWith({
      id: 'media-player',
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

  it('opens a tab to record in from its command', async () => {
    const kinds: (string | undefined)[] = []
    const context = createMockPluginContext({
      pluginId: 'md.numen.views.media.player',
      openTab: (kind) => {
        kinds.push(kind)
        return Promise.resolve('')
      },
    })
    activate(context)

    expect(context.getRegisteredCommands().map((one) => [one.id, one.title])).toStrictEqual([
      ['record', 'Record audio'],
    ])
    await context.triggerCommand('record')

    expect(kinds).toStrictEqual([RECORDER])
  })

  it('declares activate hook on default export', () => {
    expect(plugin).toBeDefined()
    expect(typeof plugin.activate).toBe('function')
  })

  it('watches the work of every plugin on a recording through the task host', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.media.player' })
    const tasks = {
      watch: vi.fn(),
      watchAll: vi.fn((onTasks: (list: readonly { about: string }[]) => void) => {
        onTasks([{ about: 'talks/a.mp3' }])
        return { dispose: () => undefined }
      }),
    }
    activate(Object.assign(context, { tasks }))
    const changes: boolean[] = []

    const stop = recordings.watchWork('talks/a.mp3', (isWorking) => changes.push(isWorking))
    stop()

    expect(tasks.watchAll).toHaveBeenCalledTimes(1)
    expect(tasks.watch).not.toHaveBeenCalled()
    expect(changes).toEqual([true])
  })
})
