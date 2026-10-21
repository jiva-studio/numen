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

  it('travels to the place when requested to', () => {
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
  it('is where the element remains when nothing was sent', () => {
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

  it('is, and reads where the element remains, when nothing was sent', () => {
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

interface Captured {
  readonly has: ReturnType<typeof vi.fn<(id: number) => boolean>>
  readonly set: ReturnType<typeof vi.fn<(id: number) => void>>
  readonly release: ReturnType<typeof vi.fn<(id: number) => void>>
}

/** Pointer capture on the element, which jsdom does not have. */
const createCapture = (element: HTMLElement, holds = false): Captured => {
  const has = vi.fn<(id: number) => boolean>(() => holds)
  const set = vi.fn<(id: number) => void>()
  const release = vi.fn<(id: number) => void>()
  element.hasPointerCapture = has
  element.setPointerCapture = set
  element.releasePointerCapture = release
  return { has, set, release }
}

const createPointer = (clientX: number, clientY: number, button = 0): PointerEvent =>
  ({ clientX, clientY, button, pointerId: 7 }) as unknown as PointerEvent

const createWheel = (deltaX: number, deltaY: number) => {
  const preventDefault = vi.fn<() => void>()
  const event = { deltaX, deltaY, preventDefault } as unknown as WheelEvent
  return { event, preventDefault }
}

const createPressed = (scrollLeft = 500, scrollTop = 300) => {
  const subject = createScroll(scrollLeft)
  subject.element.scrollTop = scrollTop
  const capture = createCapture(subject.element)
  return { ...subject, capture }
}

describe('a hand dragging the row', () => {
  it('moves the row against the hand and takes the pointer', () => {
    const { scroll, element, capture } = createPressed()
    scroll.send(900, 'smooth')

    scroll.onPointerDown(createPointer(200, 100))
    scroll.onPointerMove(createPointer(150, 60))

    expect(element.scrollLeft).toBe(550)
    expect(element.scrollTop).toBe(340)
    expect(scroll.isDragging.value).toBe(true)
    expect(capture.set).toHaveBeenCalledWith(7)
    expect(scroll.getScrollOffset()).toBe(550)
  })

  it('does not drag before the hand has travelled', () => {
    const { scroll, element, capture } = createPressed()

    scroll.onPointerDown(createPointer(200, 100))
    scroll.onPointerMove(createPointer(201, 100))

    expect(element.scrollLeft).toBe(500)
    expect(scroll.isDragging.value).toBe(false)
    expect(capture.set).not.toHaveBeenCalled()
  })

  it('ignores a move when nothing has been pressed', () => {
    const { scroll, element } = createPressed()

    scroll.onPointerMove(createPointer(100, 100))

    expect(element.scrollLeft).toBe(500)
    expect(scroll.isDragging.value).toBe(false)
  })

  it('ignores a press with another button', () => {
    const { scroll, element } = createPressed()

    scroll.onPointerDown(createPointer(200, 100, 2))
    scroll.onPointerMove(createPointer(100, 100))

    expect(element.scrollLeft).toBe(500)
  })

  it('ignores a press and a move without an element', () => {
    const { scroll, area } = createPressed()
    area.value = null

    expect(() => {
      scroll.onPointerDown(createPointer(200, 100))
      scroll.onPointerMove(createPointer(100, 100))
    }).not.toThrow()
    expect(scroll.isDragging.value).toBe(false)
  })

  it('ignores a move after the element is gone', () => {
    const { scroll, area } = createPressed()
    scroll.onPointerDown(createPointer(200, 100))
    area.value = null

    expect(() => scroll.onPointerMove(createPointer(100, 100))).not.toThrow()
    expect(scroll.isDragging.value).toBe(false)
  })

  it('does not take a pointer the element already holds', () => {
    const { scroll, element } = createPressed()
    const capture = createCapture(element, true)

    scroll.onPointerDown(createPointer(200, 100))
    scroll.onPointerMove(createPointer(100, 100))

    expect(capture.set).not.toHaveBeenCalled()
    expect(element.scrollLeft).toBe(600)
  })

  it('keeps dragging when the pointer cannot be taken', () => {
    const { scroll, element, capture } = createPressed()
    capture.set.mockImplementation(() => {
      throw new Error('no such pointer')
    })

    scroll.onPointerDown(createPointer(200, 100))

    expect(() => scroll.onPointerMove(createPointer(100, 100))).not.toThrow()
    expect(element.scrollLeft).toBe(600)
    expect(scroll.isDragging.value).toBe(true)
  })

  it('lets go of the row and the pointer', () => {
    const { scroll, element, capture } = createPressed()
    capture.has.mockReturnValue(true)
    scroll.onPointerDown(createPointer(200, 100))
    scroll.onPointerMove(createPointer(100, 100))

    scroll.letGo(createPointer(100, 100))
    scroll.onPointerMove(createPointer(0, 100))

    expect(scroll.isDragging.value).toBe(false)
    expect(capture.release).toHaveBeenCalledWith(7)
    expect(element.scrollLeft).toBe(600)
  })

  it('does not release a pointer it does not hold', () => {
    const { scroll, capture } = createPressed()

    scroll.letGo(createPointer(100, 100))

    expect(capture.release).not.toHaveBeenCalled()
  })

  it('lets go without an element', () => {
    const { scroll, area } = createPressed()
    area.value = null

    expect(() => scroll.letGo(createPointer(100, 100))).not.toThrow()
  })
})

describe('a wheel turned over the row', () => {
  const setTall = (element: HTMLElement, scrollHeight: number, clientHeight: number) => {
    Object.defineProperty(element, 'scrollHeight', { value: scrollHeight })
    Object.defineProperty(element, 'clientHeight', { value: clientHeight })
  }

  it('moves the row sideways where nothing is below', () => {
    const { scroll, element } = createScroll(500)
    setTall(element, 100, 100)
    const { event, preventDefault } = createWheel(0, 120)

    scroll.onWheel(event)

    expect(element.scrollLeft).toBe(620)
    expect(element.scrollTop).toBe(0)
    expect(preventDefault).toHaveBeenCalled()
  })

  it('moves the row down where there is something below', () => {
    const { scroll, element } = createScroll(500)
    setTall(element, 200, 100)
    const { event } = createWheel(0, 120)

    scroll.onWheel(event)

    expect(element.scrollLeft).toBe(500)
    expect(element.scrollTop).toBe(120)
  })

  it('moves along either axis alone', () => {
    const { scroll, element } = createScroll(500)
    setTall(element, 200, 100)

    scroll.onWheel(createWheel(40, 0).event)
    expect(element.scrollLeft).toBe(540)

    scroll.onWheel(createWheel(0, 30).event)
    expect(element.scrollTop).toBe(30)
  })

  it('leaves a wheel that returned nothing to the browser', () => {
    const { scroll, element } = createScroll(500)
    setTall(element, 200, 100)
    const { event, preventDefault } = createWheel(0, 0)

    scroll.onWheel(event)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(element.scrollLeft).toBe(500)
  })

  it('takes the row from wherever it was sent', () => {
    const { scroll, element } = createScroll(500)
    setTall(element, 100, 100)
    scroll.send(900, 'smooth')

    scroll.onWheel(createWheel(10, 0).event)

    expect(scroll.getScrollOffset()).toBe(510)
  })

  it('does nothing without an element', () => {
    const { scroll, area } = createScroll(0)
    area.value = null
    const { event, preventDefault } = createWheel(0, 120)

    expect(() => scroll.onWheel(event)).not.toThrow()
    expect(preventDefault).not.toHaveBeenCalled()
  })
})
