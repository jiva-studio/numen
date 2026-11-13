/**
 * The line a note tab is asked to open on, handed to its editor when there is one to take it.
 */
import { nextTick, watch } from 'vue'
import type { EditorHandle } from '@numen/plugin-sdk'

export function createLineReveal(isLoaded: () => boolean) {
  let editor: EditorHandle | null = null
  let owed: number | null = null

  const settle = () => {
    if (owed === null || !editor || !isLoaded()) return
    if (editor.reveal?.(owed)) owed = null
  }
  // The editor takes the text of the note on the render after it is read.
  const settleAfterRender = () => {
    nextTick(settle).catch(() => undefined)
  }

  watch(isLoaded, settleAfterRender, { flush: 'post' })

  return {
    attach: (attached: EditorHandle | null): void => {
      editor = attached
      settleAfterRender()
    },
    measure: (): void => editor?.measure?.(),
    reveal: (line: number): void => {
      owed = line
      settleAfterRender()
    },
  }
}
