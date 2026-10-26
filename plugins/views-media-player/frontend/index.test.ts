import { describe, expect, it, vi } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk'
import plugin, { activate, MediaRecordingTab, RecordingTab, RECORDER } from './index'

describe('media-player plugin', () => {
  it('registers audio reader and recording tab kind upon activation', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.media.player' })
    const registerReaderSpy = vi.spyOn(context, 'registerReader')
    const registerTabKindSpy = vi.spyOn(context, 'registerTabKind')

    activate(context)

    expect(registerReaderSpy).toHaveBeenCalledWith({
      id: 'media-player',
      fileExtensions: ['.mp3', '.m4a', '.wav', '.ogg', '.aac', '.flac'],
      component: RecordingTab,
      title: 'Audio Player',
      source: 'recording',
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
})
