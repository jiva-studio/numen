/** Settles when the work does or after the wait, whichever is first; a failure of the work is not this wait's to report. */
export async function waitFor(work: Promise<void>, waitMs: number): Promise<void> {
  let timer: ReturnType<typeof setTimeout> | undefined
  const late = new Promise<void>((resolve) => {
    timer = setTimeout(resolve, waitMs)
  })
  try {
    await Promise.race([work.catch(() => undefined), late])
  } finally {
    clearTimeout(timer)
  }
}
