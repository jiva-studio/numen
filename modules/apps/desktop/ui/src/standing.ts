/**
 * Where one plex is standing, and how it travels.
 *
 * A plex tab holds one of these, and holds nothing else: the vault is read once
 * for the whole window, and `showing.ts` tells every plex when to ask again.
 */
import { ref } from 'vue'
import type { Neighbourhood } from './core'

/** The one question a plex asks of the vault: what is around a note. */
export interface Neighbours {
  neighbourhood(path: string): Promise<Neighbourhood>
}

export type Standing = ReturnType<typeof standing>

export function standing(core: Neighbours) {
  const neighbourhood = ref<Neighbourhood | null>(null)
  /**
   * The note this plex is showing, as it asked for it.
   *
   * It is kept apart from what came back: the answer for a note that is gone
   * carries no path, and this is what the next question is asked with.
   */
  const here = ref('')
  /** What this plex could not show, in words the window puts up for it. */
  const trouble = ref('')

  /**
   * Which question is the current one. Two answers can be in flight — a click
   * while a change is being followed — and without this the slower one wins
   * whatever was asked last.
   */
  let asked = 0
  /** Whether the tab this plex stands in is still open. */
  let open = true

  async function go(path: string) {
    if (!open) return
    const mine = ++asked
    try {
      const answer = await core.neighbourhood(path)
      if (!open || mine !== asked) return
      if (!answer.focus?.path) {
        // The vault no longer holds it. What is on screen stays, and following
        // goes on, so putting the file back brings it straight back.
        trouble.value = `${path} is not in the vault`
        return
      }
      trouble.value = ''
      here.value = path
      neighbourhood.value = answer
    } catch (error) {
      if (!open || mine !== asked) return
      trouble.value = String(error)
    }
  }

  /** The tab has closed. An answer still on its way is let go of. */
  const close = () => {
    open = false
  }

  return { neighbourhood, here, trouble, go, close }
}
