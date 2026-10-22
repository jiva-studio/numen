/**
 * The pointer, driven without a browser: the drawing is a stand-in with a
 * matrix, and the events are made here. What a drop evaluates to is
 * `lib/arrange/drop.test.ts`; this holds what the pointer does to the window and
 * the drawing.
 */
import { effectScope, type EffectScope } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getPosition, usePlexGesture } from './gesture'
import { arrangePlex, DEFAULT_OPTIONS, type Drop } from '../lib/arrange'
import { build } from '../fixtures/build'
import type { PlexRelation } from '../lib/relation'

const frame = arrangePlex(build('A node', { parent: 1, child: 2, jump: 1 }))
const RELATIONS_OFFERED: readonly PlexRelation[] = ['parent', 'child', 'jump']
const THRESHOLD = 10

interface Matrix {
  readonly a: number
  readonly b: number
  readonly c: number
  readonly d: number
  readonly e: number
  readonly f: number
}

const IDENTITY: Matrix = { a: 1, b: 0, c: 0, d: 1, e: 0, f: 0 }

/** A drawing that turns screen pixels into plex coordinates through `inverse`. */
function createDrawing(inverse: Matrix | null = IDENTITY) {
  let matrix = inverse
  return {
    setMatrix: (next: Matrix | null) => {
      matrix = next
    },
    setPointerCapture: vi.fn(),
    releasePointerCapture: vi.fn(),
    getScreenCTM: () => (matrix ? { inverse: () => matrix } : null),
  }
}

type Drawing = ReturnType<typeof createDrawing>

const asSvg = (drawing: object) => drawing as unknown as SVGSVGElement

/** The pointer going down on something inside a drawing. */
const createDown = (drawing: Drawing | null, x: number, y: number, pointerId = 1) =>
  ({
    target: drawing ? { ownerSVGElement: drawing } : null,
    clientX: x,
    clientY: y,
    pointerId,
  }) as unknown as PointerEvent

/** An event as the window gets it. */
const createPointerEvent = (type: string, x: number, y: number, pointerId = 1) => {
  const event = new MouseEvent(type, { clientX: x, clientY: y, bubbles: true })
  Object.defineProperty(event, 'pointerId', { value: pointerId })
  return event as PointerEvent
}

const send = (type: string, x = 0, y = 0, pointerId = 1) =>
  window.dispatchEvent(createPointerEvent(type, x, y, pointerId))

const scopes: EffectScope[] = []

function setUp(relations: readonly PlexRelation[] = RELATIONS_OFFERED) {
  const gestureDrawing = createDrawing()
  const commits: Drop[] = []
  const scope = effectScope()
  scopes.push(scope)
  const gesture = scope.run(() =>
    usePlexGesture({
      frame: () => frame,
      options: () => DEFAULT_OPTIONS,
      relations: () => relations,
      threshold: () => THRESHOLD,
      commit: (drop) => commits.push(drop),
    }),
  )!
  return { gesture, drawing: gestureDrawing, commits, scope }
}

afterEach(() => {
  // A gesture left under way would go on answering the next test's events.
  for (const scope of scopes.splice(0)) scope.stop()
  vi.restoreAllMocks()
})

describe('screen to plex coordinates', () => {
  const inverse: Matrix = { a: 2, b: 3, c: 5, d: 7, e: 11, f: 13 }

  it('goes through the whole matrix', () => {
    const gestureDrawing = createDrawing(inverse)

    expect(getPosition(asSvg(gestureDrawing), createPointerEvent('pointermove', 17, 19))).toEqual({
      x: 2 * 17 + 5 * 19 + 11,
      y: 3 * 17 + 7 * 19 + 13,
    })
  })

  it('is nowhere when the drawing has no matrix to give', () => {
    const gestureDrawing = createDrawing(null)

    expect(getPosition(asSvg(gestureDrawing), createPointerEvent('pointermove', 17, 19))).toBeNull()
  })

  it('is nowhere when the drawing cannot say where it is', () => {
    expect(getPosition(asSvg({}), createPointerEvent('pointermove', 17, 19))).toBeNull()
  })
})

describe('where a gesture can begin', () => {
  it('begins on something inside a drawing, and holds the pointer there', () => {
    const { gesture, drawing } = setUp()

    gesture.begin('focus', createDown(drawing, 3, 4, 7))

    expect(gesture.from.value).toBe('focus')
    expect(gesture.at.value).toEqual({ x: 3, y: 4 })
    expect(drawing.setPointerCapture).toHaveBeenCalledWith(7)
  })

  it('does not begin on something that is not in a drawing', () => {
    const { gesture } = setUp()

    expect(() => gesture.begin('focus', createDown(null, 3, 4))).not.toThrow()

    expect(gesture.from.value).toBeNull()
  })

  it('ignores target outside canvas bounds', () => {
    const { gesture } = setUp()
    const event = { target: { ownerSVGElement: null }, clientX: 0, clientY: 0, pointerId: 1 }

    expect(() => gesture.begin('focus', event as unknown as PointerEvent)).not.toThrow()

    expect(gesture.from.value).toBeNull()
  })

  it('does not begin where the drawing cannot say where the pointer is', () => {
    const add = vi.spyOn(window, 'addEventListener')
    const { gesture } = setUp()

    gesture.begin('focus', createDown(createDrawing(null), 3, 4))

    expect(gesture.from.value).toBeNull()
    expect(gesture.at.value).toBeNull()
    expect(add).not.toHaveBeenCalled()
  })

  it('goes on when the pointer cannot be retained', () => {
    const { gesture, drawing } = setUp()
    drawing.setPointerCapture.mockImplementation(() => {
      throw new Error('the pointer is not down')
    })

    gesture.begin('focus', createDown(drawing, 0, 0))
    send('pointerup', 0, 0)

    expect(gesture.from.value).toBeNull()
  })
})

describe('how far a gesture must go to be a drag', () => {
  it('is a press until it has gone the threshold, and asks for the commonest relation', () => {
    const { gesture, drawing } = setUp()

    gesture.begin('focus', createDown(drawing, 0, 0))
    send('pointermove', -6, -7.9)

    expect(gesture.outcome.value).toEqual({ kind: 'create', from: 'focus', relation: 'child' })
  })

  it('is a drag at exactly the threshold', () => {
    const { gesture, drawing } = setUp()

    gesture.begin('focus', createDown(drawing, 0, 0))
    send('pointermove', -6, -8)

    // Back on the node it began on, which is not somewhere to let go.
    expect(gesture.outcome.value).toBeNull()
  })

  it.each([
    ['along x', -100, 0, 3, 4],
    ['along y', 0, -100, 3, 4],
  ])('measures the distance gone, not where the pointer stands, %s', (_axis, x, y, dx, dy) => {
    const { gesture, drawing } = setUp()

    gesture.begin('focus', createDown(drawing, x, y))
    send('pointermove', x + dx, y + dy)

    expect(gesture.outcome.value).toEqual({ kind: 'create', from: 'focus', relation: 'child' })
  })

  it('is a drag in the direction it went once it has gone far enough', () => {
    const { gesture, drawing } = setUp()

    gesture.begin('focus', createDown(drawing, 0, 0))
    send('pointermove', 0, -300)

    expect(gesture.outcome.value).toEqual({ kind: 'create', from: 'focus', relation: 'parent' })
  })
})

describe('what a gesture evaluates to when nothing has begun', () => {
  it('is nothing', () => {
    const { gesture } = setUp()

    expect(gesture.outcome.value).toBeNull()
  })

  it('is nothing when the pointer is lost track of mid-gesture', () => {
    const { gesture, drawing } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0))

    drawing.setMatrix(null)
    send('pointermove', 5, 5)

    expect(gesture.at.value).toBeNull()
    expect(gesture.outcome.value).toBeNull()
  })
})

describe('one pointer at a time', () => {
  it('follows only the pointer that began it', () => {
    const { gesture, drawing } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0, 1))

    send('pointermove', 50, 60, 2)
    expect(gesture.at.value).toEqual({ x: 0, y: 0 })

    send('pointermove', 50, 60, 1)
    expect(gesture.at.value).toEqual({ x: 50, y: 60 })
  })

  it('is not release by another pointer coming up', () => {
    const { gesture, drawing, commits } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0, 1))

    send('pointerup', 0, 0, 2)

    expect(gesture.from.value).toBe('focus')
    expect(commits).toEqual([])
  })

  it('is not given up on by another pointer being taken away', () => {
    const { gesture, drawing } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0, 1))

    send('pointercancel', 0, 0, 2)

    expect(gesture.from.value).toBe('focus')
  })

  it('is given up on when the browser takes its own pointer away', () => {
    const { gesture, drawing, commits } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0, 1))

    send('pointercancel', 0, 0, 1)
    send('pointerup', 0, 0, 1)

    expect(gesture.from.value).toBeNull()
    expect(commits).toEqual([])
  })
})

describe('letting go', () => {
  it('commits what it came to and is over', () => {
    const { gesture, drawing, commits } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0))
    send('pointermove', 0, -300)

    send('pointerup', 0, -300)

    expect(commits).toEqual([{ kind: 'create', from: 'focus', relation: 'parent' }])
    expect(gesture.from.value).toBeNull()
    expect(gesture.at.value).toBeNull()
    expect(gesture.outcome.value).toBeNull()
  })

  it('lets go of the pointer it retained', () => {
    const { drawing, gesture } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0, 7))

    send('pointerup', 0, 0, 7)

    expect(drawing.releasePointerCapture).toHaveBeenCalledWith(7)
  })

  it('is over even when the pointer was never retained', () => {
    const { drawing, gesture } = setUp()
    drawing.releasePointerCapture.mockImplementation(() => {
      throw new Error('not held')
    })
    gesture.begin('focus', createDown(drawing, 0, 0))

    expect(() => send('pointerup', 0, 0)).not.toThrow()

    expect(gesture.from.value).toBeNull()
  })

  it('lets go of the pointer once, however many times it is given up on', () => {
    const { drawing, gesture } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0, 7))

    gesture.cancel()
    gesture.cancel()

    expect(drawing.releasePointerCapture).toHaveBeenCalledTimes(1)
  })

  it('gives up on a gesture that was never begun without a word', () => {
    const { gesture } = setUp()

    expect(() => gesture.cancel()).not.toThrow()
  })
})

describe('the window it listens on', () => {
  it('listens for the pointer moving, coming up, being taken away, and a key', () => {
    const add = vi.spyOn(window, 'addEventListener')
    const { gesture, drawing } = setUp()

    gesture.begin('focus', createDown(drawing, 0, 0))

    expect(add.mock.calls.map(([type]) => type).sort()).toEqual([
      'keydown',
      'pointercancel',
      'pointermove',
      'pointerup',
    ])
  })

  it('stops listening to all of them when it is over', () => {
    const add = vi.spyOn(window, 'addEventListener')
    const remove = vi.spyOn(window, 'removeEventListener')
    const { gesture, drawing } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0))

    gesture.cancel()

    expect(add).toHaveBeenCalledTimes(4)
    for (const [type, handler] of add.mock.calls) {
      expect(remove).toHaveBeenCalledWith(type, handler)
    }
  })

  it('is given up on by Escape and by no other key', () => {
    const { gesture, drawing } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0))

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'a' }))
    expect(gesture.from.value).toBe('focus')

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(gesture.from.value).toBeNull()
  })
})

describe('reaching out with no direction', () => {
  it('asks for the commonest relation at once', () => {
    const { gesture, commits } = setUp()

    gesture.ask('focus')

    expect(commits).toEqual([{ kind: 'create', from: 'focus', relation: 'child' }])
  })
})

describe('a plex that goes while a hand is on it', () => {
  it('lets go of everything', () => {
    const remove = vi.spyOn(window, 'removeEventListener')
    const { gesture, drawing, scope, commits } = setUp()
    gesture.begin('focus', createDown(drawing, 0, 0, 7))

    scope.stop()

    expect(gesture.from.value).toBeNull()
    expect(drawing.releasePointerCapture).toHaveBeenCalledWith(7)
    expect(remove).toHaveBeenCalledTimes(4)
    expect(commits).toEqual([])
  })
})

describe('reaching out where no relation is offered', () => {
  it('asks for nothing', () => {
    const { gesture, commits } = setUp([])

    gesture.ask('focus')

    expect(commits).toEqual([])
  })
})
