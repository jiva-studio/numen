import { onBeforeUnmount, watch, type Ref } from 'vue'

/** The place the window's one media element is drawn in. */
export interface VideoSurfaceHost {
  readonly attach: (container: HTMLElement) => void
  readonly release: (container: HTMLElement) => void
}

/** Draws the media element in the container while it is shown, releases it when it is not shown, and when the view is unmounted. */
export function useVideoSurface(
  container: Readonly<Ref<HTMLElement | null>>,
  isShown: () => boolean,
  getHost: () => VideoSurfaceHost,
): void {
  watch(
    [isShown, container],
    ([shown, element]) => {
      if (!element) return
      if (shown) getHost().attach(element)
      else getHost().release(element)
    },
    { immediate: true, flush: 'post' },
  )

  onBeforeUnmount(() => {
    if (container.value) getHost().release(container.value)
  })
}
