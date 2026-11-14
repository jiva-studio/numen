import { describe, expect, it } from 'vitest'
import type { ConnectionFieldInfo, ConnectionInfo, ConnectionTypeInfo } from './types'
import {
  createFieldValues,
  createInitialValues,
  getAddressField,
  getEditableFields,
  getMissingFields,
  getSentFields,
  getSentSecrets,
  getStoredSecretKeys,
  isStoredKeyLost,
} from './form'

const field = (
  overrides: Partial<ConnectionFieldInfo> & Pick<ConnectionFieldInfo, 'key'>,
): ConnectionFieldInfo => ({
  type: 'string',
  title: overrides.key,
  description: '',
  isRequired: false,
  isAddress: false,
  options: [],
  ...overrides,
})

const type = (fields: readonly ConnectionFieldInfo[]): ConnectionTypeInfo => ({
  id: 'a.b/x',
  pluginId: 'a.b',
  pluginName: 'A',
  title: 'X',
  description: '',
  operations: ['llm.complete'],
  fields,
  rowFields: [],
  isAvailable: true,
})

const openRouter = type([
  field({ key: 'baseUrl', isAddress: true, fixedValue: 'https://openrouter.ai/api/v1' }),
  field({ key: 'apiKey', type: 'secret', isRequired: true }),
  field({ key: 'parallelRequests', type: 'number', defaultValue: 4, min: 1, max: 32, step: 1 }),
])

const compatible = type([
  field({ key: 'baseUrl', isAddress: true, isRequired: true }),
  field({ key: 'apiKey', type: 'secret' }),
  field({ key: 'stream', type: 'boolean' }),
  field({
    key: 'mode',
    type: 'enum',
    defaultValue: 'fast',
    options: [{ value: 'fast', title: 'Fast' }],
  }),
])

describe('getEditableFields', () => {
  it('leaves out a field with a fixed value and keeps the order of the type', () => {
    expect(getEditableFields(openRouter).map((one) => one.key)).toStrictEqual([
      'apiKey',
      'parallelRequests',
    ])
  })
})

describe('createFieldValues', () => {
  it('holds the default of each plain field and leaves secrets and fields with no default out', () => {
    expect(createFieldValues(openRouter)).toStrictEqual({ parallelRequests: 4 })
    expect(createFieldValues(compatible)).toStrictEqual({ mode: 'fast' })
  })
})

describe('getAddressField', () => {
  it('returns the field the person fills as the address, and none for a fixed one', () => {
    expect(getAddressField(compatible)?.key).toBe('baseUrl')
    expect(getAddressField(openRouter)).toBeUndefined()
    expect(getAddressField(type([]))).toBeUndefined()
  })
})

describe('getMissingFields', () => {
  it('names the required fields that have no value, secrets included', () => {
    expect(getMissingFields(compatible, {}, {})).toStrictEqual(['baseUrl'])
    expect(
      getMissingFields(compatible, { baseUrl: 'http://localhost:11434/v1' }, {}),
    ).toStrictEqual([])
    expect(getMissingFields(openRouter, {}, {})).toStrictEqual(['apiKey'])
    expect(getMissingFields(openRouter, {}, { apiKey: 'k' })).toStrictEqual([])
  })

  it('counts a secret that is already stored, and a blank text as no value', () => {
    expect(getMissingFields(openRouter, {}, {}, ['apiKey'])).toStrictEqual([])
    expect(getMissingFields(compatible, { baseUrl: '   ' }, {})).toStrictEqual(['baseUrl'])
  })
})

const stored = (overrides: Partial<ConnectionInfo> = {}): ConnectionInfo => ({
  id: 'c00000001',
  typeId: 'a.b/x',
  name: 'Server',
  revision: 1,
  fields: { baseUrl: 'https://a.example' },
  secretKeysSet: ['apiKey'],
  isTypeAvailable: true,
  isComplete: true,
  isDetected: false,
  path: '',
  uses: [],
  ...overrides,
})

describe('createInitialValues', () => {
  it('holds the defaults of the first available type for a new connection', () => {
    const unavailable = { ...compatible, isAvailable: false }
    expect(createInitialValues([unavailable, openRouter])).toStrictEqual({ parallelRequests: 4 })
    expect(createInitialValues([])).toStrictEqual({})
  })

  it('holds the stored values over the defaults of its type for an existing connection', () => {
    expect(createInitialValues([compatible], stored())).toStrictEqual({
      mode: 'fast',
      baseUrl: 'https://a.example',
    })
    expect(createInitialValues([], stored())).toStrictEqual({ baseUrl: 'https://a.example' })
  })
})

describe('getStoredSecretKeys', () => {
  it('lists the keys of the connection while the address is the stored one', () => {
    const draft = { baseUrl: 'https://a.example' }
    expect(getStoredSecretKeys(compatible, stored(), draft)).toStrictEqual(['apiKey'])
  })

  it('lists none once the address differs, and none for a new connection', () => {
    const draft = { baseUrl: 'https://b.example' }
    expect(getStoredSecretKeys(compatible, stored(), draft)).toStrictEqual([])
    expect(getStoredSecretKeys(compatible, undefined, {})).toStrictEqual([])
  })
})

describe('isStoredKeyLost', () => {
  it('is true when a stored key exists and the address was changed', () => {
    const changed = { baseUrl: 'https://b.example' }
    expect(isStoredKeyLost(compatible, stored(), changed)).toBe(true)
    expect(isStoredKeyLost(compatible, stored(), { baseUrl: 'https://a.example' })).toBe(false)
    expect(isStoredKeyLost(compatible, stored({ secretKeysSet: [] }), changed)).toBe(false)
    expect(isStoredKeyLost(compatible, undefined, changed)).toBe(false)
  })
})

describe('getSentFields', () => {
  it('sends the filled editable fields and leaves out blanks, fixed values and unknown keys', () => {
    const draft = { parallelRequests: 8, baseUrl: 'https://other.example', other: 'x' }
    expect(getSentFields(openRouter, draft)).toStrictEqual({ parallelRequests: 8 })
    expect(getSentFields(compatible, { baseUrl: '', stream: false })).toStrictEqual({
      stream: false,
    })
  })
})

describe('getSentSecrets', () => {
  it('sends the typed keys only', () => {
    expect(getSentSecrets({ apiKey: 'k', other: '' })).toStrictEqual({ apiKey: 'k' })
  })
})
