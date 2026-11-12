/**
 * Background execution of a promise with explicit rejection handling.
 */
import type { Caught } from './types'

/** Runs a promise without awaiting it and passes its rejection to onError. */
export function runInBackground<T>(task: Promise<T>, onError: (error: Caught) => void): void {
  task.catch(onError)
}
