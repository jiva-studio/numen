/**
 * The keyboard while the palette stands: what each key does and what it leaves
 * alone, where the keyboard goes as the palette opens or a step begins, and
 * where it goes back to.
 */
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref, shallowRef } from 'vue'
import { usePaletteKeys, type PaletteFieldHandle, type PaletteKeysState } from './keys'
import { usePalettePlaces, type PalettePlacesState } from './places'
import type { PaletteAction, PaletteGroup, PaletteItem } from '../lib/item'

const OPEN: PaletteAction = { id: 'open', text: 'Open the note' }

const item = (id: string): PaletteItem => ({ id, title: id, actions: [OPEN] })

const GROUPS: PaletteGroup[] = [{ id: 'g', title: 'g', items: [item('a'), item('b'), item('c')] }]

interface SetUpOptions {
  readonly open?: boolean
  readonly opensOn?: string
  readonly hasField?: boolean
  readonly typed?: string
}

const mounted: { unmount: () => void }[] = []
const backs: HTMLElement[] = []

afterEach(() => {
  mounted.splice(0).forEach((one) => one.unmount())
  backs.splice(0).forEach((one) => one.remove())
})

const setUp = (setUpOptions: SetUpOptions = {}) => {
  const isOpen = ref(setUpOptions.open ?? false)
  const step = ref('one')
  const opensOn = ref(setUpOptions.opensOn ?? '')
  const searchQuery = ref(setUpOptions.typed ?? '')
  const isPanelOpen = ref(false)
  const handle = {
    focus: vi.fn<() => void>(),
    select: vi.fn<() => void>(),
  } satisfies PaletteFieldHandle
  const field = shallowRef<PaletteFieldHandle | null>(
    setUpOptions.hasField === false ? null : handle,
  )
  const reveal = vi.fn<() => void>()
  const goBack = vi.fn<() => void>()
  const dismiss = vi.fn<() => void>()
  const back = document.createElement('button')
  document.body.append(back)
  backs.push(back)
  const errors: unknown[] = []
  let keys!: PaletteKeysState
  let places!: PalettePlacesState

  const wrapper = mount(
    defineComponent({
      setup() {
        places = usePalettePlaces({
          groups: () => GROUPS,
          typed: searchQuery,
          tellLit: () => undefined,
          tellChoice: () => undefined,
        })
        keys = usePaletteKeys({
          field,
          reveal,
          places,
          isPanelOpen,
          getOfferedActions: () => places.offered.value,
          typed: searchQuery,
          isOpen: () => isOpen.value,
          getStep: () => step.value,
          getOpensOn: () => opensOn.value,
          getOpenedFrom: () => back,
          goBack,
          dismiss,
        })
        return () => null
      },
    }),
    { global: { config: { errorHandler: (error: unknown) => errors.push(error) } } },
  )
  mounted.push(wrapper)

  const press = (key: string, init: KeyboardEventInit = {}): KeyboardEvent => {
    const event = new KeyboardEvent('keydown', { key, cancelable: true, ...init })
    keys.onKey(event)
    return event
  }

  return {
    isOpen,
    step,
    opensOn,
    typed: searchQuery,
    isPanelOpen,
    handle,
    field,
    reveal,
    goBack,
    dismiss,
    back,
    errors,
    places,
    press,
    wrapper,
  }
}

describe('usePaletteKeys', () => {
  describe('the keys', () => {
    it('step through the places, claim the key, and bring what is lit into sight', () => {
      const { places, press, reveal } = setUp({ open: true })
      places.goTo(0)
      reveal.mockClear()
      const down = press('ArrowDown')
      expect(down.defaultPrevented).toBe(true)
      expect(places.lit.value?.id).toBe('b')
      expect(reveal).toHaveBeenCalledOnce()
      const up = press('ArrowUp')
      expect(up.defaultPrevented).toBe(true)
      expect(places.lit.value?.id).toBe('a')
    })

    it('go to the first place on Home and the last on End', () => {
      const { places, press } = setUp({ open: true })
      places.goTo(1)
      expect(press('End').defaultPrevented).toBe(true)
      expect(places.lit.value?.id).toBe('c')
      expect(press('Home').defaultPrevented).toBe(true)
      expect(places.lit.value?.id).toBe('a')
    })

    it('claim Enter', () => {
      const { press } = setUp({ open: true })
      expect(press('Enter').defaultPrevented).toBe(true)
    })

    it('send Backspace in an empty field back a step, and leave it to the field otherwise', () => {
      const empty = setUp({ open: true })
      expect(empty.press('Backspace').defaultPrevented).toBe(true)
      expect(empty.goBack).toHaveBeenCalledOnce()
      const typing = setUp({ open: true, typed: 'n' })
      expect(typing.press('Backspace').defaultPrevented).toBe(false)
      expect(typing.goBack).not.toHaveBeenCalled()
    })

    it('dismiss the palette on Escape', () => {
      const { press, dismiss } = setUp({ open: true })
      expect(press('Escape').defaultPrevented).toBe(true)
      expect(dismiss).toHaveBeenCalledOnce()
    })

    it('keep the keyboard in the field on Tab', () => {
      const { press, dismiss } = setUp({ open: true })
      expect(press('Tab').defaultPrevented).toBe(true)
      expect(dismiss).not.toHaveBeenCalled()
    })

    it('leave every other key to the field', () => {
      const { press, dismiss, goBack } = setUp({ open: true })
      const event = press('a')
      expect(event.defaultPrevented).toBe(false)
      expect(dismiss).not.toHaveBeenCalled()
      expect(goBack).not.toHaveBeenCalled()
    })
  })

  describe('the field as the panel over it goes', () => {
    it('takes the keyboard back when the panel closes over an open palette', async () => {
      const { isPanelOpen, handle } = setUp({ open: true })
      await flushPromises()
      isPanelOpen.value = true
      await flushPromises()
      expect(handle.focus).toHaveBeenCalledTimes(1)
      isPanelOpen.value = false
      await flushPromises()
      expect(handle.focus).toHaveBeenCalledTimes(2)
    })

    it('does not take it while the panel opens', async () => {
      const { isPanelOpen, handle } = setUp({ open: true })
      await flushPromises()
      handle.focus.mockClear()
      isPanelOpen.value = true
      await flushPromises()
      expect(handle.focus).not.toHaveBeenCalled()
    })

    it('does not take it when the palette goes with the panel', async () => {
      const { isOpen, isPanelOpen, handle } = setUp({ open: true })
      await flushPromises()
      isPanelOpen.value = true
      await flushPromises()
      handle.focus.mockClear()
      isOpen.value = false
      await flushPromises()
      expect(handle.focus).not.toHaveBeenCalled()
    })

    it('survives a field that is not there', async () => {
      const { isPanelOpen, errors } = setUp({ open: true, hasField: false })
      await flushPromises()
      isPanelOpen.value = true
      await flushPromises()
      isPanelOpen.value = false
      await flushPromises()
      expect(errors).toEqual([])
    })
  })

  describe('as the palette opens', () => {
    it('puts the keyboard on the field, selects it and brings the lit item into sight', async () => {
      const { isOpen, handle, reveal, isPanelOpen } = setUp()
      isPanelOpen.value = true
      isOpen.value = true
      await flushPromises()
      expect(isPanelOpen.value).toBe(false)
      expect(handle.focus).toHaveBeenCalled()
      expect(handle.select).toHaveBeenCalledOnce()
      expect(reveal).toHaveBeenCalledOnce()
    })

    it('relies on the item the step opens on', async () => {
      const { isOpen, places, opensOn } = setUp()
      opensOn.value = 'b'
      places.held.value = 'c'
      isOpen.value = true
      await flushPromises()
      expect(places.lit.value?.id).toBe('b')
    })

    it('relies on the item it was on when the step names none', async () => {
      const { isOpen, places } = setUp()
      places.held.value = 'c'
      isOpen.value = true
      await flushPromises()
      expect(places.lit.value?.id).toBe('c')
    })

    it('relies on the first item when nothing is named or retained', async () => {
      const { isOpen, places } = setUp()
      isOpen.value = true
      await flushPromises()
      expect(places.lit.value?.id).toBe('a')
    })

    it('carries on without a field', async () => {
      const { isOpen, reveal, errors } = setUp({ hasField: false })
      isOpen.value = true
      await flushPromises()
      expect(errors).toEqual([])
      expect(reveal).toHaveBeenCalledOnce()
    })

    it('does the same when it is mounted open', async () => {
      const { handle, reveal } = setUp({ open: true })
      await flushPromises()
      expect(handle.select).toHaveBeenCalledOnce()
      expect(reveal).toHaveBeenCalledOnce()
    })

    it('does nothing when it is mounted closed', async () => {
      const { handle, reveal } = setUp()
      await flushPromises()
      expect(handle.focus).not.toHaveBeenCalled()
      expect(reveal).not.toHaveBeenCalled()
    })
  })

  describe('as a step begins', () => {
    it('selects what remains in the field while the palette is open', async () => {
      const { step, handle } = setUp({ open: true })
      await flushPromises()
      handle.select.mockClear()
      step.value = 'two'
      await flushPromises()
      expect(handle.select).toHaveBeenCalledOnce()
    })

    it('does nothing while the palette is closed', async () => {
      const { step, handle, reveal } = setUp()
      step.value = 'two'
      await flushPromises()
      expect(handle.select).not.toHaveBeenCalled()
      expect(reveal).not.toHaveBeenCalled()
    })
  })

  describe('as the palette closes', () => {
    it('lets go of the panel and what was lit, and goes back to where it came from', async () => {
      const { isOpen, isPanelOpen, places, back } = setUp({ open: true })
      await flushPromises()
      isPanelOpen.value = true
      isOpen.value = false
      await flushPromises()
      expect(isPanelOpen.value).toBe(false)
      expect(places.held.value).toBe('')
      expect(document.activeElement).toBe(back)
    })

    it('does not go back to an element that has left the document', async () => {
      const { isOpen, back } = setUp({ open: true })
      back.remove()
      const focus = vi.spyOn(back, 'focus')
      isOpen.value = false
      await flushPromises()
      expect(focus).not.toHaveBeenCalled()
    })
  })

  describe('as the palette is taken away', () => {
    it('goes back to where it came from when it was open', async () => {
      const { wrapper, back } = setUp({ open: true })
      await flushPromises()
      wrapper.unmount()
      expect(document.activeElement).toBe(back)
    })

    it('leaves the keyboard alone when it was closed', () => {
      const { wrapper, back, places } = setUp()
      places.held.value = 'b'
      wrapper.unmount()
      expect(document.activeElement).not.toBe(back)
      expect(places.held.value).toBe('b')
    })
  })
})
