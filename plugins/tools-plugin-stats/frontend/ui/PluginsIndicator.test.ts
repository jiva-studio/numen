import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import type { PluginProcess } from '../types'
import type { Phase } from '../model/plugins'
import PluginsIndicator from './PluginsIndicator.vue'

const NOW = 1_700_000_600_000

const ATLAS: PluginProcess = {
  pluginId: 'org.example.atlas',
  pluginName: 'Atlas',
  startTime: NOW - 600_000,
  lastUseTime: NOW - 65_000,
  leaseCount: 0,
  isInProcess: false,
  idleStopTime: NOW + 115_000,
  health: 'ok',
  healthText: '',
}

const draw = (processes: readonly PluginProcess[], phase: Phase = 'ready', clock = () => NOW) =>
  mount(PluginsIndicator, {
    props: { processes, phase, locale: 'en-US', clock },
    attachTo: document.body,
  })

afterEach(() => {
  vi.useRealTimers()
})

describe('the running plugins indicator', () => {
  it('counts the running plugins and names them in its label', () => {
    const trigger = draw([ATLAS]).find('.plugin-stats__trigger')

    expect(trigger.find('.plugin-stats__figure').text()).toBe('1')
    expect(trigger.attributes('aria-label')).toBe('1 plugin running')
    expect(trigger.attributes('tabindex')).toBe('-1')
  })

  it('shows a dash before the first list and where there is none', () => {
    expect(draw([], 'loading').find('.plugin-stats__figure').text()).toBe('–')
    expect(draw([], 'unavailable').find('.plugin-stats__trigger').attributes('aria-label')).toBe(
      'The running plugins are not available',
    )
  })

  it('opens the list on a click and closes it on Escape', async () => {
    const wrapper = draw([ATLAS])

    await wrapper.find('.plugin-stats__trigger').trigger('click')
    expect(wrapper.find('.plugin-popover__name').text()).toBe('Atlas')

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.plugin-popover').exists()).toBe(false)
    wrapper.unmount()
  })

  it('counts down while the list is open', async () => {
    vi.useFakeTimers()
    let now = NOW
    const wrapper = draw([ATLAS], 'ready', () => now)
    await wrapper.find('.plugin-stats__trigger').trigger('click')
    const stop = () => wrapper.findAll('.plugin-popover__value').at(-1)?.text()
    expect(stop()).toBe('1m 55s')

    now += 5_000
    vi.advanceTimersByTime(1_000)
    await wrapper.vm.$nextTick()

    expect(stop()).toBe('1m 50s')
    wrapper.unmount()
  })

  it('says that no plugin runs in an open list with none', async () => {
    const wrapper = draw([])

    await wrapper.find('.plugin-stats__trigger').trigger('click')

    expect(wrapper.find('.plugin-popover__empty').text()).toBe('No plugin is running')
    wrapper.unmount()
  })
})
