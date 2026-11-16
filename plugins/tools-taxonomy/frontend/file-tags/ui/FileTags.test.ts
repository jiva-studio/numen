import { afterEach, describe, expect, it } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { getTagColor } from '../../lib/color'
import FileTags from './FileTags.vue'

enableAutoUnmount(afterEach)

const TAGS = [
  'Port',
  'Sea',
  'Quay',
  'Dock',
  'Pier',
  'Harbour',
  'Jetty',
  'Wharf',
  'Berth',
  'Mole',
  'Slip',
  'Cove',
]

const mountTags = (count: number) =>
  mount(FileTags, { props: { tags: TAGS.slice(0, count) }, attachTo: document.body })

const getDots = (wrapper: ReturnType<typeof mountTags>) => wrapper.findAll('[data-tag-dot]')

describe('the tag dots of a file', () => {
  it('draw nothing for a file without tags', () => {
    const wrapper = mountTags(0)

    expect(wrapper.find('[data-tag-dots]').exists()).toBe(false)
    expect(getDots(wrapper)).toHaveLength(0)
  })

  it('draw one dot for each tag of a file that has up to four', () => {
    for (const count of [1, 2, 3, 4]) {
      expect(getDots(mountTags(count))).toHaveLength(count)
    }
  })

  it('draw four dots and no more circle for a file that has four tags', () => {
    const wrapper = mountTags(4)

    expect(getDots(wrapper)).toHaveLength(4)
    expect(wrapper.find('[data-tag-more]').exists()).toBe(false)
    expect(wrapper.get('[data-tag-dots]').text()).toBe('')
  })

  it('draw three dots and a more circle holding the rest for a file that has more', () => {
    for (const [count, rest] of [
      [5, '+2'],
      [6, '+3'],
      [12, '+9'],
    ] as const) {
      const wrapper = mountTags(count)

      expect(getDots(wrapper)).toHaveLength(3)
      expect(wrapper.findAll('[data-tag-more]')).toHaveLength(1)
      expect(wrapper.get('[data-tag-more]').text()).toBe(rest)
      expect(wrapper.get('[data-tag-dots]').text()).toBe(rest)
    }
  })

  it('draw the more circle inside the stack of dots, after the last dot, in no tag color', () => {
    const wrapper = mountTags(6)

    expect(wrapper.get('[data-tag-dots]').element.lastElementChild).toBe(
      wrapper.get('[data-tag-more]').element,
    )
    expect(wrapper.get('[data-tag-more]').attributes('style') ?? '').not.toContain('--numen-tag-')
  })

  it('name every tag of the file, not only the three drawn before the more circle', () => {
    const wrapper = mountTags(6)

    expect(wrapper.get('[data-tag-dots]').attributes('aria-label')).toBe(
      'Port, Sea, Quay, Dock, Pier, Harbour',
    )
  })

  it('name the tags in the order the file holds them', () => {
    const wrapper = mount(FileTags, { props: { tags: ['Sea', 'Port'] } })

    expect(wrapper.get('[data-tag-dots]').attributes('aria-label')).toBe('Sea, Port')
  })

  it('draw the first three tags of a long list, each in the color of its text', () => {
    const wrapper = mountTags(6)

    const drawn = getDots(wrapper).map((dot) => dot.attributes('style') ?? '')
    TAGS.slice(0, 3).forEach((tag, index) => {
      expect(drawn[index]).toContain(`--numen-tag-${getTagColor(tag)}`)
    })
  })

  it('draw a tag in the same color in every file', () => {
    const first = mount(FileTags, { props: { tags: ['Sea', 'Port'] } })
    const second = mount(FileTags, { props: { tags: ['Quay', 'Port'] } })

    expect(first.findAll('[data-tag-dot]')[1]?.attributes('style')).toBe(
      second.findAll('[data-tag-dot]')[1]?.attributes('style'),
    )
  })

  it('show a tooltip with every tag while the pointer is over them', async () => {
    const wrapper = mountTags(6)
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()

    await wrapper.get('[data-tag-dots]').trigger('mouseenter')

    const tooltips = document.body.querySelectorAll('[role="tooltip"]')
    expect(tooltips).toHaveLength(1)
    expect(tooltips[0]?.textContent?.trim()).toBe('Port, Sea, Quay, Dock, Pier, Harbour')
  })

  it('hide the tooltip when the pointer leaves them', async () => {
    const wrapper = mountTags(3)
    await wrapper.get('[data-tag-dots]').trigger('mouseenter')

    await wrapper.get('[data-tag-dots]').trigger('mouseleave')

    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()
  })
})
