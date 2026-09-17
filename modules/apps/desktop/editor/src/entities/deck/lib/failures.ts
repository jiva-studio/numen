/**
 * What a refusal of a deck or a stencil is recorded under.
 */
import type { ErrorCode } from '@/shared/errors'
import type { CardsFailure } from '../types'

/**
 * The code a refusal is recorded under. A file that moved past the caller is
 * not a fault of the file, and carries no words of its own.
 */
export const failedWith = (code: CardsFailure): ErrorCode | null =>
  code === 'changed' ? null : code
