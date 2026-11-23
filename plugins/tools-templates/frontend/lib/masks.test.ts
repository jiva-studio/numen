import { describe, expect, it } from 'vitest'
import { canHold, findGroup, parseMasks } from './masks'

interface Table {
  readonly parse: readonly { raw: string; masks: readonly string[] }[]
  readonly defaults: readonly string[]
  readonly match: readonly { mask: string; path: string; matches: boolean }[]
  readonly group: readonly { mask: string; path: string; group: string }[]
  readonly find: readonly { masks: string; path: string; group: string | null }[]
}

const fixtures = import.meta.glob('../../testdata/masks.json', {
  eager: true,
  import: 'default',
}) as Record<string, Table>

const table = fixtures['../../testdata/masks.json']
if (!table) throw new Error('no testdata/masks.json')

const texts = (raw: string) => parseMasks(raw).map((mask) => mask.segments.join('/'))

describe('the masks setting', () => {
  it('is read as the Go side reads it', () => {
    for (const c of table.parse) {
      expect(texts(c.raw), JSON.stringify(c.raw)).toEqual(c.masks.length ? c.masks : table.defaults)
    }
  })

  it('names the masks that match a path', () => {
    for (const c of table.match) {
      expect(findGroup(parseMasks(c.mask), c.path) !== undefined, `${c.mask} on ${c.path}`).toBe(
        c.matches,
      )
    }
  })

  it('groups a path by the folder below the static folder of the mask', () => {
    for (const c of table.group) {
      expect(findGroup(parseMasks(c.mask), c.path), `${c.mask} on ${c.path}`).toBe(c.group)
    }
  })

  it('takes the group of the first mask that matches', () => {
    for (const c of table.find) {
      expect(findGroup(parseMasks(c.masks), c.path), c.path).toBe(c.group ?? undefined)
    }
  })
})

describe('the folders that could hold a template', () => {
  const masks = parseMasks('Templates/**,Models/Work/*.md')

  it.each([
    ['', true],
    ['Templates', true],
    ['Templates/a/b', true],
    ['Models', true],
    ['Models/Work', true],
    ['Models/Work/sub', false],
    ['Notes', false],
    ['Models/Other', false],
  ])('reads %j as %j', (folder, expected) => {
    expect(canHold(masks, folder)).toBe(expected)
  })

  it('holds any folder under a leading **', () => {
    expect(canHold(parseMasks('**/*.template.md'), 'any/folder')).toBe(true)
  })
})
