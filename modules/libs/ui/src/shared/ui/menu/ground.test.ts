/**
 * What an open menu listens to on the window, and what it lets go of.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, shallowRef, type ShallowRef } from 'vue'
import { useMenuGround, type MenuGroundState } from './ground'

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

const setup = (menuElement: HTMLElement | null = null) => {
  const dismiss = vi.fn<() => void>()
  const menu: ShallowRef<HTMLElement | null> = shallowRef(menuElement)
  let ground: MenuGroundState | null = null
  const host = defineComponent({
    setup() {
      ground = useMenuGround(menu, dismiss)
      return () => null
    },
  })
  const wrapper = mount(host)
  mounted = wrapper
  return { dismiss, ground: ground as unknown as MenuGroundState, wrapper }
}

describe('what dismisses the menu', () => {
  it('is a press anywhere, even where no menu is drawn yet', () => {
    const { dismiss, ground } = setup()
    ground.listen()
    createElement().dispatchEvent(new Event('pointerdown', { bubbles: true }))
    expect(dismiss).toHaveBeenCalledTimes(1)
  })

  it('is a press the page itself would hold back', () => {
    const { dismiss, ground } = setup()
    ground.listen()
    const element = createElement()
    element.addEventListener('pointerdown', (event) => event.stopPropagation())
    element.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    expect(dismiss).toHaveBeenCalledTimes(1)
  })

  it('is a scroll of any element after input, which does not bubble', () => {
    const { dismiss, ground } = setup()
    ground.listen()
    window.dispatchEvent(new Event('wheel'))
    createElement().dispatchEvent(new Event('scroll'))
    expect(dismiss).toHaveBeenCalledTimes(1)
  })

  it('is not a scroll before any input, as opening the menu causes one', () => {
    const { dismiss, ground } = setup()
    ground.listen()
    createElement().dispatchEvent(new Event('scroll'))
    expect(dismiss).not.toHaveBeenCalled()
  })

  it('is not a scroll after input from an earlier opening', () => {
    const { dismiss, ground } = setup()
    ground.listen()
    window.dispatchEvent(new Event('wheel'))
    ground.release()
    ground.listen()
    createElement().dispatchEvent(new Event('scroll'))
    expect(dismiss).not.toHaveBeenCalled()
  })

  it.each(['wheel', 'touchstart', 'keydown'])('is a scroll after %s', (name) => {
    const { dismiss, ground } = setup()
    ground.listen()
    window.dispatchEvent(new Event(name))
    createElement().dispatchEvent(new Event('scroll'))
    expect(dismiss).toHaveBeenCalledTimes(1)
  })

  it('is a change of the window size', () => {
    const { dismiss, ground } = setup()
    ground.listen()
    window.dispatchEvent(new Event('resize'))
    expect(dismiss).toHaveBeenCalledTimes(1)
  })

  it('is Escape, which the page does not get to see', () => {
    const { dismiss, ground } = setup()
    ground.listen()
    const event = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true })
    window.dispatchEvent(event)
    expect(dismiss).toHaveBeenCalledTimes(1)
    expect(event.defaultPrevented).toBe(true)
  })

  it('is not any other key', () => {
    const { dismiss, ground } = setup()
    ground.listen()
    const event = new KeyboardEvent('keydown', { key: 'a', cancelable: true })
    window.dispatchEvent(event)
    expect(dismiss).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(false)
  })

  it('is not a press or a scroll inside the menu', () => {
    const menuElement = createElement()
    const inside = document.createElement('span')
    menuElement.appendChild(inside)
    const { dismiss, ground } = setup(menuElement)
    ground.listen()

    window.dispatchEvent(new Event('wheel'))
    inside.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    inside.dispatchEvent(new Event('scroll'))
    expect(dismiss).not.toHaveBeenCalled()
  })
})

describe('letting go of the window', () => {
  const fire = (): void => {
    window.dispatchEvent(new Event('wheel'))
    createElement().dispatchEvent(new Event('pointerdown', { bubbles: true }))
    createElement().dispatchEvent(new Event('scroll'))
    window.dispatchEvent(new Event('resize'))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
  }

  it('hears nothing once released', () => {
    const { dismiss, ground } = setup()
    ground.listen()
    ground.release()
    fire()
    expect(dismiss).not.toHaveBeenCalled()
  })

  it('hears nothing once the menu is unmounted while still open', () => {
    const { dismiss, ground, wrapper } = setup()
    ground.listen()
    wrapper.unmount()
    mounted = null
    fire()
    expect(dismiss).not.toHaveBeenCalled()
  })
})
