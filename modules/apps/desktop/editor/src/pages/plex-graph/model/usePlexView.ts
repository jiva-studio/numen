/**
 * What one plex is showing, and how it travels.
 *
 * A plex tab holds one of these, and holds nothing else: the vault is read once
 * for the whole window, and tells every plex when to ask again.
 */
import { ref } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { answerGuard } from '@/shared/questions'
import type { Neighbourhood } from '@/entities/note'
import { getRenamedPath, type PathRename } from '@/shared/paths'
import { alike } from '../lib/picture'

/** The one question a plex asks of the vault: what is around a note. */
export interface Neighbours {
  neighbourhood(path: string): Promise<Neighbourhood>
}

export type PlexView = ReturnType<typeof usePlexView>

export function usePlexView(core: Neighbours) {
  const neighbourhood = ref<Neighbourhood | null>(null)
  /**
   * The note this plex is showing, as it asked for it.
   */
  const here = ref('')
  /** What this plex could not show, in words the window puts up for it. */
  const error = ref('')

  /** Two answers can be in flight — a click while a change is being followed. */
  const asks = answerGuard()

  async function go(path: string) {
    if (!asks.open()) return
    const mine = asks.ask()
    try {
      const answer = await core.neighbourhood(path)
      if (!mine.current) return
      if (!answer.focus.path) {
        error.value = `${path} is not in the vault`
        return
      }
      error.value = ''
      here.value = path
      if (!alike(neighbourhood.value, answer)) neighbourhood.value = answer
    } catch (thrown) {
      if (!mine.current) return
      error.value = formatErrorMessage(thrown)
    }
  }

  const followMoves = (renamed: readonly PathRename[]) => {
    const to = getRenamedPath(renamed, here.value)
    if (!to) return
    here.value = to
    asks.drop()
  }

  const close = () => {
    asks.close()
  }

  return { neighbourhood, here, error, go, followMoves, close }
}
