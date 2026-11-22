/**
 * Background execution of a promise with explicit rejection handling.
 */
/** Runs a promise without awaiting it and passes its rejection to onError. */
export function runInBackground<T>(task: Promise<T>, onError: (error: unknown) => void): void {
  task.catch(onError)
}
