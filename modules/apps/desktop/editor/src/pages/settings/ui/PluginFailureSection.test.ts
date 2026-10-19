import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { PluginInfo, SettingEdit } from '@/entities/settings'
import PluginFailureSection from './PluginFailureSection.vue'
import { createInstallation, createPluginInfo } from '@/testing/installation'

const STOPPED = createPluginInfo('sample.stopped', 'Sample Stopped', {
  failure: { reason: 'The plugin did not finish starting in time.', failedAt: 1_700_000_000_000 },
})

const createSection = (plugins: readonly PluginInfo[]) => {
  const written: SettingEdit[] = []
  const retryPlugin = vi.fn()
  const installation = createInstallation({
    write: (said) => {
      written.push(...said)
    },
    plugins: ref(plugins),
    retryPlugin,
  })
  const wrapper = mount(PluginFailureSection, { props: { state: { installation } } })
  return { wrapper, written, retryPlugin }
}

describe('PluginFailureSection', () => {
  it('draws nothing where every plugin starts', () => {
    const { wrapper } = createSection([createPluginInfo('sample.ok', 'Sample Ok')])
    expect(wrapper.find('section').exists()).toBe(false)
  })

  it('draws a plugin held off with its reason and each answer named for it', () => {
    const { wrapper } = createSection([createPluginInfo('sample.ok', 'Sample Ok'), STOPPED])

    const group = wrapper.get('[role="group"]')
    expect(group.attributes('aria-label')).toBe('Sample Stopped')
    expect(group.text()).toContain('The plugin did not finish starting in time.')
    expect(group.text()).toContain('Stopped')
    expect(wrapper.findAll('[role="group"]')).toHaveLength(1)
    expect(wrapper.get('button[aria-label="Retry Sample Stopped"]').text()).toBe('Retry')
    expect(wrapper.get('button[aria-label="Disable Sample Stopped"]').text()).toBe('Disable')
  })

  it('asks the vault to start the plugin again', async () => {
    const { wrapper, retryPlugin } = createSection([STOPPED])

    await wrapper.get('button[aria-label="Retry Sample Stopped"]').trigger('click')

    expect(retryPlugin).toHaveBeenCalledExactlyOnceWith('sample.stopped')
  })

  it('writes the plugin disabled', async () => {
    const { wrapper, written } = createSection([STOPPED])

    await wrapper.get('button[aria-label="Disable Sample Stopped"]').trigger('click')

    expect(written).toStrictEqual([
      { at: ['plugins', 'sample.stopped', 'enabled'], value: 'false' },
    ])
  })
})
