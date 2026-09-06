/**
 * Following a vault that is still being read.
 *
 * A vault is scanned behind whoever opened it, so the first picture of it is of
 * a note the index does not hold yet. The core says what changed as it lands,
 * and the picture is drawn again each time.
 */
import { onScopeDispose } from 'vue'
import type { Core } from '../core'

/**
 * Draw again on every change, until the scope this was made in goes.
 *
 * The stream ends when the vault is closed, and an error is the same thing
 * seen from here: there is nothing to follow any more.
 */
export function follow(core: Core, again: () => void): void {
  const stop = new AbortController()
  onScopeDispose(() => stop.abort())

  void (async () => {
    try {
      for await (const _ of core.vault.watchVaultChanges({}, { signal: stop.signal })) {
        again()
      }
    } catch {
      // The vault is closed, or the page is going.
    }
  })()
}
