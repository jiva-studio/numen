import { ref } from 'vue'
import { runInBackground, win } from '@numen/ui'

const SHOWN_MS = 1500

/** Puts a text on the clipboard and says for a moment that it did. */
export function useCopyText(getText: () => string) {
  const isCopied = ref(false)

  function copy(): void {
    const clipboard = win.navigator.clipboard as Clipboard | undefined
    if (!clipboard) return
    runInBackground(
      clipboard.writeText(getText()).then(() => {
        isCopied.value = true
        win.setTimeout(() => {
          isCopied.value = false
        }, SHOWN_MS)
      }),
      () => undefined,
    )
  }

  return { isCopied, copy }
}
