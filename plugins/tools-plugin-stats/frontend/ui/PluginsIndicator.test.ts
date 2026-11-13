import { describe, expect, it } from 'vitest'
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

const draw = (processes: readonly PluginProcess[], phase: Phase = 'ready') =>
  mount(PluginsIndicator, {
    props: { processes, phase, now: NOW, locale: 'en-US' },
    attachTo: document.body,
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

  it('counts down as the time moves on', async () => {
    const wrapper = draw([ATLAS])
    await wrapper.find('.plugin-stats__trigger').trigger('click')
    const row = () => wrapper.find('.plugin-popover__plugin')
    const countdown = () => row().find('.plugin-popover__countdown')
    expect(countdown().text()).toBe('1m 55s')

    await wrapper.setProps({ now: NOW + 5_000 })

    expect(countdown().text()).toBe('1m 50s')
    expect(countdown().attributes('title')).toBe('Stops in 1m 50s; idle for 1m 10s')
    expect(row().attributes('title')).toBe('org.example.atlas\nRunning for 10m 05s')
    wrapper.unmount()
  })

  it('marks an idle plugin and one in use by the dot in front of the name', async () => {
    const { idleStopTime: _stop, ...running } = ATLAS
    const busy = { ...running, pluginId: 'org.example.maps', pluginName: 'Maps', leaseCount: 1 }
    const wrapper = draw([ATLAS, busy])
    await wrapper.find('.plugin-stats__trigger').trigger('click')
    const [atlas, maps] = wrapper.findAll('.plugin-popover__plugin')

    expect(atlas?.find('[role="img"]').attributes('aria-label')).toBe('Idle')
    expect(maps?.find('[role="img"]').attributes('aria-label')).toBe('In use')
    expect(maps?.find('.plugin-popover__countdown').text()).toBe('')
    wrapper.unmount()
  })

  it('writes the worst health item of an unhealthy plugin and nothing for a healthy one', async () => {
    const sick = {
      ...ATLAS,
      pluginId: 'org.example.maps',
      pluginName: 'Maps',
      health: 'error' as const,
      healthText: 'The map server does not answer',
    }
    const wrapper = draw([ATLAS, sick])
    await wrapper.find('.plugin-stats__trigger').trigger('click')
    const [atlas, maps] = wrapper.findAll('.plugin-popover__plugin')

    expect(atlas?.find('.plugin-popover__health').exists()).toBe(false)
    const health = maps?.find('.plugin-popover__health')
    expect(health?.text()).toBe('The map server does not answer')
    expect(health?.attributes('title')).toBe('The map server does not answer')
    wrapper.unmount()
  })

  it('says that no plugin runs in an open list with none', async () => {
    const wrapper = draw([])

    await wrapper.find('.plugin-stats__trigger').trigger('click')

    expect(wrapper.find('.plugin-popover__empty').text()).toBe('No plugin is running')
    wrapper.unmount()
  })
})
