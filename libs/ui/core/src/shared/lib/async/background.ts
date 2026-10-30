/** Starts a promise without awaiting it and passes a rejection to onError. */
export function runInBackground(task: Promise<unknown>, onError: (error: unknown) => void): void {
  task.catch(onError)
}
