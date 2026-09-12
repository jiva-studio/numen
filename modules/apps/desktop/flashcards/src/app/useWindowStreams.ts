/**
 * What the window starts when it opens and lets go of when it closes: the
 * keyboard, the counting, and the streams it is told things on.
 */
import { onMounted, onUnmounted } from 'vue'
import type { Ref } from 'vue'
import { createFollower } from '@numen/ui'
import type { Task } from '@numen/ui'

import { WINDOW, agent as agentState, cards, itself } from '@/shared/clients'
import { WORDS } from '@/pages/session'

/** What the streams ask of the window they are followed in. */
export interface WindowStreamsDeps {
  /** Trouble, in the person's own words. */
  readonly reportError: (why: unknown) => void
  /** What is being done behind the window, as cards to draw. */
  readonly setTasks: (said: readonly Task[]) => void
  /** A key pressed anywhere in the window. */
  readonly handleKey: (press: KeyboardEvent) => void
  /** Everything the window shows, counted again. */
  readonly count: () => Promise<void>
  /** The counting ended where it stands. */
  readonly stop: () => void
  /** Read again, a deck having been written or a card changed underneath. */
  readonly refresh: () => Promise<void>
  /** Why nothing can be asked here, empty while something can. */
  readonly unreachable: Ref<string>
}

export const useWindowStreams = (deps: WindowStreamsDeps) => {
  /** Whether the window is still open, which is how long anything is followed. */
  let open = true

  const follows = createFollower({
    open: () => open,
    lost: deps.reportError,
    wait: (ms) => new Promise((then) => setTimeout(then, ms)),
  })

  onMounted(() => {
    window.addEventListener('keydown', deps.handleKey)
    void deps.count()
    // Whether a card can be asked about is the window's to know before a person
    // reaches for it, so it is asked once and the way in is drawn from it.
    void agentState
      .getAgentState({})
      .then((said) => {
        deps.unreachable.value = said.unreachable
      })
      .catch(() => {
        deps.unreachable.value = WORDS.unreachable
      })
    // A deck written or a card changed underneath the window is counted again
    // without a person asking. A session is left alone: its cards were laid out
    // when it opened, and what a deck says now is read at the next one.
    void follows(
      () => cards.watchReloads({}),
      async (said) => {
        // A message carrying no reload keeps the stream open and moves nothing.
        if (!said.reload) return
        await deps.refresh()
      },
    )
    // What is being done behind the window, which is a vault read into the index.
    // It is a stream because a reading begins without the page asking for one.
    void follows(
      () => itself.watchTasks({ window: WINDOW }),
      (said) => {
        deps.setTasks(
          said.tasks.map((at) => ({
            id: at.id,
            doing: at.doing,
            about: at.about,
            error: at.error,
            isAsked: at.asked,
          })),
        )
      },
    )
  })
  onUnmounted(() => {
    open = false
    deps.stop()
    window.removeEventListener('keydown', deps.handleKey)
  })
}
