import { describe, expect, it } from 'vitest'
import type { ConnectionFieldInfo, ConnectionInfo, ConnectionTypeInfo } from './types'
import {
  getCompleteRows,
  getConnectionUsability,
  getRowKey,
  moveRow,
  readRows,
  sortByUsability,
  writeRows,
  type RowDraft,
} from './rows'

const reasons = {
  unavailable: 'Type unavailable',
  incomplete: 'Incomplete',
  cannot: (operation: string) => `This type cannot ${operation}`,
}

const TITLES = { 'llm.embed': 'embed text', 'llm.profile': 'describe a model' }

const type = (overrides: Partial<ConnectionTypeInfo> = {}): ConnectionTypeInfo => ({
  id: 'a.b/x',
  pluginId: 'a.b',
  pluginName: 'A',
  title: 'X',
  description: '',
  operations: ['llm.complete'],
  fields: [],
  rowFields: [],
  isAvailable: true,
  ...overrides,
})

const connection = (overrides: Partial<ConnectionInfo> = {}): ConnectionInfo => ({
  id: 'c00000001',
  typeId: 'a.b/x',
  name: 'Work',
  revision: 1,
  fields: {},
  secretKeysSet: [],
  isTypeAvailable: true,
  isComplete: true,
  isDetected: false,
  path: '',
  uses: [],
  ...overrides,
})

const rowField = (key: string, isRequired: boolean): ConnectionFieldInfo => ({
  key,
  type: 'string',
  title: key,
  description: '',
  isRequired,
  isAddress: false,
  options: [],
  optionsFrom: `llm.${key}`,
})

const ROW_FIELDS = [rowField('model', true), rowField('reasoning', false)]

const draft = (
  key: string,
  connectionId = 'c00000001',
  options: Readonly<Record<string, string>> = { model: 'm' },
): RowDraft => ({ key, connection: connectionId, options })

describe('readRows', () => {
  it('reads the rows of the value in force in their order, each field under its key', () => {
    expect(
      readRows([
        { connection: 'c1', model: 'a', reasoning: 'low' },
        { connection: 'c2', model: 'b' },
        { connection: 'c3' },
      ]),
    ).toStrictEqual([
      { connection: 'c1', options: { model: 'a', reasoning: 'low' } },
      { connection: 'c2', options: { model: 'b' } },
      { connection: 'c3', options: {} },
    ])
  })

  it.each([
    [undefined],
    [null],
    ['text'],
    [{ connection: 'c1' }],
    [[1]],
    [[{ connection: 'c1', model: 2 }]],
  ])('reads no row from %j', (value) => {
    expect(readRows(value)).toStrictEqual([])
  })
})

describe('writeRows', () => {
  it('writes the connection and each field of a row under its key', () => {
    expect(
      writeRows([{ connection: 'c1', options: { model: 'a', reasoning: 'low' } }]),
    ).toStrictEqual([{ connection: 'c1', model: 'a', reasoning: 'low' }])
  })
})

describe('getRowKey', () => {
  it('is the same for the same values in another order of keys', () => {
    expect(getRowKey({ connection: 'c1', options: { model: 'a', reasoning: 'low' } })).toBe(
      getRowKey({ connection: 'c1', options: { reasoning: 'low', model: 'a' } }),
    )
    expect(getRowKey({ connection: 'c1', options: { model: 'a' } })).not.toBe(
      getRowKey({ connection: 'c1', options: { model: 'b' } }),
    )
  })
})

describe('getCompleteRows', () => {
  it('keeps the rows that name a connection and a value for each required field, in their order', () => {
    const rows = getCompleteRows(
      [
        draft('a'),
        draft('b', '', { model: 'm' }),
        draft('c', 'c00000002', { model: '' }),
        draft('d', 'c00000003', { model: 'n', reasoning: 'low' }),
      ],
      () => ROW_FIELDS,
    )

    expect(rows).toStrictEqual([
      { connection: 'c00000001', options: { model: 'm' } },
      { connection: 'c00000003', options: { model: 'n', reasoning: 'low' } },
    ])
  })

  it('keeps a row of a type with no required field once it names a connection', () => {
    expect(getCompleteRows([draft('a', 'c00000001', {})], () => [])).toStrictEqual([
      { connection: 'c00000001', options: {} },
    ])
  })
})

describe('moveRow', () => {
  it('moves a row to a place and shifts the rows between', () => {
    expect(moveRow(['a', 'b', 'c'], 0, 2)).toStrictEqual(['b', 'c', 'a'])
    expect(moveRow(['a', 'b', 'c'], 2, 0)).toStrictEqual(['c', 'a', 'b'])
  })

  it('keeps the rows as they were for a place outside the list or the same place', () => {
    expect(moveRow(['a', 'b'], 1, 1)).toStrictEqual(['a', 'b'])
    expect(moveRow(['a', 'b'], 0, 5)).toStrictEqual(['a', 'b'])
    expect(moveRow(['a', 'b'], -1, 0)).toStrictEqual(['a', 'b'])
  })
})

describe('getConnectionUsability', () => {
  it('finds a complete connection of a type that serves the operations usable', () => {
    expect(
      getConnectionUsability(
        connection(),
        type(),
        { ids: ['llm.complete'], titles: TITLES },
        reasons,
      ),
    ).toStrictEqual({ isUsable: true, reason: '' })
  })

  it('gives the reason a connection cannot be used', () => {
    expect(
      getConnectionUsability(
        connection(),
        type({ operations: ['llm.embed'] }),
        { ids: ['llm.embed', 'llm.profile'], titles: TITLES },
        reasons,
      ),
    ).toStrictEqual({
      isUsable: false,
      reason: 'This type cannot describe a model',
    })
    expect(
      getConnectionUsability(
        connection(),
        type(),
        { ids: ['llm.complete', 'llm.chat'], titles: TITLES },
        reasons,
      ),
    ).toStrictEqual({ isUsable: false, reason: 'This type cannot llm.chat' })
    expect(
      getConnectionUsability(
        connection({ isTypeAvailable: false }),
        type(),
        { ids: ['llm.complete'], titles: TITLES },
        reasons,
      ),
    ).toStrictEqual({ isUsable: false, reason: 'Type unavailable' })
    expect(
      getConnectionUsability(
        connection(),
        undefined,
        { ids: ['llm.complete'], titles: TITLES },
        reasons,
      ),
    ).toStrictEqual({
      isUsable: false,
      reason: 'Type unavailable',
    })
    expect(
      getConnectionUsability(
        connection({ isComplete: false }),
        type(),
        { ids: ['llm.complete'], titles: TITLES },
        reasons,
      ),
    ).toStrictEqual({ isUsable: false, reason: 'Incomplete' })
  })
})

describe('sortByUsability', () => {
  it('puts the unusable after the usable, each in the order they came', () => {
    const items = [
      { id: 'a', isUsable: false },
      { id: 'b', isUsable: true },
      { id: 'c', isUsable: false },
      { id: 'd', isUsable: true },
    ]

    expect(sortByUsability(items).map((one) => one.id)).toStrictEqual(['b', 'd', 'a', 'c'])
  })
})
