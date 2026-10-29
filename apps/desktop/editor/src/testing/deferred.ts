/**
 * A promise a test resolves or rejects at a time of its choosing.
 *
 * Two pending requests can be settled in any order the test chooses, which
 * exercises how the window handles the slower of them.
 */
export interface Deferred<T> {
  promise: Promise<T>
  answer: (value: T) => void
  fail: (error: string) => void
}

export function createDeferred<T>(): Deferred<T> {
  let answer!: (value: T) => void
  let fail!: (error: unknown) => void
  const promise = new Promise<T>((resolve, reject) => {
    answer = resolve
    fail = reject
  })
  return { promise, answer, fail: (error: string) => fail(new Error(error)) }
}
