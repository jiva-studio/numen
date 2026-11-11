import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import { win } from '@numen/ui'
import { useWindowEscape } from './useWindowEscape'

const press = (key: string): KeyboardEvent => {
  const event = new KeyboardEvent('keydown', { key, cancelable: true })
  win.dispatchEvent(event)
  return event
}

const mountEscape = () => {
  const isOpen = ref(false)
  const escapes: string[] = []
  const view = mount(
    defineComponent({
      setup() {
        useWindowEscape(
          () => isOpen.value,
          () => escapes.push('escape'),
        )
        return () => h('div')
      },
    }),
  )
  return { isOpen, escapes, view }
}

describe('the Escape key while a view is open', () => {
  it('is heard only while the view is open', async () => {
    const { isOpen, escapes } = mountEscape()

    press('Escape')
    isOpen.value = true
    await nextTick()
    press('Escape')
    isOpen.value = false
    await nextTick()
    press('Escape')

    expect(escapes).toHaveLength(1)
  })

  it('is kept from the page when it is heard', async () => {
    const { isOpen } = mountEscape()
    isOpen.value = true
    await nextTick()

    expect(press('Escape').defaultPrevented).toBe(true)
  })

  it('ignores other keys', async () => {
    const { isOpen, escapes } = mountEscape()
    isOpen.value = true
    await nextTick()

    expect(press('a').defaultPrevented).toBe(false)
    expect(escapes).toHaveLength(0)
  })

  it('is no longer heard after the view is unmounted', async () => {
    const { isOpen, escapes, view } = mountEscape()
    isOpen.value = true
    await nextTick()

    view.unmount()
    press('Escape')

    expect(escapes).toHaveLength(0)
  })
})
