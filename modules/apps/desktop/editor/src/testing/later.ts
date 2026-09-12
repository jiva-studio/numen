/**
 * An answer a test hands over when it chooses to.
 *
 * Two questions in the air at once come back in whatever order the test says,
 * which is how what the window does with the slower of them is asked.
 */
export interface Deferred<T> {
  promise: Promise<T>
  answer: (value: T) => void
  fail: (error: string) => void
}

export function later<T>(): Deferred<T> {
  let answer!: (value: T) => void
  let fail!: (error: unknown) => void
  const promise = new Promise<T>((resolve, reject) => {
    answer = resolve
    fail = reject
  })
  return { promise, answer, fail: (error: string) => fail(new Error(error)) }
}
