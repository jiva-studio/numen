import { describe, expect, it } from 'vitest'
import { useKeyFileChoice } from './useKeyFileChoice'

describe('useKeyFileChoice', () => {
  it('starts on the passphrase and waits for it', () => {
    const choice = useKeyFileChoice()

    expect(choice.mode.value).toBe('passphrase')
    expect(choice.isComplete.value).toBe(false)
  })

  it('is complete when the passphrase is repeated', () => {
    const choice = useKeyFileChoice()
    choice.passphrase.value = 'moon river'
    choice.confirmation.value = 'moon river'

    expect(choice.isComplete.value).toBe(true)
    expect(choice.isMismatch.value).toBe(false)
  })

  it('names a repeat that differs', () => {
    const choice = useKeyFileChoice()
    choice.passphrase.value = 'moon river'
    choice.confirmation.value = 'moon'

    expect(choice.isMismatch.value).toBe(true)
    expect(choice.isComplete.value).toBe(false)
  })

  it('needs no passphrase for the plain file', () => {
    const choice = useKeyFileChoice()
    choice.mode.value = 'plaintext'

    expect(choice.isComplete.value).toBe(true)
    expect(choice.isMismatch.value).toBe(false)
  })
})
