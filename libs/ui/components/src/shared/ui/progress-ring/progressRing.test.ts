import { describe, expect, it } from 'vitest'
import { getRingView } from './progressRing'

describe('the ring of a progress', () => {
  it('is a short arc where the progress is not known', () => {
    expect(getRingView(undefined)).toEqual({ isKnown: false, percent: 0, dash: '25 75' })
    expect(getRingView(Number.NaN).isKnown).toBe(false)
  })

  it('is filled to the progress where it is known', () => {
    expect(getRingView(0.295)).toMatchObject({ isKnown: true, percent: 30, dash: '30 100' })
    expect(getRingView(0.5).dash).toBe('50 100')
  })

  it('keeps a progress out of range within the ring', () => {
    expect(getRingView(-1)).toMatchObject({ percent: 0, dash: '0 100' })
    expect(getRingView(3)).toMatchObject({ percent: 100, dash: '100 100' })
  })
})
