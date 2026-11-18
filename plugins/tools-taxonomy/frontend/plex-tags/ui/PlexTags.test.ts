import { afterEach, describe, expect, it } from 'vitest'
import { TransitionGroup } from 'vue'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { getTagColor } from '../../lib/color'
import FileTags from '../../file-tags/ui/FileTags.vue'
import PlexTags from './PlexTags.vue'

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

const mountTags = (tags: readonly string[]) => mount(PlexTags, { props: { tags } })

const getChips = (wrapper: ReturnType<typeof mountTags>) => wrapper.findAll('[data-plex-tag]')

const getMore = (wrapper: ReturnType<typeof mountTags>) => wrapper.find('[data-plex-tag-more]')

const SHOWN = TAGS.slice(0, 4)

const getColor = (style: string | undefined) =>
  /--tag:\s*var\((--numen-tag-\d)\)/.exec(style ?? '')?.[1]

describe('the tags of a note in the plex', () => {
  it('draw nothing for a note without tags', () => {
    const wrapper = mountTags([])

    expect(wrapper.find('[data-plex-tags]').exists()).toBe(false)
    expect(getChips(wrapper)).toHaveLength(0)
  })

  it('draw a chip with the name of each tag', () => {
    const wrapper = mountTags(['Port', 'Sea', 'Quay'])

    expect(getChips(wrapper).map((chip) => chip.text())).toEqual(['Port', 'Sea', 'Quay'])
  })

  it('draw the chips in the order the note holds its tags', () => {
    const wrapper = mountTags(['Sea', 'Port'])

    expect(getChips(wrapper).map((chip) => chip.text())).toEqual(['Sea', 'Port'])
  })

  it('draw the first four tags and a +N chip for the rest of a note that has many', () => {
    const wrapper = mountTags(TAGS)

    expect(getChips(wrapper).map((chip) => chip.text())).toEqual(SHOWN)
    expect(wrapper.findAll('[data-plex-tag-more]')).toHaveLength(1)
    expect(getMore(wrapper).text()).toBe('+8')
  })

  it('draw four tags and no +N chip for a note with four tags', () => {
    const wrapper = mountTags(SHOWN)

    expect(getChips(wrapper)).toHaveLength(4)
    expect(getMore(wrapper).exists()).toBe(false)
    expect(wrapper.get('[data-plex-tags]').text()).not.toMatch(/\+\d/)
  })

  it('draw four tags and +1 for a note with five tags', () => {
    const wrapper = mountTags(TAGS.slice(0, 5))

    expect(getChips(wrapper)).toHaveLength(4)
    expect(getMore(wrapper).text()).toBe('+1')
  })

  it('draw four tags and +2 for a note with six tags', () => {
    const wrapper = mountTags(TAGS.slice(0, 6))

    expect(getChips(wrapper)).toHaveLength(4)
    expect(getMore(wrapper).text()).toBe('+2')
  })

  it('draw the +N chip in no tag color and keep it out of the tag chips', () => {
    const more = getMore(mountTags(TAGS))

    expect(more.exists()).toBe(true)
    expect(more.attributes('data-plex-tag')).toBeUndefined()
    expect(more.attributes('style') ?? '').not.toMatch(/--numen-tag-/)
  })

  it('keep the spelling of a tag in its chip', () => {
    const wrapper = mountTags(['#Harbour/Quay', 'port'])

    expect(getChips(wrapper).map((chip) => chip.text())).toEqual(['#Harbour/Quay', 'port'])
  })

  it('draw each chip in the color of the text of its tag', () => {
    const wrapper = mountTags(TAGS)

    SHOWN.forEach((tag, index) => {
      expect(getColor(getChips(wrapper)[index]?.attributes('style'))).toBe(
        `--numen-tag-${getTagColor(tag)}`,
      )
    })
  })

  it('draw a tag in the same color in every note', () => {
    const first = mountTags(['Sea', 'Port'])
    const second = mountTags(['Quay', 'Port'])

    expect(getColor(getChips(first)[1]?.attributes('style'))).toBeDefined()
    expect(getChips(first)[1]?.attributes('style')).toBe(getChips(second)[1]?.attributes('style'))
  })

  it('draw a tag in the same color as the dot of the file tree for the same text', () => {
    const chips = getChips(mountTags(TAGS))

    SHOWN.forEach((tag, index) => {
      const dot = mount(FileTags, { props: { tags: [tag] } }).get('[data-tag-dot]')

      expect(getColor(chips[index]?.attributes('style'))).toBeDefined()
      expect(getColor(chips[index]?.attributes('style'))).toBe(getColor(dot.attributes('style')))
    })
  })

  it('draw two spellings of one tag in the same color', () => {
    const [upper, lower] = getChips(mountTags(['Harbour', '#harbour']))

    expect(getColor(upper?.attributes('style'))).toBeDefined()
    expect(getColor(upper?.attributes('style'))).toBe(getColor(lower?.attributes('style')))
  })

  it('draw the chips without a transition group, a move, a leave or a delay by position', () => {
    const wrapper = mountTags(TAGS)

    expect(wrapper.findComponent(TransitionGroup).exists()).toBe(false)
    expect(wrapper.html()).not.toMatch(/-(enter|leave|move)\b/)
    for (const chip of [...getChips(wrapper), getMore(wrapper)]) {
      expect(chip.attributes('style') ?? '').not.toMatch(/--i\b|delay/)
    }
  })

  it('keep the same elements when the note is drawn again with the same tags', async () => {
    const wrapper = mountTags(['Port', 'Sea', 'Quay'])
    const before = getChips(wrapper).map((chip) => chip.element)
    const html = wrapper.html()

    await wrapper.setProps({ tags: ['Port', 'Sea', 'Quay'] })

    expect(getChips(wrapper).map((chip) => chip.element)).toEqual(before)
    expect(wrapper.html()).toBe(html)
  })

  it('remove the chip of a tag the moment the tag goes, leaving no element behind', async () => {
    const wrapper = mount(PlexTags, {
      props: { tags: ['Port', 'Sea', 'Quay'] },
      global: { stubs: { transition: false, 'transition-group': false } },
    })

    await wrapper.setProps({ tags: ['Port', 'Quay'] })

    expect(getChips(wrapper).map((chip) => chip.text())).toEqual(['Port', 'Quay'])
    expect(wrapper.text()).not.toContain('Sea')
  })

  it('show no tooltip while the pointer is over a tag chip', async () => {
    const wrapper = mountTags(['Port', 'Container terminal operations'])
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()

    const chips = getChips(wrapper)
    expect(chips).toHaveLength(2)
    await chips[1]!.trigger('mouseenter')

    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()
  })

  it('hide the tooltip of the +N chip when the pointer leaves it', async () => {
    const wrapper = mountTags(TAGS.slice(0, 7))
    await getMore(wrapper).trigger('mouseenter')
    expect(document.body.querySelector('[role="tooltip"]')?.textContent?.trim()).toBe(
      'Pier, Harbour, Jetty',
    )

    await getMore(wrapper).trigger('mouseleave')

    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()
  })

  it('hide the tooltip of the +N chip when the note no longer has hidden tags', async () => {
    const wrapper = mountTags(TAGS.slice(0, 7))
    await getMore(wrapper).trigger('mouseenter')
    expect(document.body.querySelector('[role="tooltip"]')?.textContent?.trim()).toBe(
      'Pier, Harbour, Jetty',
    )

    await wrapper.setProps({ tags: TAGS.slice(0, 4) })

    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()
  })

  it('hide the tooltip once the note has no tags', async () => {
    const wrapper = mountTags(TAGS.slice(0, 7))
    await getMore(wrapper).trigger('mouseenter')
    expect(document.body.querySelector('[role="tooltip"]')?.textContent?.trim()).toBe(
      'Pier, Harbour, Jetty',
    )
    await wrapper.setProps({ tags: [] })

    await wrapper.setProps({ tags: TAGS.slice(0, 7) })

    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()
  })

  it('list the names of the tags beyond the chips in a tooltip while the pointer is over the +N chip', async () => {
    const wrapper = mountTags(TAGS.slice(0, 7))

    await getMore(wrapper).trigger('mouseenter')

    const tooltips = document.body.querySelectorAll('[role="tooltip"]')
    expect(tooltips).toHaveLength(1)
    expect(tooltips[0]?.textContent?.trim()).toBe('Pier, Harbour, Jetty')
  })

  it('list the names of the tags beyond the chips while the +N chip has the focus', async () => {
    const wrapper = mountTags(TAGS.slice(0, 5))

    expect(getMore(wrapper).attributes('tabindex')).toBe('0')
    await getMore(wrapper).trigger('focus')
    expect(document.body.querySelector('[role="tooltip"]')?.textContent?.trim()).toBe('Pier')

    await getMore(wrapper).trigger('blur')
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()
  })
})
