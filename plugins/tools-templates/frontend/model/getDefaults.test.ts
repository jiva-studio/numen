import { describe, expect, it } from 'vitest'
import { getDefaults } from './getDefaults'

describe('the defaults of the fields', () => {
  it('maps each label to its default', () => {
    expect(
      getDefaults([
        { kind: 'text', label: 'Client', options: [], default: 'Acme' },
        { kind: 'select', label: 'Kind', options: ['call', 'visit'], default: 'call' },
        { kind: 'text', label: 'Notes', options: [], default: '' },
      ]),
    ).toEqual({ Client: 'Acme', Kind: 'call', Notes: '' })
  })

  it('maps no fields to nothing', () => {
    expect(getDefaults([])).toEqual({})
  })
})
