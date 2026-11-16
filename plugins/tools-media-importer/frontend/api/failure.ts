import type { ApiError } from '@numen/plugin-sdk'
import { WORDS as words } from '../words'
import { ErrorCode } from './importer'

/** The message displayed to the user for a method failure. */
export const describeError = (error: ApiError): string => {
  switch (error.code) {
    case ErrorCode.notFound:
      return words.notALink
    case ErrorCode.failedPrecondition:
      return words.noVault
    default:
      return words.failed
  }
}
