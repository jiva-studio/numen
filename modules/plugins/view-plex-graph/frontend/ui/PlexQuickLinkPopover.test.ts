import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SEARCH_DEBOUNCE_MS } from '../lib/search'
import PlexQuickLinkPopover from './PlexQuickLinkPopover.vue'
import { WORDS as words } from '../words'

describe('PlexQuickLinkPopover component', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  const advanceHold = async () => {
    await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS)
  }

  const mountPopover = (
    props: Partial<InstanceType<typeof PlexQuickLinkPopover>['$props']> = {},
  ) => {
    return mount(PlexQuickLinkPopover, {
      props: {
        at: { x: 200, y: 150 },
        relation: 'child',
        search: async (query: string) => [
          { path: `notes/${query}-topic.md`, title: `Title ${query}` },
          { path: 'notes/existing.md', title: 'Existing Note' },
        ],
        ...props,
      },
      attachTo: document.body,
    })
  }

  it('names the field it is typed into by its label', () => {
    const wrapper = mountPopover()
    const field = wrapper.get('input').element as HTMLInputElement

    expect([...(field.labels ?? [])].map((label) => label.textContent)).toEqual([
      words.quickLinkLabel,
    ])
    wrapper.unmount()
  })

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
    await advanceHold()

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
    await advanceHold()

    expect(wrapper.text()).toContain('g-mini')
    expect(wrapper.find('.plex-quick-link__option--create').exists()).toBe(false)
    wrapper.unmount()
  })

  it('emits select-note on Enter when search results exist', async () => {
    const searchMock = vi.fn().mockResolvedValue([{ path: 'notes/g-mini.md', title: 'g-mini' }])
    const wrapper = mountPopover({ search: searchMock })

    const input = wrapper.get('input')
    await input.setValue('g-mini')
    await advanceHold()

    await input.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('select-note')).toStrictEqual([['notes/g-mini.md']])
    wrapper.unmount()
  })

  it('emits select-note when an existing search result is clicked', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')
    await input.setValue('test')
    await advanceHold()

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
    await advanceHold()

    await input.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('create-note')).toStrictEqual([['Brand New Idea']])
    wrapper.unmount()
  })

  it('navigates options with ArrowDown and ArrowUp including cycle around', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')
    await input.setValue('topic')
    await advanceHold()

    // Initially at index 0
    // ArrowUp cycles backwards to index 2 (Create option)
    await input.trigger('keydown', { key: 'ArrowUp' })
    const createBtn = wrapper.find('.plex-quick-link__option--create')
    expect(createBtn.classes()).toContain('plex-quick-link__option--active')

    // ArrowUp moves to index 1
    await input.trigger('keydown', { key: 'ArrowUp' })
    await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('select-note')).toStrictEqual([['notes/existing.md']])

    // ArrowDown from index 1 moves to index 2
    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('create-note')).toStrictEqual([['topic']])

    wrapper.unmount()
  })

  it('updates highlightedIndex on pointerenter and triggers create on click', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')
    await input.setValue('abc')
    await advanceHold()

    const options = wrapper.findAll('.plex-quick-link__option')
    const secondOption = options[1]
    await secondOption?.trigger('pointerenter')
    expect(secondOption?.classes()).toContain('plex-quick-link__option--active')

    const createBtn = wrapper.get('.plex-quick-link__option--create')
    await createBtn.trigger('pointerenter')
    expect(createBtn.classes()).toContain('plex-quick-link__option--active')

    await createBtn.trigger('click')
    expect(wrapper.emitted('create-note')).toStrictEqual([['abc']])
    wrapper.unmount()
  })

  it('displays path when title is empty', async () => {
    const searchMock = vi.fn().mockResolvedValue([{ path: 'notes/untitled-note.md', title: '' }])
    const wrapper = mountPopover({ search: searchMock })

    const input = wrapper.get('input')
    await input.setValue('untitled')
    await advanceHold()

    expect(wrapper.text()).toContain('notes/untitled-note.md')
    wrapper.unmount()
  })

  it('handles clearing query, empty query arrow keys, and empty search results', async () => {
    const searchMock = vi.fn().mockResolvedValue([])
    const wrapper = mountPopover({ search: searchMock })
    const input = wrapper.get('input')

    // Arrow keys when empty (totalItems is 0)
    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.trigger('keydown', { key: 'ArrowUp' })

    // Typing with search returning no results
    await input.setValue('custom')
    await advanceHold()
    expect(wrapper.find('.plex-quick-link__dropdown').exists()).toBe(true)

    // Clearing query
    await input.setValue('')
    await advanceHold()
    expect(wrapper.find('.plex-quick-link__dropdown').exists()).toBe(false)

    wrapper.unmount()
  })

  it('emits dismiss on pressing Escape', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')

    await input.trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('dismiss')).toHaveLength(1)
    wrapper.unmount()
  })

  it('emits dismiss when clicking outside but not when clicking inside', async () => {
    const wrapper = mountPopover()

    // Clicking inside popover
    const popover = wrapper.get('.plex-quick-link')
    popover.element.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('dismiss')).toBeUndefined()

    // Clicking outside popover
    const outsideElement = document.createElement('div')
    document.body.appendChild(outsideElement)
    outsideElement.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))

    expect(wrapper.emitted('dismiss')).toHaveLength(1)
    document.body.removeChild(outsideElement)
    wrapper.unmount()
  })

  describe('search timing', () => {
    type Found = { path: string; title: string }[]
    const deferred = <T>() => {
      let resolve!: (value: T) => void
      const promise = new Promise<T>((res) => {
        resolve = res
      })
      return { promise, resolve }
    }
    const isLoading = (wrapper: ReturnType<typeof mountPopover>) =>
      (wrapper.vm.$ as unknown as { setupState: { isLoading: boolean } }).setupState.isLoading

    it('searches once for keystrokes inside the hold window', async () => {
      const searchMock = vi.fn().mockResolvedValue([])
      const wrapper = mountPopover({ search: searchMock })
      const input = wrapper.get('input')

      await input.setValue('q')
      await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS - 1)
      await input.setValue('qu')
      await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS - 1)
      expect(searchMock).not.toHaveBeenCalled()

      await vi.advanceTimersByTimeAsync(1)
      expect(searchMock).toHaveBeenCalledTimes(1)
      expect(searchMock).toHaveBeenCalledWith('qu')
      wrapper.unmount()
    })

    it('keeps the results of the later query when answers land out of order', async () => {
      const first = deferred<Found>()
      const second = deferred<Found>()
      const searchMock = vi
        .fn()
        .mockReturnValueOnce(first.promise)
        .mockReturnValueOnce(second.promise)
      const wrapper = mountPopover({ search: searchMock })
      const input = wrapper.get('input')

      await input.setValue('a')
      await advanceHold()
      await input.setValue('ab')
      await advanceHold()
      expect(searchMock).toHaveBeenCalledTimes(2)

      second.resolve([{ path: 'notes/later.md', title: 'Later' }])
      await vi.advanceTimersByTimeAsync(0)
      first.resolve([{ path: 'notes/earlier.md', title: 'Earlier' }])
      await vi.advanceTimersByTimeAsync(0)

      expect(wrapper.text()).toContain('Later')
      expect(wrapper.text()).not.toContain('Earlier')
      wrapper.unmount()
    })

    it('stays loading until the current search finishes', async () => {
      const first = deferred<Found>()
      const second = deferred<Found>()
      const searchMock = vi
        .fn()
        .mockReturnValueOnce(first.promise)
        .mockReturnValueOnce(second.promise)
      const wrapper = mountPopover({ search: searchMock })
      const input = wrapper.get('input')

      await input.setValue('a')
      await advanceHold()
      await input.setValue('ab')
      await advanceHold()

      first.resolve([{ path: 'notes/earlier.md', title: 'Earlier' }])
      await vi.advanceTimersByTimeAsync(0)
      expect(isLoading(wrapper)).toBe(true)
      expect(wrapper.text()).not.toContain('Earlier')

      second.resolve([])
      await vi.advanceTimersByTimeAsync(0)
      expect(isLoading(wrapper)).toBe(false)
      wrapper.unmount()
    })

    it('cancels the pending search and empties the list when the query is cleared', async () => {
      const searchMock = vi.fn().mockResolvedValue([{ path: 'notes/a.md', title: 'Alpha' }])
      const wrapper = mountPopover({ search: searchMock })
      const input = wrapper.get('input')

      await input.setValue('a')
      await advanceHold()
      expect(wrapper.text()).toContain('Alpha')
      expect(searchMock).toHaveBeenCalledTimes(1)

      await input.setValue('ab')
      await input.setValue('')
      await advanceHold()

      expect(searchMock).toHaveBeenCalledTimes(1)
      expect(wrapper.find('.plex-quick-link__dropdown').exists()).toBe(false)
      expect(isLoading(wrapper)).toBe(false)
      wrapper.unmount()
    })

    it('stops loading and empties the list when the search fails', async () => {
      const searchMock = vi.fn().mockRejectedValue(new Error('ipc'))
      const wrapper = mountPopover({ search: searchMock })

      await wrapper.get('input').setValue('a')
      await advanceHold()

      expect(isLoading(wrapper)).toBe(false)
      expect(wrapper.findAll('.plex-quick-link__option')).toHaveLength(1)
      wrapper.unmount()
    })
  })
})
