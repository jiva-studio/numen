import { describe, expect, it } from 'vitest'
import { getFirstSecretName, getNextSecretName } from './serviceNames'

describe('the name of the first secret of a group', () => {
  it('is the title of the service of the group', () => {
    expect(getFirstSecretName('openrouter')).toBe('OpenRouter')
  })

  it('is Default for a group no service is known by', () => {
    expect(getFirstSecretName('Local Llama')).toBe('Default')
  })

  it('is Default for the empty group', () => {
    expect(getFirstSecretName('')).toBe('Default')
  })
})

describe('the next name of a secret', () => {
  it('is the name of the service in a group with no secret', () => {
    expect(getNextSecretName('openrouter', [])).toBe('OpenRouter')
    expect(getNextSecretName('Local Llama', [])).toBe('Default')
    expect(getNextSecretName('', [])).toBe('Default')
  })

  it('is the name of the service with V1 after the first secret', () => {
    expect(getNextSecretName('openrouter', ['OpenRouter'])).toBe('OpenRouter V1')
    expect(getNextSecretName('openrouter', ['Work', 'Home'])).toBe('OpenRouter V1')
  })

  it('is the first V number not in use', () => {
    expect(getNextSecretName('openrouter', ['OpenRouter', 'OpenRouter V1'])).toBe('OpenRouter V2')
    expect(getNextSecretName('openrouter', ['OpenRouter V1'])).toBe('OpenRouter V2')
    expect(getNextSecretName('openrouter', ['OpenRouter V2'])).toBe('OpenRouter V1')
  })
})
