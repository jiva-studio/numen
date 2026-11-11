import { nextTick, watch, type Ref } from 'vue'
import { reportFailure, runInBackground } from '@numen/ui'

/** Scrolls the thread to its end, smoothly, once it holds another line. */
export function useThreadScroll(thread: Readonly<Ref<HTMLElement | null>>, getLength: () => number) {
  watch(getLength, () => {
    runInBackground(
      nextTick(() => thread.value?.scrollTo({ top: thread.value.scrollHeight, behavior: 'smooth' })),
      reportFailure,
    )
  })
}
