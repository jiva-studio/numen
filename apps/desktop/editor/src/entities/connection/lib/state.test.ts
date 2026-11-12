import { describe, expect, it } from 'vitest'
import type { ConnectionInfo, ConnectionTypeInfo } from './types'
import {
  getAddressText,
  getConnectionState,
  getOperationsText,
  getUsedByNames,
  hasAvailableType,
  hasUnavailableConnection,
} from './state'

const type = (overrides: Partial<ConnectionTypeInfo> = {}): ConnectionTypeInfo => ({
  id: 'a.b/x',
  pluginId: 'a.b',
  pluginName: 'A',
  title: 'X',
  description: '',
  operations: ['llm.complete', 'llm.embed'],
  fields: [
    {
      key: 'baseUrl',
      type: 'string',
      title: 'Address',
      description: '',
      isRequired: false,
      isAddress: true,
      options: [],
    },
  ],
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
  uses: [],
  ...overrides,
})

describe('getConnectionState', () => {
  it('is unavailable while no installed plugin provides the type, before it is incomplete', () => {
    expect(getConnectionState(connection({ isTypeAvailable: false, isComplete: false }))).toBe(
      'unavailable',
    )
  })

  it('is incomplete while a required field has no value', () => {
    expect(getConnectionState(connection({ isComplete: false }))).toBe('incomplete')
  })

  it('is ready otherwise', () => {
    expect(getConnectionState(connection())).toBe('ready')
  })
})

describe('getOperationsText', () => {
  it('joins the titles of the operations of the type, and is empty without a type', () => {
    expect(getOperationsText(type())).toBe('Completion · Embeddings')
    expect(getOperationsText(undefined)).toBe('')
  })

  it('leaves out the operations that only list the options of a row field', () => {
    const listed = type({
      operations: ['llm.chat', 'llm.models'],
      rowFields: [
        {
          key: 'model',
          type: 'string',
          title: 'Model',
          description: '',
          isRequired: true,
          isAddress: false,
          options: [],
          optionsFrom: 'llm.models',
        },
      ],
    })

    expect(getOperationsText(listed)).toBe('Chat')
  })
})

describe('getAddressText', () => {
  it('returns the address the person entered, and nothing for a fixed address or a missing type', () => {
    const entered = connection({ fields: { baseUrl: 'http://localhost:11434/v1' } })
    expect(getAddressText(type(), entered)).toBe('http://localhost:11434/v1')
    expect(getAddressText(type(), connection())).toBe('')
    expect(getAddressText(undefined, entered)).toBe('')
    expect(getAddressText(type({ fields: [] }), entered)).toBe('')
  })
})

describe('hasAvailableType', () => {
  it('is true when some type is provided by an installed plugin', () => {
    expect(hasAvailableType([type({ isAvailable: false }), type()])).toBe(true)
    expect(hasAvailableType([type({ isAvailable: false })])).toBe(false)
    expect(hasAvailableType([])).toBe(false)
  })
})

describe('hasUnavailableConnection', () => {
  it('is true when some connection has a type no plugin provides', () => {
    expect(hasUnavailableConnection([connection(), connection({ isTypeAvailable: false })])).toBe(
      true,
    )
    expect(hasUnavailableConnection([connection()])).toBe(false)
  })
})

describe('getUsedByNames', () => {
  const use = (pluginName: string, operation = 'llm.complete') => ({
    pluginId: `a.${pluginName.toLowerCase()}`,
    pluginName,
    operations: [operation],
  })

  it('names the plugins in the order of the uses', () => {
    expect(getUsedByNames([use('Proofreading')])).toBe('Proofreading')
    expect(getUsedByNames([use('Proofreading'), use('Embeddings', 'llm.embed')])).toBe(
      'Proofreading and Embeddings',
    )
    expect(getUsedByNames([use('A'), use('B'), use('C')])).toBe('A, B and C')
  })

  it('names a plugin once whatever the number of its settings', () => {
    expect(getUsedByNames([use('Proofreading'), use('Proofreading', 'llm.embed')])).toBe(
      'Proofreading',
    )
  })

  it('returns an empty text for a connection no setting uses', () => {
    expect(getUsedByNames([])).toBe('')
  })
})
