/**
 * The pictures of a drawn document. A picture that has not arrived takes the room of its alt text,
 * and the columns are set again when it arrives and takes its own, so the columns are read only
 * once the pictures in them have arrived.
 */

/** The longest the columns wait for the pictures of a document, in milliseconds. */
export const PICTURES_WAIT = 2_000

/** The pictures in the text that have neither arrived nor failed. */
export function findPendingImages(text: HTMLElement): HTMLImageElement[] {
  return Array.from(text.querySelectorAll('img')).filter((one) => !one.complete)
}

/**
 * Calls `run` once every picture in `pictures` has arrived or failed, or after `PICTURES_WAIT`,
 * and once only.
 */
export function whenArrived(pictures: readonly HTMLImageElement[], run: () => void): void {
  let remaining = pictures.length
  let isDone = false
  const finish = () => {
    if (isDone) return
    isDone = true
    clearTimeout(limit)
    run()
  }
  const limit = setTimeout(finish, PICTURES_WAIT)
  const onSettled = () => {
    remaining -= 1
    if (remaining <= 0) finish()
  }
  for (const one of pictures) {
    one.addEventListener('load', onSettled, { once: true })
    one.addEventListener('error', onSettled, { once: true })
  }
}
