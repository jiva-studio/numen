/**
 * Plugin activation state for PDF document reading capability.
 */
import { ref } from 'vue'

export const isPdfPluginActive = ref(true)

export function setPdfPluginActive(active: boolean): void {
  isPdfPluginActive.value = active
}

export function isPluginActive(): boolean {
  return isPdfPluginActive.value
}
