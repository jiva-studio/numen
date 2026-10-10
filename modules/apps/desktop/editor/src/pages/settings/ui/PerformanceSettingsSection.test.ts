import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/entities/settings'
import PerformanceSettingsSection from './PerformanceSettingsSection.vue'
import type { Installation, SettingsTabState } from '../types'
import { WORDS as words } from '../words'
import paths from '../paths.json'

const createSection = (file: Record<string, unknown> = {}) => {
  const written: SettingEdit[] = []
  const installation: Installation = {
    getSetting: (path) => at(file, path),
    getModels: () => [],
    write: (said) => void written.push(...said),
    file: ref('/numen.json'),
    openFile: () => {},
    themes: ref([]),
    applied: ref('preset:numen'),
    mode: ref('system'),
    isPinned: ref(false),
    sizes: ref({ interfaceScale: 1, textScale: 1 }),
    bounds: ref({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    choose: () => {},
    isSyncing: ref(true),
    isHanging: ref(true),
    parts: ref(6),
    partsBounds: ref({ least: 2, most: 9 }),
    chooseParts: () => {},
    dayStarts: ref('04:00'),
    latestDayStarts: ref('12:00'),
    chooseDayStarts: () => {},
  }
  const state: SettingsTabState = { installation }
  const wrapper = mount(PerformanceSettingsSection, {
    props: { state },
  })
  return { written, wrapper }
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
    const { wrapper, written } = createSection()
    const segmented = wrapper.findComponent({ name: 'SegmentedControl' })
    segmented.vm.$emit('update:modelValue', 'maximum')
    expect(written).toStrictEqual([{ at: paths.performanceProfile, value: '"maximum"' }])
  })

  it('writes custom numeric settings when changed', async () => {
    const { wrapper, written } = createSection({ performance: { profile: 'custom' } })
    const numberFields = wrapper.findAllComponents({ name: 'NumberField' })
    expect(numberFields).toHaveLength(4)

    numberFields[0]!.vm.$emit('settle', 128)
    numberFields[1]!.vm.$emit('settle', 12)
    numberFields[2]!.vm.$emit('settle', 4)
    numberFields[3]!.vm.$emit('settle', 8)

    expect(written).toStrictEqual([
      { at: paths.performanceEmbeddingBatchSize, value: '128' },
      { at: paths.performanceOcrThreads, value: '12' },
      { at: paths.performanceOcrSessions, value: '4' },
      { at: paths.performanceLlmConcurrency, value: '8' },
    ])
  })

  it('ignores null values on custom numeric fields settle', async () => {
    const { wrapper, written } = createSection({ performance: { profile: 'custom' } })
    const numberFields = wrapper.findAllComponents({ name: 'NumberField' })

    numberFields[0]!.vm.$emit('settle', null)
    numberFields[1]!.vm.$emit('settle', null)
    numberFields[2]!.vm.$emit('settle', null)
    numberFields[3]!.vm.$emit('settle', null)

    expect(written).toHaveLength(0)
  })
})
