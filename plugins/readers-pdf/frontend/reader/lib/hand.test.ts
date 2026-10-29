import { describe, expect, it } from 'vitest'
import { DRAG_THRESHOLD, Hand, getWheelOffset } from './hand'

describe('a wheel turned over the row', () => {
  it('moves the row sideways where there is nothing below', () => {
    // wheel turned down means the next page.
    expect(getWheelOffset({ x: 0, y: 120 }, false)).toEqual({ x: 120, y: 0 })
    expect(getWheelOffset({ x: 0, y: -120 }, false)).toEqual({ x: -120, y: 0 })
  })

  it('moves the row down where there is something below', () => {
    // When zoomed in, the view scrolls on both axes, and scrolling down moves
    // down as it does everywhere else in the window.
    expect(getWheelOffset({ x: 0, y: 120 }, true)).toEqual({ x: 0, y: 120 })
  })

  it('moves sideways when the wheel says sideways, either way', () => {
    expect(getWheelOffset({ x: 40, y: 0 }, true)).toEqual({ x: 40, y: 0 })
    expect(getWheelOffset({ x: 40, y: 0 }, false)).toEqual({ x: 40, y: 0 })
  })
})

describe('a hand on the row', () => {
  const createPoint = (x: number, y = 0) => ({ x, y })

  it('holds nothing until it has taken hold', () => {
    const hand = new Hand()

    expect(hand.holding).toBe(false)
    expect(hand.dragging).toBe(false)
    expect(hand.to(createPoint(100))).toBeUndefined()
  })

  it('holds the row once it has taken hold', () => {
    const hand = new Hand()
    hand.take(createPoint(200), createPoint(500))

    expect(hand.holding).toBe(true)
  })

  it('drags once the hand has travelled exactly the threshold', () => {
    const hand = new Hand()
    hand.take(createPoint(200), createPoint(500))

    expect(hand.to(createPoint(200 + DRAG_THRESHOLD))).toEqual({ x: 500 - DRAG_THRESHOLD, y: 0 })
    expect(hand.dragging).toBe(true)
  })

  it('does not drag until the hand has travelled', () => {
    // A press that never moves is a press. Dragging from the first pixel takes
    // the click off whatever was under it.
    const hand = new Hand()
    hand.take(createPoint(200), createPoint(500))

    expect(hand.to(createPoint(200 + DRAG_THRESHOLD - 1))).toBeUndefined()
    expect(hand.dragging).toBe(false)
  })

  it('moves the row against the hand', () => {
    // Dragging left brings the following pages into view, like pushing a book
    // aside on a table.
    const hand = new Hand()
    hand.take(createPoint(200), createPoint(500))

    expect(hand.to(createPoint(150))).toEqual({ x: 550, y: 0 })
    expect(hand.dragging).toBe(true)
  })

  it('keeps dragging once it has begun, however small the next move', () => {
    const hand = new Hand()
    hand.take(createPoint(200), createPoint(500))
    hand.to(createPoint(150))

    expect(hand.to(createPoint(149))).toEqual({ x: 551, y: 0 })
  })

  it('moves both ways at once', () => {
    const hand = new Hand()
    hand.take({ x: 200, y: 100 }, { x: 500, y: 300 })

    expect(hand.to({ x: 150, y: 60 })).toEqual({ x: 550, y: 340 })
  })

  it('lets go', () => {
    const hand = new Hand()
    hand.take(createPoint(200), createPoint(500))
    hand.to(createPoint(150))

    hand.release()

    expect(hand.holding).toBe(false)
    expect(hand.to(createPoint(100))).toBeUndefined()
  })

  it('takes hold again from wherever the row now remains', () => {
    const hand = new Hand()
    hand.take(createPoint(200), createPoint(500))
    hand.to(createPoint(150))
    hand.release()

    hand.take(createPoint(400), createPoint(550))

    expect(hand.to(createPoint(350))).toEqual({ x: 600, y: 0 })
  })
})
