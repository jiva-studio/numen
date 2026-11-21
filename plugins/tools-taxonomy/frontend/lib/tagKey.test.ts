import { describe, expect, it } from 'vitest'
import { getTagKey } from './tagKey'

describe('getTagKey', () => {
  it('ignores case, the spaces around a tag, a leading hash and the spaces around a segment', () => {
    expect(getTagKey(' #Projects / Web ')).toBe('projects/web')
  })

  it('joins the Unicode forms of one letter', () => {
    expect(getTagKey('Café')).toBe(getTagKey('Café'))
  })
})
