import { describe, expect, it } from 'vitest'
import { isPassphraseMismatch, isPassphraseSet } from './isPassphraseSet'

describe('isPassphraseSet', () => {
  it('needs the same text twice', () => {
    expect(isPassphraseSet('moon river', 'moon river')).toBe(true)
    expect(isPassphraseSet('moon river', 'moon rivers')).toBe(false)
  })

  it('refuses an empty passphrase', () => {
    expect(isPassphraseSet('', '')).toBe(false)
  })
})

describe('isPassphraseMismatch', () => {
  it('waits until the repeat has started', () => {
    expect(isPassphraseMismatch('moon river', '')).toBe(false)
    expect(isPassphraseMismatch('moon river', 'moon')).toBe(true)
    expect(isPassphraseMismatch('moon river', 'moon river')).toBe(false)
  })
})
