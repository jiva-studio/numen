import { onBeforeUnmount, watch, type WatchSource } from 'vue'

/**
 * Keeps a popover dialog named by its own `aria-label`. reka-ui labels its content by the trigger
 * and writes `aria-labelledby` again whenever the trigger id changes, which wins over `aria-label`.
 * While `isActive` holds, the attribute is removed from the element marked with
 * `data-bar-chip-popover` = `chipId` as often as it appears.
 */
export function useDialogName(chipId: string, isActive: WatchSource<boolean>): void {
  let observer: MutationObserver | null = null

  function removeLabelledby(): void {
    document
      .querySelector(`[data-bar-chip-popover="${chipId}"]`)
      ?.removeAttribute('aria-labelledby')
  }

  function stop(): void {
    observer?.disconnect()
    observer = null
  }

  watch(
    isActive,
    (isNamed) => {
      stop()
      if (!isNamed) return
      removeLabelledby()
      observer = new MutationObserver(removeLabelledby)
      observer.observe(document.body, {
        childList: true,
        subtree: true,
        attributes: true,
        attributeFilter: ['aria-labelledby'],
      })
    },
    { immediate: true, flush: 'post' },
  )

  onBeforeUnmount(stop)
}
