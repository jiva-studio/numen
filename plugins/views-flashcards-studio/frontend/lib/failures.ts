import type { ErrorCode } from '../api/wire/errors'
import type { CardsFailure } from '../types'

export const getFailureCode = (code: CardsFailure): ErrorCode | null =>
  code === 'changed' ? null : code
