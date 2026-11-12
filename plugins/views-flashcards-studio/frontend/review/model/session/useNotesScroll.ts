import { nextTick, watch, type ComponentPublicInstance, type Ref } from 'vue'
import type { Neighbor } from '../../api/session/notes'
import { getNoteKey, type NotesPanelState } from './notes'

/** How much of the panel one press of space moves it. */
const STEP = 0.85

/**
 * Scrolling of the reading column: to the note a followed link named once the notes are
 * drawn, and by one page on request.
 */
export function useNotesScroll(panel: NotesPanelState, column: Readonly<Ref<HTMLElement | null>>) {
  /**
   * Where each note was drawn, so the one a link named can be scrolled to. Read
   * only when a link is followed, so it is held outside the reactive graph.
   */
  const noteElements = new Map<string, HTMLElement>()

  // A link pressed in the card opens the panel on the note it names, so reading
  // starts at that note. The watcher waits for the notes: the panel opens while they
  // are still being requested, and a note cannot be scrolled to before it is
  // rendered.
  watch(
    () => [panel.view.open.value, panel.view.col.value, panel.content.notes.value] as const,
    async ([open, col, notes]) => {
      if (!open || !col || !notes.length) return
      await nextTick(() => {
        const found = notes.find((one: Neighbor) => one.written === col || one.path === col)
        if (found)
          noteElements.get(getNoteKey(found))?.scrollIntoView({ block: 'start', behavior: 'auto' })
        panel.actions.read()
      })
    },
    { immediate: true },
  )

  const registerNoteElement = (
    one: Neighbor,
    element: Element | ComponentPublicInstance | null,
  ): void => {
    if (element instanceof HTMLElement) noteElements.set(getNoteKey(one), element)
    else noteElements.delete(getNoteKey(one))
  }

  const scrollPage = (back = false): void => {
    const col = column.value
    if (!col) return
    col.scrollBy({ top: col.clientHeight * STEP * (back ? -1 : 1), behavior: 'smooth' })
  }

  return { registerNoteElement, scrollPage }
}
