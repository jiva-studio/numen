/**
 * Background execution of a promise with explicit rejection handling.
 */

/** Runs a promise without awaiting it and passes its rejection to onError. */
export function runInBackground(task: Promise<unknown>, onError: (error: unknown) => void): void {
  task.catch(onError)
}
