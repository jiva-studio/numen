import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PlexQuickLinkPopover from './PlexQuickLinkPopover.vue'

describe('PlexQuickLinkPopover component', () => {
  const mountPopover = (
    props: Partial<InstanceType<typeof PlexQuickLinkPopover>['$props']> = {},
  ) => {
    return mount(PlexQuickLinkPopover, {
      props: {
        at: { x: 200, y: 150 },
        seat: 'child',
        search: async (query: string) => [
          { path: `notes/${query}-topic.md`, title: `Title ${query}` },
          { path: 'notes/existing.md', title: 'Existing Note' },
        ],
        ...props,
      },
      attachTo: document.body,
    })
  }

  it('renders positioned at specified coordinates', () => {
    const wrapper = mountPopover({ at: { x: 250, y: 180 } })
    const popover = wrapper.get('.plex-quick-link')

    expect(popover.attributes('style')).toContain('left: 250px')
    expect(popover.attributes('style')).toContain('top: 180px')
    wrapper.unmount()
  })

  it('searches for notes and shows existing matches before create option', async () => {
    const searchMock = vi
      .fn()
      .mockResolvedValue([{ path: 'physics/quantum.md', title: 'Quantum Physics' }])
    const wrapper = mountPopover({ search: searchMock })

    const input = wrapper.get('input')
    await input.setValue('quantum')
    await wrapper.vm.$nextTick()

    expect(searchMock).toHaveBeenCalledWith('quantum')
    expect(wrapper.text()).toContain('Quantum Physics')
    expect(wrapper.text()).toContain('Create "quantum"')

    const options = wrapper.findAll('.plex-quick-link__option')
    expect(options[0]?.text()).toContain('Quantum Physics')
    expect(options[1]?.text()).toContain('Create "quantum"')
    wrapper.unmount()
  })

  it('hides create option when there is an exact match', async () => {
    const searchMock = vi.fn().mockResolvedValue([{ path: 'notes/g-mini.md', title: 'g-mini' }])
    const wrapper = mountPopover({ search: searchMock })

    const input = wrapper.get('input')
    await input.setValue('g-mini')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('g-mini')
    expect(wrapper.find('.plex-quick-link__option--create').exists()).toBe(false)
    wrapper.unmount()
  })

  it('emits select-note on Enter when search results exist', async () => {
    const searchMock = vi.fn().mockResolvedValue([{ path: 'notes/g-mini.md', title: 'g-mini' }])
    const wrapper = mountPopover({ search: searchMock })

    const input = wrapper.get('input')
    await input.setValue('g-mini')
    await wrapper.vm.$nextTick()

    await input.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('select-note')).toStrictEqual([['notes/g-mini.md']])
    wrapper.unmount()
  })

  it('emits select-note when an existing search result is clicked', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')
    await input.setValue('test')
    await wrapper.vm.$nextTick()

    const results = wrapper.findAll('.plex-quick-link__option')
    const existingButton = results[0]
    expect(existingButton).toBeDefined()
    await existingButton?.trigger('click')

    expect(wrapper.emitted('select-note')).toStrictEqual([['notes/test-topic.md']])
    wrapper.unmount()
  })

  it('emits create-note on pressing Enter when there are no search results', async () => {
    const searchMock = vi.fn().mockResolvedValue([])
    const wrapper = mountPopover({ search: searchMock })

    const input = wrapper.get('input')
    await input.setValue('Brand New Idea')
    await wrapper.vm.$nextTick()

    await input.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('create-note')).toStrictEqual([['Brand New Idea']])
    wrapper.unmount()
  })

  it('navigates to create option with ArrowDown when search matches exist', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')
    await input.setValue('topic')
    await wrapper.vm.$nextTick()

    // 2 search results + 1 create option: move down twice to reach create option
    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('create-note')).toStrictEqual([['topic']])
    wrapper.unmount()
  })

  it('emits dismiss on pressing Escape', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')

    await input.trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('dismiss')).toHaveLength(1)
    wrapper.unmount()
  })

  it('emits dismiss when clicking outside the popover', async () => {
    const wrapper = mountPopover()

    const outsideElement = document.createElement('div')
    document.body.appendChild(outsideElement)

    outsideElement.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))

    expect(wrapper.emitted('dismiss')).toHaveLength(1)
    document.body.removeChild(outsideElement)
    wrapper.unmount()
  })
})
