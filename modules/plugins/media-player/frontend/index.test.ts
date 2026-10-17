import { describe, expect, it, vi } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk'
import plugin, { activate, AudioPlayerTab, MediaRecordingTab } from './index'

describe('media-player plugin', () => {
  it('registers audio reader and recording tab kind upon activation', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.media-player' })
    const registerReaderSpy = vi.spyOn(context, 'registerReader')
    const registerTabKindSpy = vi.spyOn(context, 'registerTabKind')

    activate(context)

    expect(registerReaderSpy).toHaveBeenCalledWith({
      id: 'media-player',
      fileExtensions: ['.mp3', '.m4a', '.wav', '.ogg', '.aac', '.flac'],
      component: AudioPlayerTab,
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

  it('declares activate hook on default export', () => {
    expect(plugin).toBeDefined()
    expect(typeof plugin.activate).toBe('function')
  })
})
