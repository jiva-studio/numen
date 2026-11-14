import { ref } from 'vue'

/**
 * Where the layout of a window stands: whether the tabs it opens with are open, and whether the
 * window writes its open tabs to the core. A window made for a detach shows one tab and never
 * writes the tabs of the main window.
 */
export function createWindowLayout(isSecondary: boolean) {
  const isReady = ref(false)
  const isWritingTabs = ref(false)
  return {
    isReady,
    isWritingTabs,
    /**
     * Marks the layout ready and starts the write of the open tabs where the window is the main
     * one and the saved tabs were read.
     */
    markStarted: (areSavedTabsRead = true): void => {
      isReady.value = true
      isWritingTabs.value = !isSecondary && areSavedTabsRead
    },
  }
}
