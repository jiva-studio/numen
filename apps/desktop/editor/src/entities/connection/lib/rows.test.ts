import { describe, expect, it } from 'vitest'
import type { ConnectionInfo, ConnectionTypeInfo, ModelInfo } from './types'
import {
  getCompleteRows,
  getConnectionUsability,
  listModelsForOperation,
  moveRow,
  readRows,
  sortByUsability,
  type RowDraft,
} from './rows'

const reasons = {
  unavailable: 'Type unavailable',
  incomplete: 'Incomplete',
  cannot: (operation: string) => `This type cannot ${operation}`,
}

const type = (overrides: Partial<ConnectionTypeInfo> = {}): ConnectionTypeInfo => ({
  id: 'a.b/x',
  pluginId: 'a.b',
  pluginName: 'A',
  title: 'X',
  description: '',
  operations: ['complete'],
  fields: [],
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
  isPrivateNetworkAllowed: false,
  uses: [],
  ...overrides,
})

const draft = (key: string, connectionId = 'c00000001', model = 'm', reasoning = ''): RowDraft => ({
  key,
  connection: connectionId,
  model,
  reasoning,
})

describe('readRows', () => {
  it('reads the rows of the value in force in their order', () => {
    expect(
      readRows([
        { connection: 'c1', model: 'a', reasoning: 'low' },
        { connection: 'c2', model: 'b' },
      ]),
    ).toStrictEqual([
      { connection: 'c1', model: 'a', reasoning: 'low' },
      { connection: 'c2', model: 'b', reasoning: '' },
    ])
  })

  it.each([[undefined], [null], ['text'], [{ connection: 'c1' }], [[1]], [[{ connection: 'c1' }]]])(
    'reads no row from %j',
    (value) => {
      expect(readRows(value)).toStrictEqual([])
    },
  )
})

describe('getCompleteRows', () => {
  it('keeps the rows that name a connection and a model, in their order', () => {
    const rows = getCompleteRows([
      draft('a'),
      draft('b', '', 'm'),
      draft('c', 'c00000002', ''),
      draft('d', 'c00000003', 'n', 'low'),
    ])

    expect(rows).toStrictEqual([
      { connection: 'c00000001', model: 'm', reasoning: '' },
      { connection: 'c00000003', model: 'n', reasoning: 'low' },
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
  it('finds a complete connection of a type that does the operation usable', () => {
    expect(getConnectionUsability(connection(), type(), 'complete', reasons)).toStrictEqual({
      isUsable: true,
      reason: '',
    })
  })

  it('gives the reason a connection cannot be used', () => {
    expect(
      getConnectionUsability(connection(), type({ operations: ['complete'] }), 'embed', reasons),
    ).toStrictEqual({
      isUsable: false,
      reason: 'This type cannot embed',
    })
    expect(
      getConnectionUsability(connection({ isTypeAvailable: false }), type(), 'complete', reasons),
    ).toStrictEqual({ isUsable: false, reason: 'Type unavailable' })
    expect(getConnectionUsability(connection(), undefined, 'complete', reasons)).toStrictEqual({
      isUsable: false,
      reason: 'Type unavailable',
    })
    expect(
      getConnectionUsability(connection({ isComplete: false }), type(), 'complete', reasons),
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

describe('listModelsForOperation', () => {
  it('lists a model that the service labels for the operation or labels for none', () => {
    const models: readonly ModelInfo[] = [
      { id: 'a', title: 'A', operations: ['complete'] },
      { id: 'b', title: 'B', operations: ['embed'] },
      { id: 'c', title: 'C', operations: [] },
    ]

    expect(listModelsForOperation(models, 'complete').map((one) => one.id)).toStrictEqual([
      'a',
      'c',
    ])
    expect(listModelsForOperation(models, 'embed').map((one) => one.id)).toStrictEqual(['b', 'c'])
  })
})
