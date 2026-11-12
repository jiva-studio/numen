import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SEARCH_DEBOUNCE_MS } from '../model/useSearchResults'
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

  it('names the field it is typed into and the list it opens', () => {
    const wrapper = mountPopover()

    expect(wrapper.get('input').attributes('aria-label')).toBe(words.quickLinkPlaceholder)
    expect(wrapper.get('[data-numen-searchable-list]').attributes('aria-label')).toBe(
      words.quickLinkLabel,
    )
    wrapper.unmount()
  })

  it('renders positioned at specified coordinates', () => {
    const wrapper = mountPopover({ at: { x: 250, y: 180 } })
    const popover = wrapper.get('.plex-quick-link')

    expect(popover.attributes('style')).toContain('left: 250px')
    expect(popover.attributes('style')).toContain('top: 180px')
    wrapper.unmount()
  })

  it('lists the create option first and the found notes below it', async () => {
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

    const options = wrapper.findAll('[role="option"]')
    expect(options[0]?.text()).toBe('Create "quantum"')
    expect(options[1]?.text()).toContain('Quantum Physics')
    wrapper.unmount()
  })

  it('offers no Create when a note has exactly the typed title, and puts that note in row 1', async () => {
    const searchMock = vi.fn().mockResolvedValue([
      { path: 'notes/books-about.md', title: 'Books about' },
      { path: 'notes/books.md', title: 'Books' },
    ])
    const wrapper = mountPopover({ search: searchMock })
    const input = wrapper.get('input')
    await input.setValue(' books ')
    await advanceHold()

    const options = wrapper.findAll('[role="option"]')
    expect(options.map((option) => option.text())).toEqual(['Books', 'Books about'])
    expect(options.some((option) => option.text().startsWith('Create'))).toBe(false)
    expect(options[0]?.attributes('aria-selected')).toBe('true')

    await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('select-note')).toStrictEqual([['notes/books.md']])
    expect(wrapper.emitted('create-note')).toBeUndefined()
    wrapper.unmount()
  })

  it('emits select-note on Enter after ArrowDown moved to a found note', async () => {
    const searchMock = vi.fn().mockResolvedValue([{ path: 'notes/g-mini.md', title: 'g-mini' }])
    const wrapper = mountPopover({ search: searchMock })

    const input = wrapper.get('input')
    await input.setValue('g-m')
    await advanceHold()

    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('select-note')).toStrictEqual([['notes/g-mini.md']])
    wrapper.unmount()
  })

  it('emits select-note when an existing search result is clicked', async () => {
    const wrapper = mountPopover()
    const input = wrapper.get('input')
    await input.setValue('test')
    await advanceHold()

    const results = wrapper.findAll('[role="option"]')
    const existingButton = results[1]
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

    // Rows: Create, Title topic, Existing Note; Create is active at first
    expect(wrapper.get('[role="option"]').attributes('aria-selected')).toBe('true')

    // ArrowUp cycles backwards to the last row
    await input.trigger('keydown', { key: 'ArrowUp' })
    await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('select-note')).toStrictEqual([['notes/existing.md']])

    // ArrowDown from the last row cycles to Create
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

    const options = wrapper.findAll('[role="option"]')
    const secondOption = options[1]
    await secondOption?.trigger('pointerenter')
    expect(secondOption?.attributes('aria-selected')).toBe('true')

    const createBtn = options[0]!
    await createBtn.trigger('pointerenter')
    expect(createBtn.attributes('aria-selected')).toBe('true')

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
    expect(wrapper.find('[role="listbox"]').exists()).toBe(true)

    // Clearing query
    await input.setValue('')
    await advanceHold()
    expect(wrapper.find('[role="listbox"]').exists()).toBe(false)

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
    const frame = wrapper.get('[data-numen-searchable-list]')
    frame.element.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    expect(wrapper.emitted('dismiss')).toBeUndefined()

    // Clicking outside popover
    const outsideElement = document.createElement('div')
    document.body.appendChild(outsideElement)
    outsideElement.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))

    expect(wrapper.emitted('dismiss')).toHaveLength(1)
    document.body.removeChild(outsideElement)
    wrapper.unmount()
  })

  describe('row order while results load', () => {
    type Found = { path: string; title: string }[]
    const found: Found = [
      { path: 'notes/books.md', title: 'Books' },
      { path: 'notes/gita.md', title: 'Bhagavad-gita' },
    ]
    const hold = () => {
      let resolve!: (value: Found) => void
      const promise = new Promise<Found>((res) => {
        resolve = res
      })
      return { promise, resolve }
    }
    const getTexts = (wrapper: ReturnType<typeof mountPopover>) =>
      wrapper.findAll('[role="option"]').map((option) => option.text())
    const getActiveOptionIndex = (wrapper: ReturnType<typeof mountPopover>) =>
      wrapper
        .findAll('[role="option"]')
        .findIndex((option) => option.attributes('aria-selected') === 'true')

    it('shows Create as row 1 and active while the search is in flight', async () => {
      const pending = hold()
      const wrapper = mountPopover({ search: () => pending.promise })

      await wrapper.get('input').setValue('B')
      await advanceHold()

      expect(getTexts(wrapper)).toEqual(['Create "B"'])
      expect(getActiveOptionIndex(wrapper)).toBe(0)
      wrapper.unmount()
    })

    it('keeps Create in row 1 and the active row unchanged when results arrive', async () => {
      const pending = hold()
      const wrapper = mountPopover({ search: () => pending.promise })
      await wrapper.get('input').setValue('B')
      await advanceHold()
      const before = wrapper.get('[role="option"]').element

      pending.resolve(found)
      await vi.advanceTimersByTimeAsync(0)

      expect(getTexts(wrapper)).toEqual(['Create "B"', 'Books', 'Bhagavad-gita'])
      expect(wrapper.get('[role="option"]').element).toBe(before)
      expect(getActiveOptionIndex(wrapper)).toBe(0)
      await wrapper.get('input').trigger('keydown', { key: 'Enter' })
      expect(wrapper.emitted('create-note')).toStrictEqual([['B']])
      expect(wrapper.emitted('select-note')).toBeUndefined()
      wrapper.unmount()
    })

    it('returns the active row to Create on typing and holds it while results arrive', async () => {
      const first = hold()
      const second = hold()
      const searchMock = vi
        .fn()
        .mockReturnValueOnce(first.promise)
        .mockReturnValueOnce(second.promise)
      const wrapper = mountPopover({ search: searchMock })
      const input = wrapper.get('input')
      await input.setValue('B')
      await advanceHold()
      first.resolve(found)
      await vi.advanceTimersByTimeAsync(0)
      await input.trigger('keydown', { key: 'ArrowDown' })
      expect(getActiveOptionIndex(wrapper)).toBe(1)

      await input.setValue('Bo')
      await advanceHold()
      expect(getActiveOptionIndex(wrapper)).toBe(0)
      second.resolve([found[0]!])
      await vi.advanceTimersByTimeAsync(0)

      expect(getTexts(wrapper)).toEqual(['Create "Bo"', 'Books'])
      expect(getActiveOptionIndex(wrapper)).toBe(0)
      wrapper.unmount()
    })

    it('switches row 1 in place to an exact match that arrives late, and Enter links it', async () => {
      const pending = hold()
      const wrapper = mountPopover({ search: () => pending.promise })
      await wrapper.get('input').setValue('Books')
      await advanceHold()
      expect(getTexts(wrapper)).toEqual(['Create "Books"'])
      const row = wrapper.get('[role="option"]').element

      pending.resolve([
        { path: 'notes/books-about.md', title: 'Books about' },
        { path: 'notes/books.md', title: 'Books' },
      ])
      await vi.advanceTimersByTimeAsync(0)

      expect(getTexts(wrapper)).toEqual(['Books', 'Books about'])
      expect(wrapper.get('[role="option"]').element).toBe(row)
      expect(getActiveOptionIndex(wrapper)).toBe(0)
      await wrapper.get('input').trigger('keydown', { key: 'Enter' })
      expect(wrapper.emitted('select-note')).toStrictEqual([['notes/books.md']])
      expect(wrapper.emitted('create-note')).toBeUndefined()
      wrapper.unmount()
    })

    it('links the exact match when Create is clicked after the match arrived', async () => {
      const pending = hold()
      const wrapper = mountPopover({ search: () => pending.promise })
      await wrapper.get('input').setValue('Books')
      await advanceHold()
      const create = wrapper.get('[role="option"]')

      pending.resolve([{ path: 'notes/books.md', title: 'Books' }])
      await vi.advanceTimersByTimeAsync(0)
      await create.trigger('click')

      expect(wrapper.emitted('select-note')).toStrictEqual([['notes/books.md']])
      expect(wrapper.emitted('create-note')).toBeUndefined()
      wrapper.unmount()
    })

    it('shows only Create when the search finds nothing', async () => {
      const pending = hold()
      const wrapper = mountPopover({ search: () => pending.promise })
      await wrapper.get('input').setValue('B')
      await advanceHold()

      pending.resolve([])
      await vi.advanceTimersByTimeAsync(0)

      expect(getTexts(wrapper)).toEqual(['Create "B"'])
      expect(getActiveOptionIndex(wrapper)).toBe(0)
      wrapper.unmount()
    })

    it('keeps the chosen found note active when later results still contain it', async () => {
      const searchMock = vi.fn().mockResolvedValue(found)
      const wrapper = mountPopover({ search: searchMock })
      await wrapper.get('input').setValue('B')
      await advanceHold()
      await wrapper.get('input').trigger('keydown', { key: 'ArrowDown' })
      await wrapper.get('input').trigger('keydown', { key: 'ArrowDown' })
      expect(getActiveOptionIndex(wrapper)).toBe(2)

      await wrapper.get('input').trigger('keydown', { key: 'Enter' })
      expect(wrapper.emitted('select-note')).toStrictEqual([['notes/gita.md']])
      wrapper.unmount()
    })
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
      expect(wrapper.find('[role="listbox"]').exists()).toBe(false)
      expect(isLoading(wrapper)).toBe(false)
      wrapper.unmount()
    })

    it('stops loading and empties the list when the search fails', async () => {
      const searchMock = vi.fn().mockRejectedValue(new Error('ipc'))
      const wrapper = mountPopover({ search: searchMock })

      await wrapper.get('input').setValue('a')
      await advanceHold()

      expect(isLoading(wrapper)).toBe(false)
      expect(wrapper.findAll('[role="option"]')).toHaveLength(1)
      wrapper.unmount()
    })
  })
})
