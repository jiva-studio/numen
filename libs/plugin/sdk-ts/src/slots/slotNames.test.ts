import { describe, expect, it } from 'vitest'
import { SLOT } from './slotNames'

describe('the slot names', () => {
  it('name the slots of the application only', () => {
    expect(Object.values(SLOT)).toContain('tabStrip.context')
    expect(Object.values(SLOT).filter((one) => /^(plexGraph|fileTree)\./.test(one))).toStrictEqual([])
  })
})
