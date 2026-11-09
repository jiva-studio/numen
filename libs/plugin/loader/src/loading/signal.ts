/** A promise and the function that settles it. */
export function createSignal(): { readonly promise: Promise<void>; readonly settle: () => void } {
  let settle: () => void = () => undefined
  const promise = new Promise<void>((resolve) => {
    settle = resolve
  })
  return { promise, settle }
}
