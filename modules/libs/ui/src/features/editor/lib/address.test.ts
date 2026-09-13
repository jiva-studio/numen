import { describe, expect, it } from 'vitest'
import { addressAt } from './address'
import { createState } from '../fixtures/state'

describe('the address under a position in the text', () => {
  const getAddressIn = (doc: string, text: string) =>
    addressAt(createState(doc), doc.indexOf(text) + 1)

  it('is what a link points at', () => {
    expect(getAddressIn('go [there](https://example.invalid)', 'there')).toBe(
      'https://example.invalid',
    )
  })

  it('is what a note in brackets points at', () => {
    expect(getAddressIn('under [[Thermodynamics]] it sits', 'Thermo')).toBe(
      'name://Thermodynamics',
    )
  })

  it('is the identifier where the brackets hold one', () => {
    expect(getAddressIn('under [[note://01J8]] it sits', 'note://')).toBe('note://01J8')
  })

  it('is nothing where the position stands in no link', () => {
    expect(getAddressIn('under nothing at all', 'nothing')).toBeNull()
  })

  it('is nothing inside code, where a link is an example of one', () => {
    expect(getAddressIn('```\n[[Thermodynamics]]\n```', 'Thermo')).toBeNull()
    expect(getAddressIn('write `[[Thermodynamics]]` for it', 'Thermo')).toBeNull()
  })
})
