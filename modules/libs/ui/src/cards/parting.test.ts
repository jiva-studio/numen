/**
 * What one face of a card comes to as it is read, as plain values. No DOM, no
 * measurement.
 */
import { describe, expect, it } from 'vitest'
import { parts } from './face'

describe('parts', () => {
  it('draws the front alone while the card is not turned', () => {
    expect(parts('f', 'b', false).map((part) => part.half)).toEqual(['front'])
  })

  it('draws the back under the front once the card is turned', () => {
    expect(parts('f', 'b', true).map((part) => part.half)).toEqual(['front', 'back'])
  })

  it('says a half holding nothing but space is blank', () => {
    expect(parts('  \n ', 'b', true).map((part) => part.blank)).toEqual([true, false])
  })

  it('hands the markdown on unchanged', () => {
    expect(parts('# Heading\n\n- one', '', false)[0]?.text).toBe('# Heading\n\n- one')
  })
})
