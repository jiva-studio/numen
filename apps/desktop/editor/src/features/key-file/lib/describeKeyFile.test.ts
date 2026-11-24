import { describe, expect, it } from 'vitest'
import type { KeyFileState } from '@/entities/key-file'
import { WORDS as words } from '../words'
import { describeKeyFile } from './describeKeyFile'

const state = (overrides: Partial<KeyFileState> = {}): KeyFileState =>
  Object.assign(
    { isInUse: true, mode: 'unset', isChoiceNeeded: false, isLocked: false, warning: '' } as KeyFileState,
    overrides,
  )

describe('describeKeyFile', () => {
  it('names the keychain when no file is in use', () => {
    expect(describeKeyFile(state({ isInUse: false, mode: 'passphrase' }))).toBe(words.rowKeychain)
  })

  it('names a plain file', () => {
    expect(describeKeyFile(state({ mode: 'plaintext' }))).toBe(words.rowPlaintext)
  })

  it('names a protected file, locked or open', () => {
    expect(describeKeyFile(state({ mode: 'passphrase' }))).toBe(words.rowPassphrase)
    expect(describeKeyFile(state({ mode: 'passphrase', isLocked: true }))).toBe(words.rowLocked)
  })

  it('names a file with no choice made', () => {
    expect(describeKeyFile(state())).toBe(words.rowUnset)
  })
})
