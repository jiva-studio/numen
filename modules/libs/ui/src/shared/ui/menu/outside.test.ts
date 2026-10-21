/**
 * What a press outside an element evaluates to.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, shallowRef } from 'vue'
import { useOutsidePress } from './outside'

let mounted: { unmount: () => void } | null = null
const elements: HTMLElement[] = []

afterEach(() => {
  mounted?.unmount()
  mounted = null
  elements.splice(0).forEach((each) => each.remove())
})

const createElement = (): HTMLElement => {
  const element = document.createElement('div')
  document.body.appendChild(element)
  elements.push(element)
  return element
}

const setup = (inside: HTMLElement | null, isActive?: () => boolean) => {
  const onOutside = vi.fn<() => void>()
  const element = shallowRef<HTMLElement | null>(inside)
  const host = defineComponent({
    setup() {
      useOutsidePress(element, onOutside, isActive)
      return () => null
    },
  })
  mounted = mount(host)
  return { onOutside, wrapper: mounted }
}

const press = (on: HTMLElement) => on.dispatchEvent(new Event('pointerdown', { bubbles: true }))

describe('a press outside', () => {
  it('is reported', () => {
    const { onOutside } = setup(createElement())
    press(createElement())
    expect(onOutside).toHaveBeenCalledTimes(1)
  })

  it('is reported where no element is drawn yet', () => {
    const { onOutside } = setup(null)
    press(createElement())
    expect(onOutside).toHaveBeenCalledTimes(1)
  })

  it('is not reported on the element or inside it', () => {
    const inside = createElement()
    const child = document.createElement('span')
    inside.appendChild(child)
    const { onOutside } = setup(inside)
    press(inside)
    press(child)
    expect(onOutside).not.toHaveBeenCalled()
  })

  it('is not reported while the element is inactive', () => {
    const { onOutside } = setup(createElement(), () => false)
    press(createElement())
    expect(onOutside).not.toHaveBeenCalled()
  })

  it('is not heard once the component is gone', () => {
    const { onOutside, wrapper } = setup(createElement())
    wrapper.unmount()
    press(createElement())
    expect(onOutside).not.toHaveBeenCalled()
  })
})
