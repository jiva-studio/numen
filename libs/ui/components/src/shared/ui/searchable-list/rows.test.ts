/**
 * What a searchable list shows: the items a query keeps, the rows they make
 * with an action row, and where the group names go.
 */
import { describe, expect, it } from 'vitest'
import {
  FIRST_KEY,
  buildRows,
  filterItems,
  findExactItem,
  getHeadings,
  stepRow,
  type ListItem,
} from './rows'

const MODELS: readonly ListItem[] = [
  { id: 'a', text: 'Plum 4', group: 'Orchard' },
  { id: 'b', text: 'Apple mini', detail: 'hedge/apple-mini', group: 'Hedge' },
  { id: 'c', text: 'Pear', group: 'Orchard' },
]

const keys = (rows: ReturnType<typeof buildRows>) => rows.map((row) => row.key)

describe('which items a query keeps', () => {
  it('keeps every item where there are no words', () => {
    expect(filterItems(MODELS, '')).toStrictEqual(MODELS)
    expect(filterItems(MODELS, '   ')).toStrictEqual(MODELS)
  })

  it('finds a word in the text, in any case', () => {
    expect(filterItems(MODELS, 'PLUM').map((one) => one.id)).toStrictEqual(['a'])
  })

  it('finds a word in the detail and in the group', () => {
    expect(filterItems(MODELS, 'hedge').map((one) => one.id)).toStrictEqual(['b'])
    expect(filterItems(MODELS, 'orchard').map((one) => one.id)).toStrictEqual(['a', 'c'])
  })

  it('wants every word found', () => {
    expect(filterItems(MODELS, 'orchard pea').map((one) => one.id)).toStrictEqual(['c'])
    expect(filterItems(MODELS, 'orchard apple')).toStrictEqual([])
    expect(filterItems([{ id: 'x', text: 'ab', detail: 'cd' }], 'bc')).toStrictEqual([])
    expect(filterItems([{ id: 'x', text: 'ab' }], 'stryker')).toStrictEqual([])
  })
})

describe('the item whose text is the query', () => {
  it('ignores case and the space around the query', () => {
    expect(findExactItem(MODELS, ' PEAR ')?.id).toBe('c')
  })

  it('is none for a part of a text, and for nothing typed', () => {
    expect(findExactItem(MODELS, 'sonn')).toBeUndefined()
    expect(findExactItem(MODELS, '  ')).toBeUndefined()
    expect(findExactItem([{ id: 'empty', text: '' }], '  ')).toBeUndefined()
  })
})

describe('the rows of a list without an action row', () => {
  it('are the items, each under its identifier', () => {
    expect(keys(buildRows(MODELS, '', false))).toStrictEqual(['a', 'b', 'c'])
  })
})

describe('the rows of a list with an action row', () => {
  it('are none while nothing is typed', () => {
    expect(buildRows(MODELS, '  ', true)).toStrictEqual([])
  })

  it('put the action first and the items after it', () => {
    const rows = buildRows(MODELS, 'x', true)
    expect(keys(rows)).toStrictEqual([FIRST_KEY, 'a', 'b', 'c'])
    expect(rows[0]?.kind).toBe('action')
  })

  it('keep the action alone as the items have not arrived', () => {
    expect(keys(buildRows([], 'x', true))).toStrictEqual([FIRST_KEY])
  })

  it('give the first row to an item that is exactly the query, under the key the action had', () => {
    const rows = buildRows(MODELS, 'pear', true)
    expect(keys(rows)).toStrictEqual([FIRST_KEY, 'a', 'b'])
    expect(rows[0]).toMatchObject({ kind: 'item', item: { id: 'c' } })
  })

  it('do not repeat the exact item further down', () => {
    const rows = buildRows(MODELS, 'plum 4', true)
    expect(rows.filter((row) => row.kind === 'item' && row.item.id === 'a')).toHaveLength(1)
  })
})

describe('the group names over the rows', () => {
  it('stand where the group differs from the row before', () => {
    const rows = buildRows(MODELS, '', false)
    expect(getHeadings(rows)).toStrictEqual(['Orchard', 'Hedge', 'Orchard'])
  })

  it('are not drawn over items of one group again', () => {
    const rows = buildRows([MODELS[0]!, MODELS[2]!], '', false)
    expect(getHeadings(rows)).toStrictEqual(['Orchard', undefined])
  })

  it('are none for items that name no group, and none over the action row', () => {
    const rows = buildRows([{ id: 'n', text: 'Page' }], 'x', true)
    expect(getHeadings(rows)).toStrictEqual([undefined, undefined])
    const mixed = buildRows(
      [
        { id: 'a', text: 'A', group: 'G' },
        { id: 'b', text: 'B' },
      ],
      '',
      false,
    )
    expect(getHeadings(mixed)).toStrictEqual(['G', undefined])
  })
})

describe('the row a step lands on', () => {
  const THREE = buildRows(MODELS, '', false)

  it('wraps at both ends', () => {
    expect(stepRow(THREE, 2, 1)).toBe(0)
    expect(stepRow(THREE, 0, -1)).toBe(2)
  })

  it('goes down and up by one', () => {
    expect(stepRow(THREE, 0, 1)).toBe(1)
    expect(stepRow(THREE, 2, -1)).toBe(1)
  })

  it('passes over a disabled item', () => {
    const rows = buildRows(
      [MODELS[0]!, { ...MODELS[1]!, isDisabled: true }, MODELS[2]!],
      '',
      false,
    )
    expect(stepRow(rows, 0, 1)).toBe(2)
    expect(stepRow(rows, 2, -1)).toBe(0)
  })

  it('is none where there is no row', () => {
    expect(stepRow([], 0, 1)).toBe(-1)
  })
})
