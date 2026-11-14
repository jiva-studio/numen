import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { Stats } from '../api/stats'
import StatsIndicator from './StatsIndicator.vue'

const STATS: Stats = {
  vectors: 12480,
  chunks: 13100,
  waiting: 620,
  sources: 84,
  kinds: [
    { kind: 'notes', sources: 70, chunks: 4100 },
    { kind: 'pdf', sources: 0, chunks: 0 },
  ],
  model: 'bge-m3',
  recipe: 'local|bge-m3|1024|512|mean|int8@0.4',
  dimensions: 1024,
  profile: 'balanced',
  threads: 4,
  speed: 18.5,
  databaseBytes: 73400320,
  vectorBytes: 52428800,
}

const draw = (stats: Stats | null, phase: 'loading' | 'ready' | 'unavailable' = 'ready') =>
  mount(StatsIndicator, { props: { stats, phase, locale: 'en-US' }, attachTo: document.body })

describe('the index stats indicator', () => {
  it('keeps the mark and the figure in their own cells in every state', () => {
    for (const view of [draw(STATS), draw({ ...STATS, waiting: 0 }), draw(null, 'loading')]) {
      expect(view.find('.index-stats__trigger > .index-stats__icon').exists()).toBe(true)
      expect(view.find('.index-stats__trigger > .index-stats__figure').exists()).toBe(true)
    }
  })

  it('is out of the tab order', () => {
    expect(draw(STATS).find('.index-stats__trigger').attributes('tabindex')).toBe('-1')
  })

  it('shows the vectors alone, with no end value, while chunks wait', () => {
    const wrapper = draw(STATS)
    const trigger = wrapper.find('.index-stats__trigger')

    expect(trigger.text()).toBe('⬡12,480')
    expect(trigger.text()).not.toContain('/')
    expect(trigger.attributes('aria-label')).toBe('12,480 vectors, indexing')
    expect(trigger.attributes('title')).toBe('12,480 vectors, indexing')
    expect(trigger.classes()).toStrictEqual(['index-stats__trigger'])
    expect(wrapper.find('.index-stats__icon').classes()).toStrictEqual(['index-stats__icon'])
  })

  it('shows the vectors alone and a still mark once the queue is empty', () => {
    const wrapper = draw({ ...STATS, vectors: 13100, waiting: 0 })

    expect(wrapper.find('.index-stats__trigger').text()).toBe('⬡13,100')
    expect(wrapper.find('.index-stats__trigger').attributes('aria-label')).toBe('13,100 vectors')
  })

  it('draws the mark and a dash before the first answer', () => {
    const view = draw(null, 'loading')
    expect(view.find('.index-stats').exists()).toBe(true)
    expect(view.find('.index-stats__trigger').text()).toBe('⬡–')
  })

  it('shows a dash where the index has nothing to show', () => {
    const wrapper = draw(null, 'unavailable')

    expect(wrapper.find('.index-stats__trigger').text()).toBe('⬡–')
  })

  it('opens the popover on a click, with the rows, and closes it on Escape', async () => {
    const wrapper = draw(STATS)
    expect(wrapper.find('.index-popover').exists()).toBe(false)

    await wrapper.find('.index-stats__trigger').trigger('click')

    expect(wrapper.find('.index-stats__trigger').attributes('aria-expanded')).toBe('true')
    const labels = wrapper.findAll('.index-popover__label').map((one) => one.text())
    expect(labels).toContain('Vectors')
    expect(labels).toContain('Chunks')
    expect(labels).toContain('Notes')
    expect(labels).not.toContain('PDF')

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.index-popover').exists()).toBe(false)
    wrapper.unmount()
  })

  it('keeps the popover open when the window is resized', async () => {
    const wrapper = draw(STATS)
    await wrapper.find('.index-stats__trigger').trigger('click')

    window.dispatchEvent(new Event('resize'))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.index-popover').exists()).toBe(true)
    wrapper.unmount()
  })

  it('closes the popover with its close button', async () => {
    const wrapper = draw(STATS)
    await wrapper.find('.index-stats__trigger').trigger('click')

    await wrapper.find('.index-popover__close-btn').trigger('click')

    expect(wrapper.find('.index-popover').exists()).toBe(false)
    wrapper.unmount()
  })
})
