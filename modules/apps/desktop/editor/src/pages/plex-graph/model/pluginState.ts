/**
 * Plugin activation state for Plexus Graph visualizer capability.
 */
import { ref } from 'vue'

export const isPlexPluginActive = ref(true)

export function setPlexPluginActive(active: boolean): void {
  isPlexPluginActive.value = active
}

export function isPluginActive(): boolean {
  return isPlexPluginActive.value
}
