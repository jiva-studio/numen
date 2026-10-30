/**
 * A promise nobody awaits.
 *
 * The caller names what receives its failure, so a task that runs in the
 * background cannot fail unseen.
 */

/** Runs a task nobody awaits and passes its failure to onError. */
export function runInBackground(task: Promise<unknown>, onError: (error: unknown) => void): void {
  task.catch(onError)
}
