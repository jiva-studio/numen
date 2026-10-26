import type { ErrorCode } from '../wire/errors'
import type { CardsFailure } from '../types'

export const getFailureCode = (code: CardsFailure): ErrorCode | null =>
  code === 'changed' ? null : code
