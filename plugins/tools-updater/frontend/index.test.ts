import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import type { Status } from './api/updater'
import { activate, UPDATE_VIEW } from './index'
import type { Timer } from './model/updates'

const AVAILABLE: Status = {
  state: 'available',
  update: {
    version: '2026.10.1',
    build: 950,
    channel: 'stable',
    releasedAt: '2026-10-15T12:00:00Z',
    notesUrl: 'https://github.com/jiva-studio/numen/releases/tag/v2026.10.1',
    downloadUrl: 'https://dl.numen.md/numen-windows-setup.exe',
  },
}

const flush = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve()
}

const setup = (answer: Status) => {
  const waits: Array<{ run: () => void; delayMs: number; id: number }> = []
  const timer: Timer = {
    set: (run, delayMs) => {
      waits.push({ run, delayMs, id: waits.length })
      return waits.length - 1
    },
    clear: (handle) => {
      const index = waits.findIndex((one) => one.id === handle)
      if (index !== -1) waits.splice(index, 1)
    },
  }
  const check = vi.fn(async () => ({ ok: true as const, value: answer }))
  const context = createMockPluginContext({
    pluginId: 'md.numen.tools.updater',
    app: { name: 'numen', version: '2026.9.0', build: 900, platform: 'windows' },
    initialSettings: { checkIntervalHours: 2 },
  })
  const host = createMockHostServices()
  activate(context, host, { api: { check }, timer })
  return { context, check, waits }
}

describe('the updater frontend', () => {
  it('registers the status view the manifest declares', () => {
    const { context } = setup(AVAILABLE)

    expect(UPDATE_VIEW).toBe('update')
    expect(context.getRegisteredSlotViews().map((one) => one.id)).toEqual(['update'])
  })

  it('checks the running build as the context describes it', async () => {
    const { check } = setup(AVAILABLE)
    await flush()

    expect(check).toHaveBeenCalledWith({
      name: 'numen',
      version: '2026.9.0',
      build: 900,
      platform: 'windows',
    })
  })

  it('draws the indicator for the update found, and the popover opens the download', async () => {
    const { context } = setup(AVAILABLE)
    await flush()
    const component = context.getRegisteredSlotViews()[0]?.component as never

    const wrapper = mount(component)
    expect(wrapper.find('.update-indicator__label').text()).toBe('v2026.10.1')
    await wrapper.find('.update-indicator__trigger').trigger('click')
    const download = wrapper.findAll('button').find((one) => one.text() === 'Download')
    await download?.trigger('click')
    await flush()

    expect(context.getOpenedAddresses()).toEqual(['https://dl.numen.md/numen-windows-setup.exe'])
    expect(wrapper.find('.update-popover').exists()).toBe(false)
  })

  it('closes the popover on Escape and keeps the title and the version apart', async () => {
    const { context } = setup(AVAILABLE)
    await flush()
    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await wrapper.find('.update-indicator__trigger').trigger('click')

    expect(wrapper.find('.update-popover__title').text()).toBe('Update available')
    expect(wrapper.find('.update-popover__header').text()).toBe('Update available')
    expect(wrapper.find('.update-popover__release').text()).toContain('v2026.10.1')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flush()

    expect(wrapper.find('.update-popover').exists()).toBe(false)
    wrapper.unmount()
  })

  it('draws nothing for a build that is current', async () => {
    const { context } = setup({ state: 'current' })
    await flush()

    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)

    expect(wrapper.find('.update-indicator').exists()).toBe(false)
  })

  it('draws nothing before the first check has an answer', () => {
    const { context } = setup(AVAILABLE)

    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)

    expect(wrapper.find('.update-indicator').exists()).toBe(false)
  })

  it('says so in its tooltip when an update is available', async () => {
    const { context } = setup(AVAILABLE)
    await flush()

    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)

    expect(wrapper.find('.update-indicator__trigger').attributes('tabindex')).toBe('-1')
    expect(wrapper.find('.update-indicator__trigger').attributes('title')).toBe(
      'Update available: v2026.10.1. You run v2026.9.0. Click to download.',
    )
  })

  it('draws nothing again once the update is dismissed', async () => {
    const { context } = setup(AVAILABLE)
    await flush()
    const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)
    await wrapper.find('.update-indicator__trigger').trigger('click')

    await wrapper
      .findAll('button')
      .find((one) => one.text() === 'Dismiss')
      ?.trigger('click')

    expect(wrapper.find('.update-indicator').exists()).toBe(false)
  })

  it('waits the interval of the setting between checks and stops with the plugin', async () => {
    const { context, waits } = setup(AVAILABLE)
    await flush()

    expect(waits.map((one) => one.delayMs)).toEqual([2 * 60 * 60 * 1000])
    context.dispose()

    expect(waits).toEqual([])
  })
})
