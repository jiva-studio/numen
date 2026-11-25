/**
 * Runs the calls of the identity section one at a time: the section is busy while one runs, and a
 * refusal or a thrown failure becomes the problem shown to the person; a call the person cancelled shows none.
 */
import { ref } from 'vue'
import type { Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'

const isAbort = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'name' in error && error.name === 'AbortError'

export function useIdentityCall() {
  const problem = ref('')
  const isBusy = ref(false)

  const run = async <T>(call: () => Promise<Result<T, string>>): Promise<Result<T, string>> => {
    isBusy.value = true
    problem.value = ''
    try {
      const answer = await call()
      if (!answer.ok) problem.value = answer.error
      return answer
    } catch (caughtError) {
      if (isAbort(caughtError)) return { ok: false, error: '' }
      const error = formatError(caughtError)
      problem.value = error
      return { ok: false, error }
    } finally {
      isBusy.value = false
    }
  }

  return { problem, isBusy, run }
}
