import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/entities/settings'
import PerformanceSettingsSection from './PerformanceSettingsSection.vue'
import type { Installation, SettingsTabState } from '../types'
import { WORDS as words } from '../words'
import paths from '../paths.json'
import { pick } from '@/testing/pick'

const createSection = (file: Record<string, unknown> = {}) => {
  const content: SettingEdit[] = []
  const installation: Installation = {
    getSetting: (path) => at(file, path),
    getModels: () => [],
    write: (response) => {
      content.push(...response)
    },
    file: ref('/numen.json'),
    openFile: vi.fn(),
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
    isNesting: ref(true),
    headings: ref(6),
    headingsBounds: ref({ least: 2, most: 9 }),
    chooseHeadings: vi.fn(),
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
    expect(wrapper.find('.settings__number').exists()).toBe(false)
  })

  it('draws eco profile detail when eco is set', () => {
    const { wrapper } = createSection({ performance: { profile: 'eco' } })
    expect(wrapper.text()).toContain(words.ecoDetail)
  })

  it('draws maximum profile detail when maximum is set', () => {
    const { wrapper } = createSection({ performance: { profile: 'maximum' } })
    expect(wrapper.text()).toContain(words.maximumDetail)
  })

  it('draws custom profile detail and custom fields when custom is set', () => {
    const { wrapper } = createSection({
      performance: {
        profile: 'custom',
        embedding_batch_size: 64,
        ocr_threads: 8,
        ocr_sessions: 2,
        llm_concurrency: 4,
      },
    })
    expect(wrapper.text()).toContain(words.customDetail)
    expect(wrapper.findAll('.settings__number')).toHaveLength(4)
  })

  it('writes profile change', async () => {
    const { wrapper, recordedWrites } = createSection()
    const segmented = wrapper.findComponent({ name: 'SegmentedControl' })
    segmented.vm.$emit('update:modelValue', 'maximum')
    expect(recordedWrites).toStrictEqual([{ at: paths.performanceProfile, value: '"maximum"' }])
  })

  it('writes custom numeric settings when changed', async () => {
    const { wrapper, recordedWrites } = createSection({ performance: { profile: 'custom' } })
    const numberFields = wrapper.findAllComponents({ name: 'NumberField' })
    expect(numberFields).toHaveLength(4)

    pick(numberFields, 0).vm.$emit('commit', 128)
    pick(numberFields, 1).vm.$emit('commit', 12)
    pick(numberFields, 2).vm.$emit('commit', 4)
    pick(numberFields, 3).vm.$emit('commit', 8)

    expect(recordedWrites).toStrictEqual([
      { at: paths.performanceEmbeddingBatchSize, value: '128' },
      { at: paths.performanceOcrThreads, value: '12' },
      { at: paths.performanceOcrSessions, value: '4' },
      { at: paths.performanceLlmConcurrency, value: '8' },
    ])
  })

  it('ignores null values on custom numeric fields commit', async () => {
    const { wrapper, recordedWrites } = createSection({ performance: { profile: 'custom' } })
    const numberFields = wrapper.findAllComponents({ name: 'NumberField' })

    pick(numberFields, 0).vm.$emit('commit', null)
    pick(numberFields, 1).vm.$emit('commit', null)
    pick(numberFields, 2).vm.$emit('commit', null)
    pick(numberFields, 3).vm.$emit('commit', null)

    expect(recordedWrites).toHaveLength(0)
  })
})
