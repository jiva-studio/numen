import { describe, expect, it } from 'vitest'

import { Rating } from '@numen/protocol'

import { called, grades, rated } from './grades'

describe('the four answers', () => {
  it('are asked in the order they get harder to say', () => {
    expect(grades).toEqual(['again', 'hard', 'good', 'easy'])
  })

  // What is written down is what the person pressed. Nothing downstream can
  // tell one rating from another, so a pair crossed here is a card sent away
  // for a week because a person said they had forgotten it.
  it('are written down as the four they name', () => {
    expect(rated.again).toBe(Rating.AGAIN)
    expect(rated.hard).toBe(Rating.HARD)
    expect(rated.good).toBe(Rating.GOOD)
    expect(rated.easy).toBe(Rating.EASY)
  })

  it('each have a word to press', () => {
    expect(called.again).toBe('Again')
    expect(called.hard).toBe('Hard')
    expect(called.good).toBe('Good')
    expect(called.easy).toBe('Easy')
  })
})
