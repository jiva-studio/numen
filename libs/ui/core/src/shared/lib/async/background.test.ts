import { describe, expect, it } from 'vitest'
import { runInBackground } from './background'

describe('runInBackground', () => {
  it('passes the failure of the task to onError', async () => {
    const failures: unknown[] = []
    const failure = new Error('lost')
    runInBackground(Promise.reject(failure), (error) => failures.push(error))
    await Promise.resolve()
    await Promise.resolve()
    expect(failures).toEqual([failure])
  })

  it('does not call onError when the task succeeds', async () => {
    const failures: unknown[] = []
    runInBackground(Promise.resolve(1), (error) => failures.push(error))
    await Promise.resolve()
    await Promise.resolve()
    expect(failures).toEqual([])
  })
})
