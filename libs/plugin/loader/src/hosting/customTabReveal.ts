/**
 * The line a note tab is requested to open on, passed to its editor when there is one to take it.
 */
import { nextTick, watch } from 'vue'
import type { EditorHandle } from '@numen/plugin-sdk'

/** How many frames a line the editor refused is offered again; the editor takes the text within a few. */
const MAX_RETRIES = 10

const requestNextFrame = (run: () => void): void => {
  if (typeof requestAnimationFrame === 'function') requestAnimationFrame(run)
  else setTimeout(run, 16)
}

export function createLineReveal(
  isLoaded: () => boolean,
  schedule: (run: () => void) => void = requestNextFrame,
) {
  let editor: EditorHandle | null = null
  let owed: number | null = null
  let retries = 0
  let generation = 0

  const settle = () => {
    if (owed === null || !editor || !isLoaded()) return
    if (editor.reveal?.(owed)) {
      owed = null
      return
    }
    // The editor has no text yet: offer the line again on a later frame instead of dropping it.
    if (retries >= MAX_RETRIES) return
    retries += 1
    const asked = generation
    schedule(() => {
      if (asked === generation) settle()
    })
  }
  // The editor takes the text of the note on the render after it is read.
  const settleAfterRender = () => {
    generation += 1
    const asked = generation
    nextTick(() => {
      if (asked === generation) settle()
    }).catch(() => undefined)
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
      retries = 0
      settleAfterRender()
    },
  }
}
