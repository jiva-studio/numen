import { describe, expect, it } from 'vitest'
import { ENV_MODEL_KEY, isModelKeySet } from './live.js'

describe('isModelKeySet', () => {
  it('is false when the key is missing or empty', () => {
    expect(isModelKeySet({})).toBe(false)
    expect(isModelKeySet({ [ENV_MODEL_KEY]: '' })).toBe(false)
  })

  it('is true when the key has a value', () => {
    expect(isModelKeySet({ [ENV_MODEL_KEY]: 'sk-invented' })).toBe(true)
  })
})
