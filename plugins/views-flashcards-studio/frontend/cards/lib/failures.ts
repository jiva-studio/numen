import type { ErrorCode } from '@numen/wire'
import type { CardsFailure } from '../types'

export const getFailureCode = (code: CardsFailure): ErrorCode | null =>
  code === 'changed' ? null : code
