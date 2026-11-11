import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { useSecretValueEntry } from './useSecretValueEntry'

describe('useSecretValueEntry', () => {
  it('emits the trimmed value and clears the field without a submit function', () => {
    const sent: string[] = []
    const field = useSecretValueEntry({ getSubmit: () => undefined, emitCommit: (v) => sent.push(v) })
    field.entry.value = '  tok-123  '
    field.actions.commit()
    expect(sent).toEqual(['tok-123'])
    expect(field.entry.value).toBe('')
  })

  it('ignores a blank value', () => {
    const sent: string[] = []
    const field = useSecretValueEntry({ getSubmit: () => undefined, emitCommit: (v) => sent.push(v) })
    field.entry.value = '   '
    field.actions.commit()
    expect(sent).toEqual([])
  })

  it('keeps a refused value, marks it failed and sends it again only when asked', async () => {
    const calls: string[] = []
    const field = useSecretValueEntry({
      getSubmit: () => (value) => {
        calls.push(value)
        return Promise.resolve(calls.length > 1)
      },
      emitCommit: () => undefined,
    })
    field.entry.value = 'tok-123'
    field.actions.commit()
    await flushPromises()
    expect(field.hasFailed.value).toBe(true)
    expect(field.entry.value).toBe('tok-123')

    field.actions.commit()
    await flushPromises()
    expect(calls).toEqual(['tok-123'])

    field.actions.commitAgain()
    await flushPromises()
    expect(calls).toEqual(['tok-123', 'tok-123'])
    expect(field.hasFailed.value).toBe(false)
    expect(field.entry.value).toBe('')
  })

  it('sends no second value while one is in flight', async () => {
    let release: (accepted: boolean) => void = () => undefined
    const calls: string[] = []
    const field = useSecretValueEntry({
      getSubmit: () => (value) => {
        calls.push(value)
        return new Promise<boolean>((resolve) => {
          release = resolve
        })
      },
      emitCommit: () => undefined,
    })
    field.entry.value = 'tok-123'
    field.actions.commit()
    field.actions.commit()
    expect(calls).toEqual(['tok-123'])
    release(true)
    await flushPromises()
    expect(field.entry.value).toBe('')
  })
})
