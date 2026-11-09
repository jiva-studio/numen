import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
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

const mounted: VueWrapper[] = []

afterEach(() => {
  for (const wrapper of mounted.splice(0)) wrapper.unmount()
})

const drawView = (component: never): VueWrapper => {
  const wrapper = mount(component, { attachTo: document.body })
  mounted.push(wrapper)
  return wrapper
}

const getTrigger = (): HTMLButtonElement => {
  const trigger = document.body.querySelector<HTMLButtonElement>('button[aria-haspopup="dialog"]')
  if (!trigger) throw new Error('the indicator draws no trigger')
  return trigger
}

const getButtons = (): HTMLButtonElement[] => [...document.body.querySelectorAll('button')]

const press = async (element: HTMLElement): Promise<void> => {
  element.click()
  await flushPromises()
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
  activate(context, { api: { check }, timer })
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

    drawView(context.getRegisteredSlotViews()[0]?.component as never)
    expect(getTrigger().textContent?.trim()).toBe('v2026.10.1')
    await press(getTrigger())
    const download = getButtons().find((one) => one.textContent === 'Download')
    download?.click()
    await flush()

    expect(context.getOpenedAddresses()).toEqual(['https://dl.numen.md/numen-windows-setup.exe'])
    expect(document.body.querySelector('.update-popover')).toBeNull()
  })

  it('closes the popover on Escape and keeps the title and the version apart', async () => {
    const { context } = setup(AVAILABLE)
    await flush()
    drawView(context.getRegisteredSlotViews()[0]?.component as never)
    await press(getTrigger())

    expect(document.body.querySelector('.update-popover__title')?.textContent).toBe(
      'Update available',
    )
    expect(document.body.querySelector('.update-popover__header')?.textContent?.trim()).toBe(
      'Update available',
    )
    expect(document.body.querySelector('.update-popover__release')?.textContent).toContain(
      'v2026.10.1',
    )
    document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flush()

    expect(document.body.querySelector('.update-popover')).toBeNull()
  })

  it('draws nothing for a build that is current', async () => {
    const { context } = setup({ state: 'current' })
    await flush()

    drawView(context.getRegisteredSlotViews()[0]?.component as never)

    expect(document.body.querySelector('button[aria-haspopup]')).toBeNull()
  })

  it('draws nothing before the first check has an answer', () => {
    const { context } = setup(AVAILABLE)

    drawView(context.getRegisteredSlotViews()[0]?.component as never)

    expect(document.body.querySelector('button[aria-haspopup]')).toBeNull()
  })

  it('says so in its tooltip when an update is available', async () => {
    const { context } = setup(AVAILABLE)
    await flush()

    drawView(context.getRegisteredSlotViews()[0]?.component as never)

    expect(getTrigger().getAttribute('tabindex')).toBe('-1')
    expect(getTrigger().getAttribute('title')).toBe(
      'Update available: v2026.10.1. You run v2026.9.0. Click to download.',
    )
  })

  it('draws nothing again once the update is dismissed', async () => {
    const { context } = setup(AVAILABLE)
    await flush()
    drawView(context.getRegisteredSlotViews()[0]?.component as never)
    await press(getTrigger())

    getButtons()
      .find((one) => one.textContent === 'Dismiss')
      ?.click()
    await flush()

    expect(document.body.querySelector('button[aria-haspopup]')).toBeNull()
  })

  it('waits the interval of the setting between checks and stops with the plugin', async () => {
    const { context, waits } = setup(AVAILABLE)
    await flush()

    expect(waits.map((one) => one.delayMs)).toEqual([2 * 60 * 60 * 1000])
    context.dispose()

    expect(waits).toEqual([])
  })
})
