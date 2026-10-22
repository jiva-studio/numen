import { describe, expect, it } from 'vitest'
import { line, mark } from './styles'

describe('the decoration for a class', () => {
  it('is the same decoration each time a line is requested for it', () => {
    expect(line('cm-one')).toBe(line('cm-one'))
    expect(line('cm-one')).not.toBe(line('cm-two'))
  })

  it('is the same decoration each time a stretch of text is requested for it', () => {
    expect(mark('cm-one')).toBe(mark('cm-one'))
    expect(mark('cm-one')).not.toBe(mark('cm-two'))
  })

  it('is kept apart for a line and for a stretch of text', () => {
    expect(line('cm-one')).not.toBe(mark('cm-one'))
  })
})
