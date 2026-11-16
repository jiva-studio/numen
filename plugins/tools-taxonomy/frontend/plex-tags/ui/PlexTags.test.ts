import { afterEach, describe, expect, it } from 'vitest'
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

const getColor = (style: string | undefined) => /--tag:\s*var\((--numen-tag-\d)\)/.exec(style ?? '')?.[1]

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

  it('draw a chip for every tag of a note that has many, with no count of the rest', () => {
    const wrapper = mountTags(TAGS)

    expect(getChips(wrapper)).toHaveLength(TAGS.length)
    expect(wrapper.get('[data-plex-tags]').text()).not.toMatch(/\+\d/)
  })

  it('keep the spelling of a tag in its chip', () => {
    const wrapper = mountTags(['#Harbour/Quay', 'port'])

    expect(getChips(wrapper).map((chip) => chip.text())).toEqual(['#Harbour/Quay', 'port'])
  })

  it('draw each chip in the color of the text of its tag', () => {
    const wrapper = mountTags(TAGS)

    TAGS.forEach((tag, index) => {
      expect(getColor(getChips(wrapper)[index]?.attributes('style'))).toBe(
        `--numen-tag-${getTagColor(tag)}`,
      )
    })
  })

  it('draw a tag in the same color in every note', () => {
    const first = mountTags(['Sea', 'Port'])
    const second = mountTags(['Quay', 'Port'])

    expect(getChips(first)[1]?.attributes('style')).toBe(getChips(second)[1]?.attributes('style'))
  })

  it('draw a tag in the same color as the dot of the file tree for the same text', () => {
    const chips = getChips(mountTags(TAGS))

    TAGS.forEach((tag, index) => {
      const dot = mount(FileTags, { props: { tags: [tag] } }).get('[data-tag-dot]')

      expect(getColor(chips[index]?.attributes('style'))).toBeDefined()
      expect(getColor(chips[index]?.attributes('style'))).toBe(getColor(dot.attributes('style')))
    })
  })

  it('draw two spellings of one tag in the same color', () => {
    const [upper, lower] = getChips(mountTags(['Harbour', '#harbour']))

    expect(getColor(upper?.attributes('style'))).toBe(getColor(lower?.attributes('style')))
  })
})
