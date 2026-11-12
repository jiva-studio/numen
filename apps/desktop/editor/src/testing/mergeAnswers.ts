import type { CoreAnswers } from '@numen/core-client/testing'

/**
 * The answers of several entities as one set. The answers of a service that two sets both name are
 * merged call by call, so one set never replaces the other's calls of that service.
 */
export const mergeAnswers = (...sets: readonly CoreAnswers[]): CoreAnswers => {
  const merged: Record<string, object> = {}
  for (const set of sets) {
    for (const [service, calls] of Object.entries(set)) {
      merged[service] = { ...merged[service], ...calls }
    }
  }
  return merged
}
