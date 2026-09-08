/**
 * An answer a test hands over when it chooses to.
 *
 * Two questions in the air at once come back in whatever order the test says,
 * which is how what the window does with the slower of them is asked.
 */
export interface Deferred<T> {
  promise: Promise<T>
  answers: (value: T) => void
  fails: (why: string) => void
}

export function later<T>(): Deferred<T> {
  let answers!: (value: T) => void
  let fails!: (why: unknown) => void
  const promise = new Promise<T>((resolve, reject) => {
    answers = resolve
    fails = reject
  })
  return { promise, answers, fails: (why: string) => fails(new Error(why)) }
}
