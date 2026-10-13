/**
 * What opening a menu puts in place and what closing it takes away, without a
 * menu to draw.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import { useMenuOpen } from './open'

const create = (start: boolean, from: HTMLElement | null = null) => {
  const isOpen = ref(start)
  const calls: string[] = []
  const wrapper = mount(
    defineComponent({
      setup() {
        useMenuOpen({
          isOpen: () => isOpen.value,
          listen: () => calls.push('listen'),
          release: () => calls.push('release'),
          measure: () => calls.push('measure'),
          focusInitial: () => calls.push('focusInitial'),
          reset: () => calls.push('reset'),
          getFrom: () => from,
        })
        return () => h('div')
      },
    }),
  )
  return { isOpen, calls, wrapper }
}

describe('opening and closing', () => {
  it('listens at once and measures and takes the keyboard a tick later', async () => {
    const { calls } = create(true)

    expect(calls).toStrictEqual(['listen'])
    await nextTick()
    expect(calls).toStrictEqual(['listen', 'measure', 'focusInitial'])
  })

  it('does nothing for a menu that was never opened', async () => {
    const { calls } = create(false)

    await nextTick()

    expect(calls).toStrictEqual([])
  })

  it('lets go of the window and forgets the keyboard as it closes', async () => {
    const { isOpen, calls } = create(true)
    await nextTick()
    calls.length = 0

    isOpen.value = false
    await nextTick()

    expect(calls).toStrictEqual(['release', 'reset'])
  })

  it('stays put for an open that repeats', async () => {
    const { isOpen, calls } = create(true)
    await nextTick()

    isOpen.value = true
    await nextTick()

    expect(calls.filter((call) => call === 'listen')).toHaveLength(1)
  })

  it('gives the keyboard back to where it came from, while that is on the page', async () => {
    const from = document.createElement('button')
    document.body.append(from)
    const focus = vi.spyOn(from, 'focus')
    const { isOpen } = create(true, from)
    await nextTick()

    isOpen.value = false
    await nextTick()

    expect(focus).toHaveBeenCalledTimes(1)
    from.remove()
  })

  it('does not hand the keyboard to an element that has left the page', async () => {
    const from = document.createElement('button')
    const focus = vi.spyOn(from, 'focus')
    const { isOpen } = create(true, from)
    await nextTick()

    isOpen.value = false
    await nextTick()

    expect(focus).not.toHaveBeenCalled()
  })

  it('lets go of the window when it goes while still open', async () => {
    const { calls, wrapper } = create(true)
    await nextTick()
    calls.length = 0

    wrapper.unmount()

    expect(calls).toStrictEqual(['release', 'reset'])
  })
})
