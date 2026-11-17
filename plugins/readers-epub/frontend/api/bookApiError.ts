import type { ApiError } from '@numen/plugin-sdk'

/** A method of the API that failed. */
export class BookApiError extends Error {
  readonly code: string

  constructor(error: ApiError) {
    super(error.message)
    this.name = 'BookApiError'
    this.code = error.code
  }
}
