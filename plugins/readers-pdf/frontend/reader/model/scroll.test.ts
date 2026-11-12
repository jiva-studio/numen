import { describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import { usePanScroll } from './scroll'

interface Stage {
  readonly element: HTMLElement
  readonly scrollTo: ReturnType<typeof vi.fn<(options: ScrollToOptions) => void>>
}

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
  return { ...stage, area, scroll: usePanScroll(area) }
}

describe('a row sent somewhere', () => {
  it('is not dragging to begin with', () => {
    expect(createScroll().scroll.state.isDragging.value).toBe(false)
  })

  it('travels to the place when requested to', () => {
    const { scroll, scrollTo } = createScroll(0)

    scroll.moves.send(100, 'smooth')

    expect(scrollTo).toHaveBeenCalledWith({ left: 100, behavior: 'smooth' })
    expect(scroll.state.offset.value).toBe(0)
    expect(scroll.moves.getScrollOffset()).toBe(100)
  })

  it('is at the place at once when it does not travel', () => {
    const { scroll, scrollTo } = createScroll(0)

    scroll.moves.send(100, 'auto')

    expect(scrollTo).toHaveBeenCalledWith({ left: 100, behavior: 'auto' })
    expect(scroll.state.offset.value).toBe(100)
  })

  it('is sent at once when it is within a pixel of the place', () => {
    const { scroll, scrollTo } = createScroll(10)

    scroll.moves.send(11, 'smooth')
    scroll.moves.send(9, 'smooth')
    scroll.moves.send(10, 'smooth')

    expect(scrollTo).toHaveBeenCalledTimes(3)
    expect(scrollTo).toHaveBeenLastCalledWith({ left: 10, behavior: 'auto' })
    expect(scroll.moves.getScrollOffset()).toBe(10)
  })

  it('replaces a travel in flight when it is sent to the place it is at', () => {
    const { scroll, scrollTo, element } = createScroll(100)

    scroll.moves.send(500, 'smooth')
    element.scrollLeft = 100.5
    scroll.moves.send(100, 'smooth')

    expect(scrollTo).toHaveBeenCalledTimes(2)
    expect(scrollTo).toHaveBeenLastCalledWith({ left: 100, behavior: 'auto' })
    expect(scroll.moves.getScrollOffset()).toBe(100.5)
  })

  it('moves when the place is two pixels away', () => {
    const { scroll, scrollTo } = createScroll(10)

    scroll.moves.send(12, 'smooth')

    expect(scrollTo).toHaveBeenCalledTimes(1)
  })

  it('does nothing without an element', () => {
    const { scroll, area, scrollTo } = createScroll(0)
    area.value = null

    expect(() => scroll.moves.send(100, 'smooth')).not.toThrow()

    expect(scrollTo).not.toHaveBeenCalled()
    expect(scroll.moves.getScrollOffset()).toBe(0)
  })
})

describe('how far along the row is', () => {
  it('is where the element remains when nothing was sent', () => {
    expect(createScroll(50).scroll.moves.getScrollOffset()).toBe(50)
  })

  it('is the last known place when there is no element', () => {
    const { scroll, area } = createScroll(50)
    scroll.state.offset.value = 7
    area.value = null

    expect(scroll.moves.getScrollOffset()).toBe(7)
  })
})

describe('whether the row is standing still', () => {
  it('is not without an element', () => {
    const { scroll, area } = createScroll(0)
    area.value = null

    expect(scroll.moves.isStill()).toBe(false)
  })

  it('is, and reads where the element remains, when nothing was sent', () => {
    const { scroll } = createScroll(40)

    expect(scroll.moves.isStill()).toBe(true)
    expect(scroll.state.offset.value).toBe(40)
  })

  it('is not while the row is travelling to where it was sent', () => {
    const { scroll } = createScroll(0)
    scroll.moves.send(100, 'smooth')

    expect(scroll.moves.isStill()).toBe(false)
    expect(scroll.moves.getScrollOffset()).toBe(100)
  })

  it('is once the row is within a pixel of where it was sent, and then follows the element', () => {
    const { scroll, element } = createScroll(0)
    scroll.moves.send(100, 'smooth')

    element.scrollLeft = 99
    expect(scroll.moves.isStill()).toBe(true)

    element.scrollLeft = 30
    expect(scroll.moves.getScrollOffset()).toBe(30)
    expect(scroll.moves.isStill()).toBe(true)
  })

  it('is not two pixels short of where it was sent', () => {
    const { scroll, element } = createScroll(0)
    scroll.moves.send(100, 'smooth')

    element.scrollLeft = 98

    expect(scroll.moves.isStill()).toBe(false)
  })
})

interface Captured {
  readonly has: ReturnType<typeof vi.fn<(id: number) => boolean>>
  readonly set: ReturnType<typeof vi.fn<(id: number) => void>>
  readonly release: ReturnType<typeof vi.fn<(id: number) => void>>
}

const createCapture = (element: HTMLElement, holds = false): Captured => {
  const has = vi.fn<(id: number) => boolean>(() => holds)
  const activeSet = vi.fn<(id: number) => void>()
  const release = vi.fn<(id: number) => void>()
  element.hasPointerCapture = has
  element.setPointerCapture = activeSet
  element.releasePointerCapture = release
  return { has, set: activeSet, release }
}

const createPointer = (clientX: number, clientY: number, button = 0): PointerEvent =>
  ({ clientX, clientY, button, pointerId: 7 }) as unknown as PointerEvent

const createWheel = (deltaX: number, deltaY: number) => {
  const preventDefault = vi.fn<() => void>()
  const event = { deltaX, deltaY, preventDefault } as unknown as WheelEvent
  return { event, preventDefault }
}

const createPressedScroll = (scrollLeft = 500, scrollTop = 300) => {
  const subject = createScroll(scrollLeft)
  subject.element.scrollTop = scrollTop
  const capture = createCapture(subject.element)
  return { ...subject, capture }
}

describe('a hand dragging the row', () => {
  it('moves the row against the hand and takes the pointer', () => {
    const { scroll, element, capture } = createPressedScroll()
    scroll.moves.send(900, 'smooth')

    scroll.drag.onPointerDown(createPointer(200, 100))
    scroll.drag.onPointerMove(createPointer(150, 60))

    expect(element.scrollLeft).toBe(550)
    expect(element.scrollTop).toBe(340)
    expect(scroll.state.isDragging.value).toBe(true)
    expect(capture.set).toHaveBeenCalledWith(7)
    expect(scroll.moves.getScrollOffset()).toBe(550)
  })

  it('does not drag before the hand has travelled', () => {
    const { scroll, element, capture } = createPressedScroll()

    scroll.drag.onPointerDown(createPointer(200, 100))
    scroll.drag.onPointerMove(createPointer(201, 100))

    expect(element.scrollLeft).toBe(500)
    expect(scroll.state.isDragging.value).toBe(false)
    expect(capture.set).not.toHaveBeenCalled()
  })

  it('ignores a move when nothing has been pressed', () => {
    const { scroll, element } = createPressedScroll()

    scroll.drag.onPointerMove(createPointer(100, 100))

    expect(element.scrollLeft).toBe(500)
    expect(scroll.state.isDragging.value).toBe(false)
  })

  it('ignores a press with another button', () => {
    const { scroll, element } = createPressedScroll()

    scroll.drag.onPointerDown(createPointer(200, 100, 2))
    scroll.drag.onPointerMove(createPointer(100, 100))

    expect(element.scrollLeft).toBe(500)
  })

  it('ignores a press and a move without an element', () => {
    const { scroll, area } = createPressedScroll()
    area.value = null

    expect(() => {
      scroll.drag.onPointerDown(createPointer(200, 100))
      scroll.drag.onPointerMove(createPointer(100, 100))
    }).not.toThrow()
    expect(scroll.state.isDragging.value).toBe(false)
  })

  it('ignores a move after the element is removed', () => {
    const { scroll, area } = createPressedScroll()
    scroll.drag.onPointerDown(createPointer(200, 100))
    area.value = null

    expect(() => scroll.drag.onPointerMove(createPointer(100, 100))).not.toThrow()
    expect(scroll.state.isDragging.value).toBe(false)
  })

  it('does not capture a pointer the element already holds', () => {
    const { scroll, element } = createPressedScroll()
    const capture = createCapture(element, true)

    scroll.drag.onPointerDown(createPointer(200, 100))
    scroll.drag.onPointerMove(createPointer(100, 100))

    expect(capture.set).not.toHaveBeenCalled()
    expect(element.scrollLeft).toBe(600)
  })

  it('keeps dragging when the pointer cannot be taken', () => {
    const { scroll, element, capture } = createPressedScroll()
    capture.set.mockImplementation(() => {
      throw new Error('no such pointer')
    })

    scroll.drag.onPointerDown(createPointer(200, 100))

    expect(() => scroll.drag.onPointerMove(createPointer(100, 100))).not.toThrow()
    expect(element.scrollLeft).toBe(600)
    expect(scroll.state.isDragging.value).toBe(true)
  })

  it('releases the row and the pointer', () => {
    const { scroll, element, capture } = createPressedScroll()
    capture.has.mockReturnValue(true)
    scroll.drag.onPointerDown(createPointer(200, 100))
    scroll.drag.onPointerMove(createPointer(100, 100))

    scroll.drag.letGo(createPointer(100, 100))
    scroll.drag.onPointerMove(createPointer(0, 100))

    expect(scroll.state.isDragging.value).toBe(false)
    expect(capture.release).toHaveBeenCalledWith(7)
    expect(element.scrollLeft).toBe(600)
  })

  it('does not release a pointer it does not hold', () => {
    const { scroll, capture } = createPressedScroll()

    scroll.drag.letGo(createPointer(100, 100))

    expect(capture.release).not.toHaveBeenCalled()
  })

  it('releases without an element', () => {
    const { scroll, area } = createPressedScroll()
    area.value = null

    expect(() => scroll.drag.letGo(createPointer(100, 100))).not.toThrow()
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
    scroll.moves.send(900, 'smooth')

    scroll.onWheel(createWheel(10, 0).event)

    expect(scroll.moves.getScrollOffset()).toBe(510)
  })

  it('does nothing without an element', () => {
    const { scroll, area } = createScroll(0)
    area.value = null
    const { event, preventDefault } = createWheel(0, 120)

    expect(() => scroll.onWheel(event)).not.toThrow()
    expect(preventDefault).not.toHaveBeenCalled()
  })
})

describe('a row the browser scrolled back to its start', () => {
  it('is put back where it was last scrolled to', () => {
    const { scroll, element } = createScroll(0)
    element.scrollLeft = 480
    scroll.moves.isStill()
    element.scrollLeft = 0

    scroll.moves.restore()

    expect(element.scrollLeft).toBe(480)
  })

  it('is put at the page it was travelling to when the browser ended the travel', () => {
    const { scroll, element } = createScroll(0)
    scroll.moves.send(900, 'smooth')
    element.scrollLeft = 0

    scroll.moves.restore()

    expect(element.scrollLeft).toBe(900)
    expect(scroll.moves.isStill()).toBe(true)
  })

  it('is left alone while it jumps to a page', () => {
    const { scroll, element } = createScroll(0)

    scroll.moves.send(900, 'auto')
    scroll.moves.restore()

    expect(element.scrollLeft).toBe(0)
  })

  it('is left alone when it is where it was', () => {
    const { scroll, element } = createScroll(0)
    element.scrollLeft = 300
    scroll.moves.isStill()
    element.scrollLeft = 300.5

    scroll.moves.restore()

    expect(element.scrollLeft).toBe(300.5)
  })
})

describe('a row the browser moved without a scroll event', () => {
  it('is moved when it is not where it was last reported', () => {
    const { scroll, element } = createScroll(0)
    element.scrollLeft = 480
    scroll.moves.isStill()
    element.scrollLeft = 0

    expect(scroll.moves.isMoved()).toBe(true)
  })

  it('is moved when it is not where it was sent', () => {
    const { scroll, element } = createScroll(0)
    scroll.moves.send(900, 'smooth')
    element.scrollLeft = 0

    expect(scroll.moves.isMoved()).toBe(true)
  })

  it('is not moved where it was last reported', () => {
    const { scroll, element } = createScroll(0)
    element.scrollLeft = 480
    scroll.moves.isStill()
    element.scrollLeft = 480.5

    expect(scroll.moves.isMoved()).toBe(false)
  })
})
