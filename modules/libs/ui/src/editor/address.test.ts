import { describe, expect, it } from 'vitest'
import { addressAt } from './address'
import { parsed } from './fixtures/state'

describe('the address under a position in the text', () => {
  const at = (doc: string, text: string) => addressAt(parsed(doc), doc.indexOf(text) + 1)

  it('is what a link points at', () => {
    expect(at('go [there](https://example.invalid)', 'there')).toBe('https://example.invalid')
  })

  it('is what a note in brackets points at', () => {
    expect(at('under [[Thermodynamics]] it sits', 'Thermo')).toBe('name://Thermodynamics')
  })

  it('is the identifier where the brackets hold one', () => {
    expect(at('under [[note://01J8]] it sits', 'note://')).toBe('note://01J8')
  })

  it('is nothing where the position stands in no link', () => {
    expect(at('under nothing at all', 'nothing')).toBeNull()
  })

  it('is nothing inside code, where a link is an example of one', () => {
    expect(at('```\n[[Thermodynamics]]\n```', 'Thermo')).toBeNull()
    expect(at('write `[[Thermodynamics]]` for it', 'Thermo')).toBeNull()
  })
})
