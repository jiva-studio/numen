import { afterEach, describe, expect, it } from 'vitest'
import { Keys, MissingKeyError } from './keys.js'

const VARIABLE = 'NUMEN_ACCEPTANCE_TESTS_KEYS_TEST_VARIABLE'
const TABLE = [
  ['key name', 'environment variable'],
  ['Main', VARIABLE],
]

describe('Keys', () => {
  afterEach(() => {
    delete process.env[VARIABLE]
  })

  it('reads the value of the variable for a declared name', () => {
    process.env[VARIABLE] = 'invented-value'
    const keys = new Keys()
    keys.declare(TABLE)
    expect(keys.read('Main')).toBe('invented-value')
  })

  it('reads the text itself when no key has the name', () => {
    process.env[VARIABLE] = 'invented-value'
    const keys = new Keys()
    keys.declare(TABLE)
    expect(keys.read('other-text')).toBe('other-text')
  })

  it('reads the text itself when nothing is declared', () => {
    expect(new Keys().read('Main')).toBe('Main')
  })

  it('names the variable that is not set, and not a value', () => {
    expect(() => new Keys().declare(TABLE)).toThrow(VARIABLE)
  })

  it('separates a missing variable from a malformed table', () => {
    expect(() => new Keys().declare(TABLE)).toThrow(MissingKeyError)
    expect(() => new Keys().declare([['key name'], ['Main']])).not.toThrow(MissingKeyError)
  })

  it('names the variable that is empty', () => {
    process.env[VARIABLE] = ''
    expect(() => new Keys().declare(TABLE)).toThrow(VARIABLE)
  })

  it('asks for the two columns', () => {
    expect(() => new Keys().declare([['key name'], ['Main']])).toThrow('environment variable')
  })
})
