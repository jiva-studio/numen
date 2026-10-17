import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UpdateIndicator from './UpdateIndicator.vue'
import type { ReleaseManifest } from '../model/types'

describe('UpdateIndicator', () => {
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
    ],
  }

  function createMockFetch(manifest: ReleaseManifest | null): typeof fetch {
    return vi.fn().mockImplementation(() => {
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve(manifest),
      } as unknown as Response)
    })
  }

  it('renders nothing when there is no newer release', async () => {
    const fetchFn = createMockFetch(sampleManifest)
    const wrapper = mount(UpdateIndicator, {
      props: {
        options: {
          currentVersion: '2026.10.1',
          currentBuild: 950,
          fetchFn,
          checkIntervalMs: 0,
        },
      },
    })

    await new Promise((done) => setTimeout(done, 10))
    expect(wrapper.find('.update-indicator').exists()).toBe(false)
  })

  it('renders update button when update is available', async () => {
    const fetchFn = createMockFetch(sampleManifest)
    const openUrl = vi.fn()

    const wrapper = mount(UpdateIndicator, {
      props: {
        options: {
          currentVersion: '2026.9.1',
          currentBuild: 900,
          platform: 'darwin',
          fetchFn,
          openUrl,
          checkIntervalMs: 0,
        },
      },
      attachTo: document.body,
    })

    await new Promise((done) => setTimeout(done, 10))

    expect(wrapper.find('.update-indicator').exists()).toBe(true)
    expect(wrapper.text()).toContain('v2026.10.1')
    expect(wrapper.text()).not.toContain('available')
    expect(wrapper.find('.update-popover').exists()).toBe(false)

    // Toggle popover
    await wrapper.find('.update-indicator__trigger').trigger('click')
    expect(wrapper.find('.update-popover').exists()).toBe(true)
    expect(wrapper.text()).toContain('Download the disk image to update Numen.')
    expect(wrapper.text()).toContain('Released on 15 October 2026')

    // Click download
    const downloadBtn = wrapper.find('.update-popover__btn--primary')
    await downloadBtn.trigger('click')
    expect(openUrl).toHaveBeenCalledWith('https://dl.numen.md/numen-macos.dmg')
    expect(wrapper.find('.update-popover').exists()).toBe(false)
  })

  it('dismisses notification when dismiss button is clicked', async () => {
    const fetchFn = createMockFetch(sampleManifest)
    const wrapper = mount(UpdateIndicator, {
      props: {
        options: {
          currentVersion: '2026.9.1',
          currentBuild: 900,
          fetchFn,
          checkIntervalMs: 0,
        },
      },
      attachTo: document.body,
    })

    await new Promise((done) => setTimeout(done, 10))
    await wrapper.find('.update-indicator__trigger').trigger('click')

    const dismissBtn = wrapper.find('.update-popover__btn--quiet')
    await dismissBtn.trigger('click')

    expect(wrapper.find('.update-indicator').exists()).toBe(false)
  })

  it('closes popover on Escape key', async () => {
    const fetchFn = createMockFetch(sampleManifest)
    const wrapper = mount(UpdateIndicator, {
      props: {
        options: {
          currentVersion: '2026.9.1',
          currentBuild: 900,
          fetchFn,
          checkIntervalMs: 0,
        },
      },
      attachTo: document.body,
    })

    await new Promise((done) => setTimeout(done, 10))
    await wrapper.find('.update-indicator__trigger').trigger('click')
    expect(wrapper.find('.update-popover').exists()).toBe(true)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.update-popover').exists()).toBe(false)
  })

  it('renders download and release notes buttons on Linux', async () => {
    const fetchFn = createMockFetch(sampleManifest)
    const openUrl = vi.fn()

    const wrapper = mount(UpdateIndicator, {
      props: {
        options: {
          currentVersion: '2026.9.1',
          currentBuild: 900,
          platform: 'linux',
          fetchFn,
          openUrl,
          checkIntervalMs: 0,
        },
      },
      attachTo: document.body,
    })

    await new Promise((done) => setTimeout(done, 10))
    await wrapper.find('.update-indicator__trigger').trigger('click')

    expect(wrapper.find('.update-popover').exists()).toBe(true)
    expect(wrapper.text()).toContain('Visit the download page or update via your package manager.')
    expect(wrapper.text()).toContain('Download')
    expect(wrapper.text()).toContain('Release notes')
    expect(wrapper.text()).toContain('Dismiss')

    const downloadBtn = wrapper.find('.update-popover__btn--primary')
    await downloadBtn.trigger('click')
    expect(openUrl).toHaveBeenCalledWith('https://numen.md/#get')

    await wrapper.find('.update-indicator__trigger').trigger('click')
    const notesBtn = wrapper.find('.update-popover__btn--ghost')
    await notesBtn.trigger('click')
    expect(openUrl).toHaveBeenCalledWith(
      'https://github.com/jiva-studio/numen/releases/tag/v2026.10.1',
    )
  })

  it('does not open duplicate URL if download click event is defaultPrevented', async () => {
    const fetchFn = createMockFetch(sampleManifest)
    const openUrl = vi.fn()

    const wrapper = mount(UpdateIndicator, {
      props: {
        options: {
          currentVersion: '2026.9.1',
          currentBuild: 900,
          platform: 'darwin',
          fetchFn,
          openUrl,
          checkIntervalMs: 0,
        },
      },
      attachTo: document.body,
    })

    await new Promise((done) => setTimeout(done, 10))
    await wrapper.find('.update-indicator__trigger').trigger('click')

    const downloadBtn = wrapper.find('.update-popover__btn--primary')
    const event = new MouseEvent('click', { bubbles: true, cancelable: true })
    event.preventDefault()
    downloadBtn.element.dispatchEvent(event)
    await wrapper.vm.$nextTick()

    expect(openUrl).not.toHaveBeenCalled()
    expect(wrapper.find('.update-popover').exists()).toBe(false)
  })

  it('does not open duplicate URL if release notes click event is defaultPrevented', async () => {
    const fetchFn = createMockFetch(sampleManifest)
    const openUrl = vi.fn()

    const wrapper = mount(UpdateIndicator, {
      props: {
        options: {
          currentVersion: '2026.9.1',
          currentBuild: 900,
          platform: 'linux',
          fetchFn,
          openUrl,
          checkIntervalMs: 0,
        },
      },
      attachTo: document.body,
    })

    await new Promise((done) => setTimeout(done, 10))
    await wrapper.find('.update-indicator__trigger').trigger('click')

    const notesBtn = wrapper.find('.update-popover__btn--ghost')
    const event = new MouseEvent('click', { bubbles: true, cancelable: true })
    event.preventDefault()
    notesBtn.element.dispatchEvent(event)
    await wrapper.vm.$nextTick()

    expect(openUrl).not.toHaveBeenCalled()
    expect(wrapper.find('.update-popover').exists()).toBe(false)
  })
})
