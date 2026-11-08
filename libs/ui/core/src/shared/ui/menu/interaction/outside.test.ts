/**
 * What a press outside an element evaluates to.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, shallowRef } from 'vue'
import { isInside, useOutsidePress } from './outside'

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

  it('is not reported inside an open shadow root the element is in', () => {
    const host = createElement()
    const root = host.attachShadow({ mode: 'open' })
    const inside = document.createElement('div')
    const child = document.createElement('button')
    inside.appendChild(child)
    root.appendChild(inside)
    const { onOutside } = setup(inside)

    child.dispatchEvent(new Event('pointerdown', { bubbles: true, composed: true }))

    expect(onOutside).not.toHaveBeenCalled()
  })

  it('is reported on the host of a shadow root the element is in', () => {
    const host = createElement()
    const root = host.attachShadow({ mode: 'open' })
    const inside = document.createElement('div')
    root.appendChild(inside)
    const { onOutside } = setup(inside)

    press(host)

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

  it('is not heard once the component is removed', () => {
    const { onOutside, wrapper } = setup(createElement())
    wrapper.unmount()
    press(createElement())
    expect(onOutside).not.toHaveBeenCalled()
  })

  it('checks target node containment when composedPath is empty', () => {
    const inside = createElement()
    const child = document.createElement('span')
    inside.appendChild(child)
    const outside = createElement()
    expect(isInside(inside, { composedPath: () => [], target: child } as unknown as Event)).toBe(
      true,
    )
    expect(isInside(inside, { composedPath: () => [], target: outside } as unknown as Event)).toBe(
      false,
    )
    expect(isInside(inside, { composedPath: () => [], target: null } as unknown as Event)).toBe(
      false,
    )
  })
})
