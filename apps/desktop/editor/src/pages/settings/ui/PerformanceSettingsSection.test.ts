import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/entities/settings'
import PerformanceSettingsSection from './PerformanceSettingsSection.vue'
import type { Installation, SettingsTabState } from '../types'
import { WORDS as words } from '../words'
import paths from '../paths.json'

const createSection = (file: Record<string, unknown> = {}) => {
  const content: SettingEdit[] = []
  const installation: Installation = {
    getSetting: (path) => at(file, path),
    getModels: () => [],
    write: (response) => {
      content.push(...response)
    },
    file: ref('/numen.json'),
    themes: ref([]),
    applied: ref('preset:numen'),
    mode: ref('system'),
    isPinned: ref(false),
    sizes: ref({ interfaceScale: 1, textScale: 1 }),
    bounds: ref({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    choose: vi.fn(),
    isSyncing: ref(true),
    writeSecret: vi.fn(),
    deleteSecret: vi.fn(),
    retryPlugin: vi.fn(),
    choosePath: vi.fn(async () => ''),
  }
  const state: SettingsTabState = { installation }
  const wrapper = mount(PerformanceSettingsSection, {
    props: { state },
  })
  return { recordedWrites: content, wrapper }
}

describe('PerformanceSettingsSection', () => {
  it('draws the heading and default balanced profile', () => {
    const { wrapper } = createSection()
    expect(wrapper.text()).toContain(words.performance)
    expect(wrapper.text()).toContain(words.balancedDetail)
  })

  it('draws eco profile detail when eco is set', () => {
    const { wrapper } = createSection({ performance: { profile: 'eco' } })
    expect(wrapper.text()).toContain(words.ecoDetail)
  })

  it('draws maximum profile detail when maximum is set', () => {
    const { wrapper } = createSection({ performance: { profile: 'maximum' } })
    expect(wrapper.text()).toContain(words.maximumDetail)
  })

  it('draws a profile it does not offer as balanced', () => {
    const { wrapper } = createSection({ performance: { profile: 'custom' } })
    expect(wrapper.text()).toContain(words.balancedDetail)
    const segmented = wrapper.findComponent({ name: 'SegmentedControl' })
    expect(segmented.props('modelValue')).toBe('balanced')
  })

  it('offers the three profiles and no tuning fields', () => {
    const { wrapper } = createSection()
    const segmented = wrapper.findComponent({ name: 'SegmentedControl' })
    expect(segmented.props('choices')).toStrictEqual([
      { id: 'eco', text: words.eco },
      { id: 'balanced', text: words.balanced },
      { id: 'maximum', text: words.maximum },
    ])
    expect(wrapper.findAllComponents({ name: 'NumberField' })).toHaveLength(0)
  })

  it('writes profile change', async () => {
    const { wrapper, recordedWrites } = createSection()
    const segmented = wrapper.findComponent({ name: 'SegmentedControl' })
    segmented.vm.$emit('update:modelValue', 'maximum')
    expect(recordedWrites).toStrictEqual([{ at: paths.performanceProfile, value: '"maximum"' }])
  })
})
