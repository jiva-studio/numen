/**
 * The panel of actions over the palette: closed until requested, closed again
 * when its item offers nothing, and a choice made in it never applies to an item
 * that is not highlighted.
 */
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, ref, type ComponentPublicInstance } from 'vue'
import { useActionPanelToggle } from './useActionPanelToggle'
import { revealInGroups } from './revealInGroups'
import { useActionPanel } from './useActionPanel'
import type { PaletteAction } from '../lib/item'

// A row of the panel is the component of an action, which exposes its element.
const createRow = (element: HTMLElement | null): ComponentPublicInstance =>
  ({ element }) as Partial<ComponentPublicInstance> as ComponentPublicInstance

const OPEN: PaletteAction = { id: 'open', text: 'Open note' }
const RENAME: PaletteAction = { id: 'rename', text: 'Rename note' }
const REMOVE: PaletteAction = { id: 'remove', text: 'Remove note' }

const setUp = (lit: string | undefined) => {
  const actions = ref<readonly PaletteAction[]>([OPEN])
  const choose = vi.fn<(item: string, action: string) => void>()
  const dismiss = vi.fn<() => void>()
  const panel = useActionPanelToggle({
    getActions: () => actions.value,
    getLit: () => lit,
    choose,
    dismiss,
  })
  return { actions, choose, dismiss, panel }
}

describe('useActionPanelToggle', () => {
  it('remains closed until it is opened', () => {
    expect(setUp('note').panel.open.value).toBe(false)
  })

  it('closes when item has no remaining actions', async () => {
    const { actions, panel } = setUp('note')
    panel.open.value = true
    actions.value = []
    await nextTick()
    expect(panel.open.value).toBe(false)
  })

  it('passes the lit item and the chosen action back', () => {
    const { choose, panel } = setUp('note')
    panel.chooseAction('open')
    expect(choose).toHaveBeenCalledExactlyOnceWith('note', 'open')
  })

  it('chooses nothing while no item is lit', () => {
    const { choose, panel } = setUp(undefined)
    panel.chooseAction('open')
    expect(choose).not.toHaveBeenCalled()
  })

  it('puts the panel away on a press, and leaves the palette standing', () => {
    const { dismiss, panel } = setUp('note')
    panel.open.value = true
    panel.onPress()
    expect(panel.open.value).toBe(false)
    expect(dismiss).not.toHaveBeenCalled()
  })

  it('takes the panel away first on the ground, and the palette on the next press', () => {
    const { dismiss, panel } = setUp('note')
    panel.open.value = true
    panel.onBackdrop()
    expect(panel.open.value).toBe(false)
    expect(dismiss).not.toHaveBeenCalled()
    panel.onBackdrop()
    expect(dismiss).toHaveBeenCalledOnce()
  })
})

describe('revealInGroups', () => {
  it('waits a tick and reveals the item in every group, or does nothing for null', async () => {
    const first = { reveal: vi.fn<(item: string) => void>() }
    const second = { reveal: vi.fn<(item: string) => void>() }

    const promise = revealInGroups([first, second], 'entropy')
    expect(first.reveal).not.toHaveBeenCalled()
    await promise

    expect(first.reveal).toHaveBeenCalledExactlyOnceWith('entropy')
    expect(second.reveal).toHaveBeenCalledExactlyOnceWith('entropy')
    await expect(revealInGroups(null, 'entropy')).resolves.toBeUndefined()
  })
})

function setUpActionPanelView(
  initialOpen = false,
  initialActions: readonly PaletteAction[] = [OPEN, RENAME, REMOVE],
  hasInput = true,
) {
  const actions = ref<readonly PaletteAction[]>(initialActions)
  const open = ref(initialOpen)
  const input = document.createElement('input')
  const focusSpy = vi.spyOn(input, 'focus')
  const searchInput = ref<HTMLInputElement | null>(hasInput ? input : null)
  const choose = vi.fn<(action: string) => void>()
  let view: ReturnType<typeof useActionPanel> | undefined
  const wrapper = mount(
    defineComponent({
      setup() {
        view = useActionPanel({
          getActions: () => actions.value,
          open,
          searchInput,
          choose,
        })
        return () => null
      },
    }),
  )
  if (!view) throw new Error('action panel view was not initialized')
  return { actions, open, focusSpy, choose, view, wrapper }
}

describe('useActionPanel view model', () => {
  it('starts with an empty query and no active action when closed, and ignores action list changes until opened', async () => {
    const { actions, open, focusSpy, view, wrapper } = setUpActionPanelView(false)

    expect(view.searchQuery.value).toBe('')
    expect(view.here.value).toBe(-1)
    expect(focusSpy).not.toHaveBeenCalled()

    actions.value = [RENAME, OPEN]
    await nextTick()
    expect(view.here.value).toBe(-1)

    view.searchQuery.value = 'stale'
    open.value = true
    await flushPromises()
    expect(view.searchQuery.value).toBe('')
    expect(view.here.value).toBe(0)
    expect(focusSpy).toHaveBeenCalledOnce()

    open.value = false
    await flushPromises()
    expect(view.here.value).toBe(-1)
    wrapper.unmount()
  })

  it('opens safely without a search input and resets activeActionId to empty when no actions match', async () => {
    const { actions, view, wrapper } = setUpActionPanelView(true, [], false)
    await flushPromises()
    expect(view.here.value).toBe(-1)

    actions.value = [OPEN, { id: 'Stryker was here!', text: 'Fallback' }]
    await nextTick()
    expect(view.here.value).toBe(0)
    wrapper.unmount()
  })

  it('tracks pointer movement across x and y and ignores stationary pointer events', () => {
    const first = setUpActionPanelView(true)
    first.view.onRowPointerMove(1, new PointerEvent('pointermove', { clientX: 1, clientY: -1 }))
    expect(first.view.here.value).toBe(1)
    first.view.onRowPointerMove(0, new PointerEvent('pointermove', { clientX: 1, clientY: -1 }))
    expect(first.view.here.value).toBe(1)
    first.view.onRowPointerMove(2, new PointerEvent('pointermove', { clientX: 1, clientY: 5 }))
    expect(first.view.here.value).toBe(2)
    first.view.onRowPointerMove(0, new PointerEvent('pointermove', { clientX: 5, clientY: 5 }))
    expect(first.view.here.value).toBe(0)
    first.wrapper.unmount()

    const second = setUpActionPanelView(true)
    second.view.onRowPointerMove(2, new PointerEvent('pointermove', { clientX: -1, clientY: 1 }))
    expect(second.view.here.value).toBe(2)
    second.wrapper.unmount()
  })

  it('chooses an in-bounds action and ignores an out-of-bounds index', () => {
    const { open, choose, view, wrapper } = setUpActionPanelView(true)

    view.onChooseAction(99)
    expect(choose).not.toHaveBeenCalled()
    expect(open.value).toBe(true)

    view.onChooseAction(1)
    expect(choose).toHaveBeenCalledExactlyOnceWith('rename')
    expect(open.value).toBe(false)
    wrapper.unmount()
  })

  it('handles keyboard navigation, Home, End, Enter, Escape, Tab, and row scrolling', async () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    try {
      const { open, choose, view, wrapper } = setUpActionPanelView(true)
      expect(view.rows.getActionName(2)).toBe(`${view.uid}-action-2`)

      const elRename = document.createElement('div')
      const scrollSpy = vi.fn()
      elRename.scrollIntoView = scrollSpy

      view.rows.holdRow('rename', createRow(elRename))
      const down = new KeyboardEvent('keydown', { key: 'ArrowDown', cancelable: true })
      view.onKey(down)
      expect(down.defaultPrevented).toBe(true)
      expect(view.here.value).toBe(1)
      await flushPromises()
      expect(scrollSpy).toHaveBeenCalledExactlyOnceWith({ block: 'nearest' })

      // Removing or replacing the row with non-RowHandle values removes it from the map.
      for (const invalidRow of [null, document.createElement('div'), {} as ComponentPublicInstance]) {
        view.rows.holdRow('rename', createRow(elRename))
        view.rows.holdRow('rename', invalidRow)
        scrollSpy.mockClear()
        view.onKey(new KeyboardEvent('keydown', { key: 'Home', cancelable: true }))
        view.onKey(new KeyboardEvent('keydown', { key: 'ArrowDown', cancelable: true }))
        await flushPromises()
        expect(scrollSpy).not.toHaveBeenCalled()
      }

      // RowHandle with null element does not throw when revealed.
      view.rows.holdRow('remove', createRow(null))
      const end = new KeyboardEvent('keydown', { key: 'End', cancelable: true })
      view.onKey(end)
      expect(end.defaultPrevented).toBe(true)
      expect(view.here.value).toBe(2)
      await flushPromises()

      const home = new KeyboardEvent('keydown', { key: 'Home', cancelable: true })
      view.onKey(home)
      expect(home.defaultPrevented).toBe(true)
      expect(view.here.value).toBe(0)
      await flushPromises()

      const tab = new KeyboardEvent('keydown', { key: 'Tab', cancelable: true })
      view.onKey(tab)
      expect(tab.defaultPrevented).toBe(true)
      expect(choose).not.toHaveBeenCalled()

      const letter = new KeyboardEvent('keydown', { key: 'a', cancelable: true })
      view.onKey(letter)
      expect(letter.defaultPrevented).toBe(false)

      const enter = new KeyboardEvent('keydown', { key: 'Enter', cancelable: true })
      view.onKey(enter)
      expect(enter.defaultPrevented).toBe(true)
      expect(choose).toHaveBeenCalledExactlyOnceWith('open')
      expect(open.value).toBe(false)

      open.value = true
      await flushPromises()
      const escape = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true })
      view.onKey(escape)
      expect(escape.defaultPrevented).toBe(true)
      expect(open.value).toBe(false)

      expect(errorSpy).not.toHaveBeenCalled()
      wrapper.unmount()
    } finally {
      errorSpy.mockRestore()
    }
  })
})
