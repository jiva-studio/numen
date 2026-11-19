import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import MediaRecordingTab from './MediaRecordingTab.vue'
import { createRecorderTab } from '../api/saveRecording'
import { createMockHostServices } from '@numen/plugin-sdk'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  vi.useRealTimers()
})

describe('MediaRecordingTab', () => {
  it('stays idle and explains a missing microphone', async () => {
    const tab = mount(MediaRecordingTab)
    await tab.get('[aria-label="Start recording"]').trigger('click')
    expect(tab.get('[role="alert"]').text()).toMatch(/microphone/i)
    expect(tab.text()).toContain('IDLE')
    expect(tab.find('[aria-label="Stop recording"]').exists()).toBe(false)
    tab.unmount()
  })

  it('saves the final chunk in the selected folder and opens the saved file', async () => {
    vi.setSystemTime(new Date('2026-11-19T14:00:00.000Z'))
    const stopTrack = vi.fn()
    const stream = { getTracks: () => [{ stop: stopTrack }] }
    vi.stubGlobal('navigator', { mediaDevices: { getUserMedia: () => Promise.resolve(stream) } })
    vi.stubGlobal(
      'MediaRecorder',
      class {
        static isTypeSupported(type: string) {
          return type === 'audio/mp4'
        }
        state = 'inactive'
        mimeType = 'audio/mp4'
        ondataavailable?: (event: { data: Blob }) => void
        onstop?: () => void
        start() {
          this.state = 'recording'
        }
        stop() {
          this.state = 'inactive'
          this.ondataavailable?.({ data: new Blob(['final audio']) })
          this.onstop?.()
        }
      },
    )
    vi.stubGlobal(
      'AudioContext',
      class {
        createAnalyser() {
          return { disconnect: vi.fn(), getByteFrequencyData: vi.fn() }
        }
        createMediaStreamSource() {
          return { connect: vi.fn() }
        }
        close() {
          return Promise.resolve()
        }
      },
    )
    vi.stubGlobal('URL', { createObjectURL: () => 'blob:audio', revokeObjectURL: vi.fn() })
    const open = vi.fn()
    const host = createMockHostServices({ openTab: open })
    const vault = { id: 'test-vault', name: 'Test vault' }
    Object.assign(host.vault, { vaults: { getState: () => ({ current: vault, vaults: [vault] }) } })
    const fixture = (kind: string) =>
      readFileSync(
        resolve(import.meta.dirname, '../../testdata/api', 'SaveRecording.' + kind + '.json'),
        'utf8',
      )
    const response = fixture('response')
    const fetch = vi.fn().mockResolvedValue(new Response(response))
    vi.stubGlobal('fetch', fetch)
    const tab = mount(MediaRecordingTab, { props: { state: createRecorderTab('Talks', host) } })
    await tab.get('[aria-label="Start recording"]').trigger('click')
    await flushPromises()
    await tab.get('[aria-label="Stop recording"]').trigger('click')
    await vi.waitFor(() => expect(open).toHaveBeenCalledWith('media', JSON.parse(response).path))
    expect(JSON.parse(fetch.mock.calls[0]?.[1].body)).toEqual(JSON.parse(fixture('request')))
    expect(stopTrack).toHaveBeenCalledOnce()
    tab.unmount()
  })
})
