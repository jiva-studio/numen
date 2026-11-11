import { nextTick, watch, type Ref } from 'vue'
import { reportFailure, runInBackground } from '@numen/ui'

/** Scrolls the line at the active index to the center of the list while the list is followed. */
export function useFollowedLine(
  list: Readonly<Ref<HTMLElement | null>>,
  getActive: () => number,
  isFollowing: () => boolean,
) {
  watch(getActive, (index) => {
    if (!isFollowing()) return
    runInBackground(
      nextTick(() =>
        list.value?.children[index]?.scrollIntoView({ block: 'center', behavior: 'smooth' }),
      ),
      reportFailure,
    )
  })
}
