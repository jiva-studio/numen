import { describe, expect, it } from 'vitest'

import { Rating } from '../../../api/types.js'

import { GRADE_LABELS, grades, RATING_BY_GRADE } from './grades'

describe('the four answers', () => {
  it('are requested in the order they get harder to say', () => {
    expect(grades).toEqual(['again', 'hard', 'good', 'easy'])
  })

  // The recorded rating is the grade the user selected. Nothing downstream can
  // distinguish one rating from another, so a swapped pair here schedules a
  // forgotten card a week ahead.
  it('are written down as the four they name', () => {
    expect(RATING_BY_GRADE.again).toBe(Rating.AGAIN)
    expect(RATING_BY_GRADE.hard).toBe(Rating.HARD)
    expect(RATING_BY_GRADE.good).toBe(Rating.GOOD)
    expect(RATING_BY_GRADE.easy).toBe(Rating.EASY)
  })

  it('each have a word to press', () => {
    expect(GRADE_LABELS.again).toBe('Again')
    expect(GRADE_LABELS.hard).toBe('Hard')
    expect(GRADE_LABELS.good).toBe('Good')
    expect(GRADE_LABELS.easy).toBe('Easy')
  })
})
