import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { useAppUpdate } from './useAppUpdate'
import type { ReleaseManifest } from './types'

describe('useAppUpdate', () => {
  const sampleManifest: ReleaseManifest = {
    version: '2026.10.1',
    build: 950,
    commit: 'abc1234',
    channel: 'stable',
    notes: 'https://dl.numen.md/releases/2026.10.1/notes.md',
    released: '2026-10-15T12:00:00Z',
    files: [
      {
        name: 'numen-macos.dmg',
        platform: 'darwin',
        arch: 'universal',
        kind: 'dmg',
        url: 'https://dl.numen.md/numen-macos.dmg',
        size: 75000000,
        sha256: 'hash-mac',
      },
      {
        name: 'numen-windows-setup.exe',
        platform: 'windows',
        arch: 'amd64',
        kind: 'exe',
        url: 'https://dl.numen.md/numen-windows-setup.exe',
        size: 65000000,
        sha256: 'hash-win',
      },
      {
        name: 'numen-linux-amd64.deb',
        platform: 'linux',
        arch: 'amd64',
        kind: 'deb',
        url: 'https://dl.numen.md/numen-linux-amd64.deb',
        size: 55000000,
        sha256: 'hash-linux',
      },
    ],
  }

  function createMockFetch(manifest: ReleaseManifest | null, status = 200): typeof fetch {
    return vi.fn().mockImplementation(() => {
      if (status !== 200 || !manifest) {
        return Promise.resolve({
          ok: false,
          status,
          json: () => Promise.reject(new Error('Failed')),
        } as unknown as Response)
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve(manifest),
      } as unknown as Response)
    })
  }

  it('detects available update when remote build is newer', async () => {
    const fetchFn = createMockFetch(sampleManifest)
    const openUrl = vi.fn()

    let updateState!: ReturnType<typeof useAppUpdate>
    const TestComponent = defineComponent({
      setup() {
        updateState = useAppUpdate({
          currentVersion: '2026.9.1',
          currentBuild: 900,
          platform: 'darwin',
          fetchFn,
          openUrl,
          checkIntervalMs: 0,
        })
        return () => h('div')
      },
    })

    mount(TestComponent)
    await new Promise((done) => setTimeout(done, 10))

    expect(updateState.hasUpdate.value).toBe(true)
    expect(updateState.isDismissed.value).toBe(false)
    expect(updateState.updateInfo.value).toEqual({
      currentVersion: '2026.9.1',
      currentBuild: 900,
      latestVersion: '2026.10.1',
      latestBuild: 950,
      releasedAt: '2026-10-15T12:00:00Z',
      notesUrl: 'https://github.com/jiva-studio/numen/releases/tag/v2026.10.1',
      platformFile: sampleManifest.files![0],
      downloadUrl: 'https://dl.numen.md/numen-macos.dmg',
    })

    updateState.openDownload()
    expect(openUrl).toHaveBeenCalledWith('https://dl.numen.md/numen-macos.dmg')

    updateState.openReleaseNotes()
    expect(openUrl).toHaveBeenCalledWith(
      'https://github.com/jiva-studio/numen/releases/tag/v2026.10.1',
    )

    updateState.dismissUpdate()
    expect(updateState.isDismissed.value).toBe(true)
  })

  it('selects windows installer when platform is windows', async () => {
    const fetchFn = createMockFetch(sampleManifest)
    const openUrl = vi.fn()

    let updateState!: ReturnType<typeof useAppUpdate>
    const TestComponent = defineComponent({
      setup() {
        updateState = useAppUpdate({
          currentVersion: '2026.9.1',
          currentBuild: 900,
          platform: 'windows',
          fetchFn,
          openUrl,
          checkIntervalMs: 0,
        })
        return () => h('div')
      },
    })

    mount(TestComponent)
    await new Promise((done) => setTimeout(done, 10))

    expect(updateState.hasUpdate.value).toBe(true)
    expect(updateState.updateInfo.value?.downloadUrl).toBe(
      'https://dl.numen.md/numen-windows-setup.exe',
    )
  })

  it('reports no update when local build is equal or newer', async () => {
    const fetchFn = createMockFetch(sampleManifest)

    let updateState!: ReturnType<typeof useAppUpdate>
    const TestComponent = defineComponent({
      setup() {
        updateState = useAppUpdate({
          currentVersion: '2026.10.1',
          currentBuild: 950,
          fetchFn,
          checkIntervalMs: 0,
        })
        return () => h('div')
      },
    })

    mount(TestComponent)
    await new Promise((done) => setTimeout(done, 10))

    expect(updateState.hasUpdate.value).toBe(false)
    expect(updateState.updateInfo.value).toBeNull()
  })

  it('handles fetch errors quietly', async () => {
    const fetchFn = vi.fn().mockRejectedValue(new Error('Network error'))

    let updateState!: ReturnType<typeof useAppUpdate>
    const TestComponent = defineComponent({
      setup() {
        updateState = useAppUpdate({
          currentVersion: '2026.9.1',
          currentBuild: 900,
          fetchFn,
          checkIntervalMs: 0,
        })
        return () => h('div')
      },
    })

    mount(TestComponent)
    await new Promise((done) => setTimeout(done, 10))

    expect(updateState.hasUpdate.value).toBe(false)
    expect(updateState.isChecking.value).toBe(false)
  })

  it('detects available update for unversioned dev build', async () => {
    const fetchFn = createMockFetch(sampleManifest)

    let updateState!: ReturnType<typeof useAppUpdate>
    const TestComponent = defineComponent({
      setup() {
        updateState = useAppUpdate({
          currentVersion: '0.0.0-dev',
          currentBuild: 0,
          platform: 'darwin',
          fetchFn,
          checkIntervalMs: 0,
        })
        return () => h('div')
      },
    })

    mount(TestComponent)
    await new Promise((done) => setTimeout(done, 10))

    expect(updateState.hasUpdate.value).toBe(true)
    expect(updateState.updateInfo.value?.latestVersion).toBe('2026.10.1')
  })

  it('reports no update when current version matches remote version', async () => {
    const fetchFn = createMockFetch(sampleManifest)

    let updateState!: ReturnType<typeof useAppUpdate>
    const TestComponent = defineComponent({
      setup() {
        updateState = useAppUpdate({
          currentVersion: '2026.10.1',
          currentBuild: 0,
          fetchFn,
          checkIntervalMs: 0,
        })
        return () => h('div')
      },
    })

    mount(TestComponent)
    await new Promise((done) => setTimeout(done, 10))

    expect(updateState.hasUpdate.value).toBe(false)
  })

  it('clears interval timer on unmount', () => {
    vi.useFakeTimers()
    const clearIntervalSpy = vi.spyOn(globalThis, 'clearInterval')

    const TestComponent = defineComponent({
      setup() {
        useAppUpdate({
          checkIntervalMs: 5000,
          fetchFn: vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) }),
        })
        return () => h('div')
      },
    })

    const wrapper = mount(TestComponent)
    wrapper.unmount()

    expect(clearIntervalSpy).toHaveBeenCalled()
    vi.useRealTimers()
  })
})
