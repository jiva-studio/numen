import { describe, expect, it } from 'vitest'
import { SLOT } from './slotNames'

describe('the slot names', () => {
  it('name the end of a row of the file tree', () => {
    expect((SLOT as Record<string, string>)['fileTreeRowEnd']).toBe('fileTree.rowEnd')
  })
})
