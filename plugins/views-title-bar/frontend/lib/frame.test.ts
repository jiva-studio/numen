import { describe, expect, it } from 'vitest'
import { isBarShown, readHideInFullscreenSetting } from './frame'

describe('readHideInFullscreenSetting', () => {
  it('turns on for true only', () => {
    expect(readHideInFullscreenSetting(true)).toBe(true)
    for (const value of [false, undefined, 'true', 1, null]) {
      expect(readHideInFullscreenSetting(value)).toBe(false)
    }
  })
})

describe('isBarShown', () => {
  it('hides the bar only when the setting is on and the window is in full screen', () => {
    expect(isBarShown(true, true)).toBe(false)
    expect(isBarShown(true, false)).toBe(true)
    expect(isBarShown(false, true)).toBe(true)
    expect(isBarShown(false, false)).toBe(true)
  })
})
