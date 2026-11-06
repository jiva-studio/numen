/**
 * The window the plugins of the editor run in: the state of its frame and the window buttons, which
 * the frame carries out.
 */
import { computed, watch } from 'vue'
import { runInBackground } from '@numen/ui'
import type { WindowHost, WindowState } from '@numen/plugin-host'
import type { WindowFrame } from './useWindowFrame'

export function createWindowHost(
  frame: WindowFrame,
  onError: (error: unknown) => void,
): WindowHost {
  const state = computed<WindowState>(() => ({
    isMaximized: frame.isMaximized.value,
    isFullscreen: frame.isFullscreen.value,
  }))
  return {
    getState: () => state.value,
    watch: (handler) => ({ dispose: watch(state, handler, { immediate: true }) }),
    minimize: () => runInBackground(frame.minimizeWindow(), onError),
    toggleMaximize: () => runInBackground(frame.toggleMaximizeWindow(), onError),
    close: () => runInBackground(frame.closeWindow(), onError),
  }
}
