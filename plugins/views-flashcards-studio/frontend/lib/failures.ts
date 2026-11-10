import type { CardsFailure, ErrorCode } from '../types'

export const getFailureCode = (code: CardsFailure): ErrorCode | null =>
  code === 'changed' ? null : code
