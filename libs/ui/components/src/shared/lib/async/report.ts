/** Whatever a try block threw or a promise rejected with. */
export type Caught = unknown

/** What a function validates before it is typed: a parameter of a parseX function or a type guard. */
export type Unvalidated = unknown

/** Rethrows a rejection of a background task as an uncaught error. */
export function reportFailure(error: Caught): void {
  queueMicrotask(() => {
    throw error
  })
}
