/**
 * Plugin activation state for EPUB book reading capability.
 */
import { ref } from 'vue'

export const isBookPluginActive = ref(true)

export function setBookPluginActive(active: boolean): void {
  isBookPluginActive.value = active
}

export function isPluginActive(): boolean {
  return isBookPluginActive.value
}
