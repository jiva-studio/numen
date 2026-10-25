/**
 * The hand on a book: which gestures turn the page and which turn nothing.
 *
 * The browser's selection is stood in for, so a test says what is taken up and
 * where, and the reader is asked what it turned.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'

import type { PageNavigationAction } from '@/shared/lib/dom/pageNavigation'
import { createBookHand } from './hand'

const WIDTH = 1000

const setUp = (edge = 0) => {
  const area = document.createElement('div')
  Object.defineProperty(area, 'clientWidth', { value: WIDTH })
  const paper = document.createElement('div')
  paper.textContent = 'some words of the book'
  area.append(paper)
  document.body.append(area)

  const turn = vi.fn<(way: PageNavigationAction) => void>()
  const areaRef = shallowRef<HTMLElement | null>(area)
  const paperRef = shallowRef<HTMLElement | null>(paper)
  const hand = createBookHand({
    area: areaRef,
    paper: paperRef,
    turn,
    getEdge: () => edge,
  })
  return { area, areaRef, paper, paperRef, turn, hand }
}

const pointer = (clientX: number, target: unknown = null, button = 0): PointerEvent =>
  ({ button, clientX, target }) as unknown as PointerEvent

/** A selection standing where the test says. */
const select = (selection: {
  isCollapsed: boolean
  text: string
  anchorNode: Node | null
}): void => {
  vi.spyOn(window, 'getSelection').mockReturnValue({
    isCollapsed: selection.isCollapsed,
    anchorNode: selection.anchorNode,
    toString: () => selection.text,
  } as unknown as Selection)
}

afterEach(() => {
  vi.restoreAllMocks()
  document.body.replaceChildren()
})

describe('a swipe across the reading area', () => {
  it('turns the page the way the hand went', () => {
    const { hand, turn } = setUp()
    vi.spyOn(window, 'getSelection').mockReturnValue(null)

    hand.takeDown(pointer(600))
    hand.letGo(pointer(400))

    expect(turn).toHaveBeenCalledExactlyOnceWith('next')
  })

  it('is measured from the near edge on both ends of the gesture', () => {
    const { hand, turn } = setUp(100)
    vi.spyOn(window, 'getSelection').mockReturnValue(null)

    hand.takeDown(pointer(500))
    hand.letGo(pointer(480))

    expect(turn).not.toHaveBeenCalled()
  })
})

describe('a press near an edge', () => {
  it('is measured from the near edge of the reading area', () => {
    const { hand, turn } = setUp(100)
    vi.spyOn(window, 'getSelection').mockReturnValue(null)

    hand.takeDown(pointer(150))
    hand.letGo(pointer(150))

    expect(turn).toHaveBeenCalledExactlyOnceWith('back')
  })

  it('turns nothing where the hand went down with another button', () => {
    const { hand, turn } = setUp()
    vi.spyOn(window, 'getSelection').mockReturnValue(null)

    hand.takeDown(pointer(10, null, 2))
    hand.letGo(pointer(10))

    expect(turn).not.toHaveBeenCalled()
  })

  it('turns nothing where no hand went down', () => {
    const { hand, turn } = setUp()
    vi.spyOn(window, 'getSelection').mockReturnValue(null)

    hand.letGo(pointer(10))

    expect(turn).not.toHaveBeenCalled()
  })

  it('turns nothing where the reading area is gone', () => {
    const { hand, areaRef, turn } = setUp()
    vi.spyOn(window, 'getSelection').mockReturnValue(null)

    hand.takeDown(pointer(10))
    areaRef.value = null
    hand.letGo(pointer(10))

    expect(turn).not.toHaveBeenCalled()
  })

  it('turns nothing where the hand lifted on a link', () => {
    const { hand, paper, turn } = setUp()
    vi.spyOn(window, 'getSelection').mockReturnValue(null)
    paper.innerHTML = '<a href="x.xhtml"><em>link</em></a>'

    hand.takeDown(pointer(10))
    hand.letGo(pointer(10, paper.querySelector('em')))

    expect(turn).not.toHaveBeenCalled()
  })

  it('turns where the hand lifted on something that is no element', () => {
    const { hand, turn } = setUp()
    vi.spyOn(window, 'getSelection').mockReturnValue(null)

    hand.takeDown(pointer(10))
    hand.letGo(pointer(10, {}))

    expect(turn).toHaveBeenCalledExactlyOnceWith('back')
  })

  it('turns where the hand lifted on nothing', () => {
    const { hand, turn } = setUp()
    vi.spyOn(window, 'getSelection').mockReturnValue(null)

    hand.takeDown(pointer(990))
    hand.letGo(pointer(990, null))

    expect(turn).toHaveBeenCalledExactlyOnceWith('next')
  })

  it('turns where the hand lifted on an element that is no link', () => {
    const { hand, paper, turn } = setUp()
    vi.spyOn(window, 'getSelection').mockReturnValue(null)

    hand.takeDown(pointer(10))
    hand.letGo(pointer(10, paper))

    expect(turn).toHaveBeenCalledExactlyOnceWith('back')
  })
})

describe('words taken up in the text', () => {
  it('stop a swipe from turning', () => {
    const { hand, paper, turn } = setUp()
    select({ isCollapsed: false, text: 'words', anchorNode: paper.firstChild })

    hand.takeDown(pointer(600))
    hand.letGo(pointer(300))

    expect(turn).not.toHaveBeenCalled()
  })

  it('do not stop a turn where the selection is collapsed', () => {
    const { hand, paper, turn } = setUp()
    select({ isCollapsed: true, text: 'words', anchorNode: paper.firstChild })

    hand.takeDown(pointer(600))
    hand.letGo(pointer(300))

    expect(turn).toHaveBeenCalledExactlyOnceWith('next')
  })

  it('do not stop a turn where only blank space is taken', () => {
    const { hand, paper, turn } = setUp()
    select({ isCollapsed: false, text: ' \n ', anchorNode: paper.firstChild })

    hand.takeDown(pointer(600))
    hand.letGo(pointer(300))

    expect(turn).toHaveBeenCalledExactlyOnceWith('next')
  })

  it('do not stop a turn where they stand outside the text', () => {
    const { hand, area, turn } = setUp()
    const outside = document.createElement('p')
    outside.textContent = 'elsewhere'
    area.parentElement?.append(outside)
    select({ isCollapsed: false, text: 'elsewhere', anchorNode: outside.firstChild })

    hand.takeDown(pointer(600))
    hand.letGo(pointer(300))

    expect(turn).toHaveBeenCalledExactlyOnceWith('next')
  })

  it('do not stop a turn where no text is drawn', () => {
    const { hand, paper, paperRef, turn } = setUp()
    select({ isCollapsed: false, text: 'words', anchorNode: paper.firstChild })
    paperRef.value = null

    hand.takeDown(pointer(600))
    hand.letGo(pointer(300))

    expect(turn).toHaveBeenCalledExactlyOnceWith('next')
  })

  it('do not stop a turn where the selection has no anchor', () => {
    const { hand, turn } = setUp()
    select({ isCollapsed: false, text: 'words', anchorNode: null })

    hand.takeDown(pointer(600))
    hand.letGo(pointer(300))

    expect(turn).toHaveBeenCalledExactlyOnceWith('next')
  })
})
