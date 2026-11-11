/**
 * The reader's position in a book, and what turning a page evaluates to.
 *
 * The document is four spreads of one column, each holding one run. The style
 * of the rendered text is replaced by a stub, so a test can read what was
 * written to it and in what order.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed, shallowRef } from 'vue'

import type { Flow, Mark } from '../lib/spread'
import type { SettledBookProps } from '../lib/props'
import { getEasedProgress, getTurnTarget, useBookTurns } from './turns'
import type { LeavingSpread } from './leaving'

const FLOW: Flow = { contentWidth: 4000, width: 1000, gap: 0, columns: 1 }
const MARKS: readonly Mark[] = [
  { at: 0, x: 0 },
  { at: 100, x: 1000 },
  { at: 200, x: 2000 },
  { at: 300, x: 3000 },
]

const recordStyle = (element: HTMLElement, translate = ''): string[] => {
  const writes: string[] = []
  const style = {
    get translate() {
      return translate
    },
    set translate(value: string) {
      translate = value
      writes.push(`translate ${value}`)
    },
    set transition(value: string) {
      writes.push(`transition [${value}]`)
    },
  }
  Object.defineProperty(element, 'style', { value: style })
  return writes
}

const paperWithStyle = () => {
  const paper = document.createElement('div')
  const writes = recordStyle(paper)
  return { paper, writes }
}

const setUp = (marks: readonly Mark[] = MARKS, offset = 0, rendered = true, duration?: number) => {
  const { paper, writes } = paperWithStyle()
  const props = {
    book: { start: 0, end: 400 },
    span: { start: 0, end: 400 },
    at: offset,
  } as SettledBookProps
  const onMove = vi.fn<(pos: number) => void>()
  const turns = useBookTurns({
    paper: shallowRef<HTMLElement | null>(rendered ? paper : null),
    props,
    flow: computed(() => FLOW),
    spreadCount: computed(() => 4),
    marks: shallowRef(marks),
    onMove,
    ...(duration !== undefined ? { getDuration: () => duration } : {}),
  })
  return { turns, writes, onMove, paper }
}

describe('how much of the chapter is still to come', () => {
  it('is the columns after the spread in front', () => {
    const { turns } = setUp()
    expect(turns.position.leftInChapter.value).toBe(3)

    turns.moves.stand(1, 'auto')

    expect(turns.position.leftInChapter.value).toBe(2)
  })
})

describe('putting a spread in front', () => {
  it('keeps the spread within the document', () => {
    const { turns } = setUp()

    turns.moves.stand(10, 'auto')
    expect(turns.position.standing.value).toBe(3)

    turns.moves.stand(-2, 'auto')
    expect(turns.position.standing.value).toBe(0)
  })

  it('carries the columns to the spread at once, with no frames between', () => {
    const { turns, writes } = setUp()

    turns.moves.stand(2, 'auto')

    expect(writes).toStrictEqual(['translate -2000px 0'])
  })

  it('carries the columns at once where the motion lasts no time', () => {
    const { turns, writes } = setUp()

    turns.moves.stand(2, 'smooth')

    expect(writes).toStrictEqual(['translate -2000px 0'])
  })

  it('counts the spread without a drawn document', () => {
    const { turns, writes } = setUp(MARKS, 0, false)

    expect(() => turns.moves.stand(2, 'auto')).not.toThrow()
    expect(turns.position.standing.value).toBe(2)
    expect(writes).toStrictEqual([])
  })
})

describe('going to a spread', () => {
  it('carries the columns and says what remains in front', () => {
    const { turns, writes, onMove } = setUp()

    turns.moves.goTo(1)

    expect(writes).toStrictEqual(['translate -1000px 0'])
    expect(onMove).toHaveBeenCalledExactlyOnceWith(100)
  })

  it('says nothing where the offset in front is the one already retained', () => {
    const { turns, onMove } = setUp(MARKS, 100)

    turns.moves.goTo(1)

    expect(onMove).not.toHaveBeenCalled()
  })

  it('says nothing where no run remains in the spread', () => {
    const { turns, onMove } = setUp(MARKS.slice(0, 2))

    turns.moves.goTo(3)

    expect(onMove).not.toHaveBeenCalled()
  })
})

describe('the offset kept in front', () => {
  it('is that of the first run in the spread in front', () => {
    const { turns } = setUp(MARKS, 7)

    turns.moves.stand(2, 'auto')

    expect(turns.moves.getKeptOffset()).toBe(200)
  })

  it('is the offset retained where no run remains in the spread in front', () => {
    const { turns } = setUp(MARKS.slice(0, 2), 7)

    turns.moves.stand(3, 'auto')

    expect(turns.moves.getKeptOffset()).toBe(7)
  })

  it('is the offset passed to keepOffset while its spread is in front', () => {
    const { turns } = setUp()

    turns.moves.stand(2, 'auto')
    turns.moves.keepOffset(250)

    expect(turns.moves.getKeptOffset()).toBe(250)
  })

  it('is the first run of the spread in front once the kept offset is turned away from', () => {
    const { turns } = setUp()

    turns.moves.stand(2, 'auto')
    turns.moves.keepOffset(250)
    turns.moves.stand(0, 'auto')

    expect(turns.moves.getKeptOffset()).toBe(0)
  })
})

describe('a turn drawn frame by frame', () => {
  const MOTION_MS = 200

  /** Runs the frames of a clock that moves 20 ms a frame, and leaves each frame to its test. */
  const setUpFrames = () => {
    let now = 0
    const queue: FrameRequestCallback[] = []
    vi.spyOn(performance, 'now').mockImplementation(() => now)
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      queue.push(callback)
      return queue.length
    })
    vi.spyOn(window, 'cancelAnimationFrame').mockImplementation((id) => {
      queue[id - 1] = () => undefined
    })
    const frame = () => {
      now += 20
      queue.shift()?.(now)
    }
    const advance = (ms: number) => {
      now += ms
    }
    return { frame, advance }
  }

  const moved = (writes: readonly string[]): number[] =>
    writes.map((one) => Number.parseFloat(one.replace('translate ', '')))

  it('moves the columns toward the spread on every frame and ends exactly on it', () => {
    const { frame } = setUpFrames()
    const { turns, writes } = setUp(MARKS, 0, true, MOTION_MS)
    turns.moves.stand(0, 'auto')
    writes.length = 0

    turns.moves.stand(1, 'smooth')
    for (let index = 0; index < 20; index += 1) frame()

    const places = moved(writes)
    expect(places.length).toBeGreaterThan(5)
    expect(places.at(-1)).toBe(-1000)
    for (let index = 1; index < places.length; index += 1) {
      expect(places[index]).toBeLessThanOrEqual(places[index - 1] as number)
    }
    expect(writes.at(-1)).toBe('translate -1000px 0')
    vi.restoreAllMocks()
  })

  it('writes nothing after the last frame', () => {
    const { frame } = setUpFrames()
    const { turns, writes } = setUp(MARKS, 0, true, MOTION_MS)
    turns.moves.stand(0, 'auto')
    writes.length = 0

    turns.moves.stand(1, 'smooth')
    for (let index = 0; index < 20; index += 1) frame()
    const count = writes.length
    for (let index = 0; index < 5; index += 1) frame()

    expect(writes).toHaveLength(count)
    vi.restoreAllMocks()
  })

  it('goes on from where the columns stand when another turn is made, and never goes back', () => {
    const { frame } = setUpFrames()
    const { turns, writes } = setUp(MARKS, 0, true, MOTION_MS)
    turns.moves.stand(0, 'auto')
    writes.length = 0

    turns.moves.stand(1, 'smooth')
    frame()
    frame()
    turns.moves.stand(2, 'smooth')
    for (let index = 0; index < 20; index += 1) frame()

    const places = moved(writes)
    for (let index = 1; index < places.length; index += 1) {
      expect(places[index]).toBeLessThanOrEqual(places[index - 1] as number)
    }
    expect(places.at(-1)).toBe(-2000)
    vi.restoreAllMocks()
  })

  it('puts the columns where an unsmooth stand says, and ends a turn under way', () => {
    const { frame } = setUpFrames()
    const { turns, writes } = setUp(MARKS, 0, true, MOTION_MS)
    turns.moves.stand(0, 'auto')
    writes.length = 0

    turns.moves.stand(1, 'smooth')
    frame()
    turns.moves.stand(3, 'auto')
    const count = writes.length
    for (let index = 0; index < 20; index += 1) frame()

    expect(writes.at(-1)).toBe('translate -3000px 0')
    expect(writes).toHaveLength(count)
    vi.restoreAllMocks()
  })

  it('moves the columns across frames when the thread blocks before the first frame', () => {
    const { frame, advance } = setUpFrames()
    const { turns, writes } = setUp(MARKS, 0, true, MOTION_MS)
    turns.moves.stand(0, 'auto')
    writes.length = 0

    turns.moves.stand(1, 'smooth')
    advance(260)
    for (let index = 0; index < 20; index += 1) frame()

    expect(moved(writes).length).toBeGreaterThan(5)
    expect(writes.at(-1)).toBe('translate -1000px 0')
    vi.restoreAllMocks()
  })
})

describe('the curve of a turn', () => {
  it('starts at nothing, ends at the whole, and never goes past it', () => {
    expect(getEasedProgress(0)).toBeCloseTo(0, 5)
    expect(getEasedProgress(1)).toBeCloseTo(1, 5)
    let last = 0
    for (let time = 0; time <= 1; time += 0.01) {
      const progress = getEasedProgress(time)
      expect(progress).toBeGreaterThanOrEqual(last - 1e-9)
      expect(progress).toBeLessThanOrEqual(1 + 1e-9)
      last = progress
    }
  })

  it('goes most of the way early, as the ease-out curve of the motion tokens does', () => {
    expect(getEasedProgress(0.2)).toBeGreaterThan(0.6)
  })
})

describe('a spread entered from the adjacent document', () => {
  const createLeaving = (translate: string, target = Number.parseFloat(translate)) => {
    const holder = document.createElement('div')
    const copy = document.createElement('div')
    const copyWrites = recordStyle(copy, translate)
    holder.append(copy)
    const leaving: LeavingSpread = { copy, target, remove: vi.fn(() => copy.remove()) }
    return { holder, leaving, copyWrites }
  }

  it('starts one spread past the far edge and moves the copy out after a forward turn', () => {
    const { turns, writes } = setUp()
    const { holder, leaving, copyWrites } = createLeaving('-5000px 0')

    turns.moves.enterSpread(0, 'next', leaving)

    expect(turns.position.standing.value).toBe(0)
    expect(writes).toStrictEqual(['translate 1000px 0', 'translate 0px 0'])
    expect(copyWrites).toStrictEqual(['translate -5000px 0', 'translate -6000px 0'])
    expect(leaving.remove).toHaveBeenCalledOnce()
    expect(holder.children).toHaveLength(0)
  })

  it('starts one spread on the near side after a backward turn', () => {
    const { turns, writes } = setUp()
    const { leaving, copyWrites } = createLeaving('0px 0')

    turns.moves.enterSpread(3, 'back', leaving)

    expect(turns.position.standing.value).toBe(3)
    expect(writes).toStrictEqual(['translate -4000px 0', 'translate -3000px 0'])
    expect(copyWrites).toStrictEqual(['translate 0px 0', 'translate 1000px 0'])
  })

  it('carries a copy taken during a forward turn one spread past where that turn went', () => {
    const { turns, writes } = setUp()
    const { leaving, copyWrites } = createLeaving('-2600px 0', -3000)

    turns.moves.enterSpread(0, 'next', leaving)

    expect(writes).toStrictEqual(['translate 1400px 0', 'translate 0px 0'])
    expect(copyWrites).toStrictEqual(['translate -2600px 0', 'translate -4000px 0'])
  })

  it('carries a copy taken during a backward turn one spread past where that turn went', () => {
    const { turns, writes } = setUp()
    const { leaving, copyWrites } = createLeaving('-1400px 0', -1000)

    turns.moves.enterSpread(3, 'back', leaving)

    expect(writes).toStrictEqual(['translate -4400px 0', 'translate -3000px 0'])
    expect(copyWrites).toStrictEqual(['translate -1400px 0', 'translate 0px 0'])
  })

  it('removes the copy without a drawn document', () => {
    const { turns } = setUp(MARKS, 0, false)
    const { holder, leaving } = createLeaving('0px 0')

    turns.moves.enterSpread(1, 'next', leaving)

    expect(turns.position.standing.value).toBe(1)
    expect(leaving.remove).toHaveBeenCalledOnce()
    expect(holder.children).toHaveLength(0)
  })
})

describe('the place a turn goes to', () => {
  it('is the translate of the columns without a turn, or zero without one written', () => {
    const paper = document.createElement('div')
    expect(getTurnTarget(paper)).toBe(0)

    paper.style.translate = '-2000px 0'

    expect(getTurnTarget(paper)).toBe(-2000)
  })

  it('is the end of the turn under way, not the place its frames have reached', () => {
    let now = 0
    const queue: FrameRequestCallback[] = []
    vi.spyOn(performance, 'now').mockImplementation(() => now)
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      queue.push(callback)
      return queue.length
    })
    const { turns, paper, writes } = setUp(MARKS, 0, true, 200)
    turns.moves.stand(0, 'auto')

    turns.moves.stand(2, 'smooth')
    now += 20
    queue.shift()?.(now)

    expect(Number.parseFloat(writes.at(-1)?.replace('translate ', '') ?? '')).toBeGreaterThan(-2000)
    expect(getTurnTarget(paper)).toBe(-2000)
    vi.restoreAllMocks()
  })
})
