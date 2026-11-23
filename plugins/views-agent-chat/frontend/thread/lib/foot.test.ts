import { describe, expect, it } from 'vitest'
import { isAtFoot, getFoot } from './foot'
import { SLACK } from './slack'

const area = (scrollTop: number, scrollHeight = 500, clientHeight = 100) => ({
  scrollTop,
  scrollHeight,
  clientHeight,
})

describe('where the foot is', () => {
  it('is as far down as what is in the area runs past what shows', () => {
    expect(getFoot(area(0))).toBe(400)
  })

  it('is the head, for an area with less in it than shows', () => {
    expect(getFoot(area(0, 40, 100))).toBe(0)
  })
})

describe('whether the foot is in view', () => {
  it('it is, at the foot', () => {
    expect(isAtFoot(area(400))).toBe(true)
  })

  it('it is, within the slack of it', () => {
    expect(isAtFoot(area(400 - SLACK))).toBe(true)
  })

  it('it is not, a hair further up', () => {
    expect(isAtFoot(area(400 - SLACK - 1))).toBe(false)
  })

  it('it is not, halfway up a long conversation', () => {
    expect(isAtFoot(area(200))).toBe(false)
  })

  it('it is, in an area with nothing to scroll', () => {
    expect(isAtFoot(area(0, 40, 100))).toBe(true)
  })
})
