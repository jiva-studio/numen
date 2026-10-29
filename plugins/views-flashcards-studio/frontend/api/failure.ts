/**
 * The failure of an API method, and the sentence it is reported as.
 */
import { formatErrorMessage } from '@numen/plugin-sdk'
import type { ApiError } from '@numen/plugin-sdk'

/** A method that failed: the code of the API error and the text the backend gave. */
export class ApiFailure extends Error {
  readonly code: string

  constructor(error: ApiError) {
    super(error.message)
    this.name = 'ApiFailure'
    this.code = error.code
  }
}

/** The failure of a call that the window itself stopped. */
export const CANCELED: ApiError = { code: 'canceled', message: 'canceled' }

/** The codes whose message is written for the person who waits. */
const MESSAGE_CODES: ReadonlySet<string> = new Set([
  'invalid_argument',
  'failed_precondition',
  'unavailable',
])

/** What each code says when its message is not written for the person. */
const CODE_WORDS: Readonly<Record<string, string>> = {
  invalid_argument: 'invalid request',
  not_found: 'not found',
  failed_precondition: 'not possible right now',
  unavailable: 'Numen is busy; try again',
}

/** What is said when nothing more precise can honestly be said. */
const UNEXPECTED = 'internal error'

/** The words of a caught failure; the SDK words any failure that is not an API failure. */
export const formatFailureMessage = (failure: unknown): string => {
  if (!(failure instanceof ApiFailure)) return formatErrorMessage(failure)
  if (failure.code === CANCELED.code) return ''
  const message = failure.message.trim()
  if (MESSAGE_CODES.has(failure.code) && message) return message
  return CODE_WORDS[failure.code] ?? UNEXPECTED
}
