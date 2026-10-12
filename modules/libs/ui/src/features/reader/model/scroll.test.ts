import { describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import { useHandScroll } from './scroll'

interface Stage {
  readonly element: HTMLElement
  readonly scrollTo: ReturnType<typeof vi.fn<(options: ScrollToOptions) => void>>
}

/** A row element whose scroll offset is set by the test. */
const createStage = (scrollLeft: number): Stage => {
  const element = document.createElement('div')
  Object.defineProperty(element, 'scrollLeft', { value: scrollLeft, writable: true })
  Object.defineProperty(element, 'scrollTop', { value: 0, writable: true })
  const scrollTo = vi.fn<(options: ScrollToOptions) => void>()
  element.scrollTo = scrollTo as unknown as typeof element.scrollTo
  return { element, scrollTo }
}

const createScroll = (scrollLeft = 0) => {
  const stage = createStage(scrollLeft)
  const area = shallowRef<HTMLElement | null>(stage.element)
  return { ...stage, area, scroll: useHandScroll(area) }
}

describe('a row sent somewhere', () => {
  it('is not dragging to begin with', () => {
    expect(createScroll().scroll.isDragging.value).toBe(false)
  })

  it('travels to the place when asked to', () => {
    const { scroll, scrollTo } = createScroll(0)

    scroll.send(100, 'smooth')

    expect(scrollTo).toHaveBeenCalledWith({ left: 100, behavior: 'smooth' })
    expect(scroll.along.value).toBe(0)
    expect(scroll.getScrollOffset()).toBe(100)
  })

  it('is at the place at once when it does not travel', () => {
    const { scroll, scrollTo } = createScroll(0)

    scroll.send(100, 'auto')

    expect(scrollTo).toHaveBeenCalledWith({ left: 100, behavior: 'auto' })
    expect(scroll.along.value).toBe(100)
  })

  it('stays where it is when it is within a pixel of the place', () => {
    const { scroll, scrollTo } = createScroll(10)

    scroll.send(11, 'smooth')
    scroll.send(9, 'smooth')
    scroll.send(10, 'smooth')

    expect(scrollTo).not.toHaveBeenCalled()
    expect(scroll.getScrollOffset()).toBe(10)
  })

  it('moves when the place is two pixels away', () => {
    const { scroll, scrollTo } = createScroll(10)

    scroll.send(12, 'smooth')

    expect(scrollTo).toHaveBeenCalledTimes(1)
  })

  it('does nothing without an element', () => {
    const { scroll, area, scrollTo } = createScroll(0)
    area.value = null

    expect(() => scroll.send(100, 'smooth')).not.toThrow()

    expect(scrollTo).not.toHaveBeenCalled()
    expect(scroll.getScrollOffset()).toBe(0)
  })
})

describe('how far along the row is', () => {
  it('is where the element stands when nothing was sent', () => {
    expect(createScroll(50).scroll.getScrollOffset()).toBe(50)
  })

  it('is the last known place when there is no element', () => {
    const { scroll, area } = createScroll(50)
    scroll.along.value = 7
    area.value = null

    expect(scroll.getScrollOffset()).toBe(7)
  })
})

describe('whether the row is standing still', () => {
  it('is not without an element', () => {
    const { scroll, area } = createScroll(0)
    area.value = null

    expect(scroll.isStill()).toBe(false)
  })

  it('is, and reads where the element stands, when nothing was sent', () => {
    const { scroll } = createScroll(40)

    expect(scroll.isStill()).toBe(true)
    expect(scroll.along.value).toBe(40)
  })

  it('is not while the row is travelling to where it was sent', () => {
    const { scroll } = createScroll(0)
    scroll.send(100, 'smooth')

    expect(scroll.isStill()).toBe(false)
    expect(scroll.getScrollOffset()).toBe(100)
  })

  it('is once the row is within a pixel of where it was sent, and then follows the element', () => {
    const { scroll, element } = createScroll(0)
    scroll.send(100, 'smooth')

    element.scrollLeft = 99
    expect(scroll.isStill()).toBe(true)

    element.scrollLeft = 30
    expect(scroll.getScrollOffset()).toBe(30)
    expect(scroll.isStill()).toBe(true)
  })

  it('is not two pixels short of where it was sent', () => {
    const { scroll, element } = createScroll(0)
    scroll.send(100, 'smooth')

    element.scrollLeft = 98

    expect(scroll.isStill()).toBe(false)
  })
})
