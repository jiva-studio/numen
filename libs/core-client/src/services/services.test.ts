import { describe, expect, it } from 'vitest'
import { findServiceInfo, getServiceInfo } from './services'

describe('getServiceInfo', () => {
  it('returns the title and description of a known service', () => {
    expect(getServiceInfo('openai')).toEqual({
      title: 'OpenAI',
      description: 'The maker of ChatGPT and the GPT models.',
    })
  })

  it('returns the group as the title of an unknown service', () => {
    expect(getServiceInfo('harbour')).toEqual({ title: 'harbour', description: '' })
  })

  it('ignores the case of the group', () => {
    expect(getServiceInfo('OpenRouter').title).toBe('OpenRouter')
  })

  it('ignores the whitespace around the group', () => {
    expect(getServiceInfo('  anthropic\t').title).toBe('Anthropic')
  })

  it('does not match a name of the object prototype', () => {
    expect(getServiceInfo('constructor')).toEqual({ title: 'constructor', description: '' })
  })
})

describe('findServiceInfo', () => {
  it('returns the service of a known group, ignoring case and whitespace', () => {
    expect(findServiceInfo('  OpenAI ')?.title).toBe('OpenAI')
  })

  it('returns undefined for an unknown group and for a prototype name', () => {
    expect(findServiceInfo('harbour')).toBeUndefined()
    expect(findServiceInfo('constructor')).toBeUndefined()
  })
})
