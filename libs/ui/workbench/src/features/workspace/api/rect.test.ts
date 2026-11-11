import { describe, expect, it } from 'vitest'
import { getRect } from './rect'

describe('getRect', () => {
  it('reads what an element takes up on screen', () => {
    const element = {
      getBoundingClientRect: () => ({ left: 10, top: 20, width: 30, height: 40 }),
    } as unknown as Element

    expect(getRect(element)).toStrictEqual({ x: 10, y: 20, width: 30, height: 40 })
  })
})
